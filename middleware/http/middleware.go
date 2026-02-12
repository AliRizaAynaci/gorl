// Package middleware provides HTTP middleware for integrating GoRL rate limiting
// into standard net/http applications.
//
// Usage:
//
//	limiter, _ := gorl.New(core.Config{...})
//	mux := http.NewServeMux()
//	mux.Handle("/api/", middleware.RateLimit(limiter, middleware.Options{
//	    KeyFunc: middleware.KeyByIP,
//	}))
//	http.ListenAndServe(":8080", mux)
package middleware

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"

	"github.com/AliRizaAynaci/gorl/v2/core"
)

// KeyFunc extracts a rate-limiting key from an HTTP request.
type KeyFunc func(r *http.Request) string

// DeniedHandler is called when a request is rate-limited.
// If nil, a default 429 response is sent.
type DeniedHandler func(w http.ResponseWriter, r *http.Request, result core.Result)

// ErrorHandler is called when the limiter returns an error.
// If nil, a default 500 response is sent.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// Options configures the rate-limiting middleware behavior.
type Options struct {
	// KeyFunc extracts the rate-limiting key from the request.
	// Required. Use KeyByIP, KeyByHeader, or a custom function.
	KeyFunc KeyFunc

	// OnDenied is called when a request exceeds the rate limit.
	// If nil, a default JSON 429 response is returned.
	OnDenied DeniedHandler

	// OnError is called when the limiter encounters an internal error.
	// If nil, a default 500 response is returned.
	OnError ErrorHandler

	// SetHeaders controls whether standard rate-limit headers are added to every response.
	// Defaults to true.
	SetHeaders *bool
}

// shouldSetHeaders returns whether rate-limit headers should be added.
func (o Options) shouldSetHeaders() bool {
	if o.SetHeaders == nil {
		return true
	}
	return *o.SetHeaders
}

// --- Built-in Key Extractors ---

// KeyByIP returns a KeyFunc that extracts the client IP address.
// It checks X-Forwarded-For and X-Real-Ip headers before falling back to RemoteAddr.
func KeyByIP() KeyFunc {
	return func(r *http.Request) string {
		// Check X-Forwarded-For first (proxied requests)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first (client) IP
			if idx := strings.IndexByte(xff, ','); idx != -1 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}

		// Check X-Real-Ip
		if xri := r.Header.Get("X-Real-Ip"); xri != "" {
			return xri
		}

		// Fall back to RemoteAddr (strip port)
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return ip
	}
}

// KeyByHeader returns a KeyFunc that extracts the key from a specific request header.
// Useful for API key or token-based rate limiting.
func KeyByHeader(header string) KeyFunc {
	return func(r *http.Request) string {
		return r.Header.Get(header)
	}
}

// KeyByPath returns a KeyFunc that combines client IP with the request path,
// enabling per-endpoint rate limiting.
func KeyByPath() KeyFunc {
	ipFunc := KeyByIP()
	return func(r *http.Request) string {
		return ipFunc(r) + ":" + r.URL.Path
	}
}

// --- Middleware ---

// RateLimit returns an http.Handler middleware that applies rate limiting.
//
// For every incoming request, it extracts a key using opts.KeyFunc,
// calls limiter.Allow, and either passes the request to the next handler
// or returns a 429 Too Many Requests response.
//
// Standard rate-limit headers (RateLimit-Limit, RateLimit-Remaining,
// RateLimit-Reset, Retry-After) are set on all responses by default.
func RateLimit(limiter core.Limiter, opts Options, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := opts.KeyFunc(r)

		res, err := limiter.Allow(r.Context(), key)
		if err != nil {
			if opts.OnError != nil {
				opts.OnError(w, r, err)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Set rate-limit headers on all responses (allowed or denied)
		if opts.shouldSetHeaders() {
			setRateLimitHeaders(w, res)
		}

		if !res.Allowed {
			if opts.OnDenied != nil {
				opts.OnDenied(w, r, res)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate limit exceeded","retry_after":"%.0fs"}`, res.RetryAfter.Seconds())
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimitFunc is a convenience wrapper that accepts an http.HandlerFunc instead of http.Handler.
func RateLimitFunc(limiter core.Limiter, opts Options, next http.HandlerFunc) http.Handler {
	return RateLimit(limiter, opts, next)
}

// NewMiddleware returns a function that can be used to wrap handlers,
// useful for chaining middleware in frameworks.
//
//	rl := middleware.NewMiddleware(limiter, opts)
//	http.Handle("/api", rl(myHandler))
func NewMiddleware(limiter core.Limiter, opts Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RateLimit(limiter, opts, next)
	}
}

// --- Helpers ---

// setRateLimitHeaders writes standard rate-limit headers to the response.
func setRateLimitHeaders(w http.ResponseWriter, res core.Result) {
	w.Header().Set("RateLimit-Limit", fmt.Sprintf("%d", res.Limit))
	w.Header().Set("RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))

	resetSec := int(math.Ceil(res.Reset.Seconds()))
	w.Header().Set("RateLimit-Reset", fmt.Sprintf("%d", resetSec))

	if !res.Allowed && res.RetryAfter > 0 {
		retrySec := int(math.Ceil(res.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retrySec))
	}
}

// WithContext is a helper KeyFunc wrapper that injects a custom context value
// before extracting the key, useful for tracing or tenant isolation.
func WithContext(key string, val interface{}, inner KeyFunc) KeyFunc {
	return func(r *http.Request) string {
		ctx := context.WithValue(r.Context(), key, val)
		r2 := r.WithContext(ctx)
		return inner(r2)
	}
}
