// Package echomw provides a ready-to-use GoRL rate-limiting middleware for Echo.
//
// Usage:
//
//	limiter, _ := gorl.New(core.Config{...})
//	e := echo.New()
//	e.Use(echomw.RateLimit(limiter))
package echomw

import (
	"fmt"
	"math"
	"net/http"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/labstack/echo/v4"
)

// Config configures the rate-limiting middleware.
type Config struct {
	// KeyFunc extracts the rate-limiting key from the Echo context.
	// Defaults to c.RealIP() if nil.
	KeyFunc func(c echo.Context) string

	// DeniedHandler is called when a request is rate-limited.
	// Defaults to a JSON 429 response if nil.
	DeniedHandler echo.HandlerFunc

	// ErrorHandler is called when the limiter returns an error.
	// Defaults to a JSON 500 response if nil.
	ErrorHandler func(c echo.Context, err error) error
}

// configDefault fills in zero-value fields with sensible defaults.
func configDefault(cfg ...Config) Config {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.KeyFunc == nil {
		c.KeyFunc = func(ctx echo.Context) string {
			return ctx.RealIP()
		}
	}
	return c
}

// RateLimit returns an Echo middleware that applies rate limiting.
//
// It extracts a key per request, calls limiter.Allow, sets standard
// RateLimit-* headers, and either passes the request through or
// returns 429 Too Many Requests.
func RateLimit(limiter core.Limiter, cfg ...Config) echo.MiddlewareFunc {
	c := configDefault(cfg...)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			key := c.KeyFunc(ctx)

			res, err := limiter.Allow(ctx.Request().Context(), key)
			if err != nil {
				if c.ErrorHandler != nil {
					return c.ErrorHandler(ctx, err)
				}
				return ctx.JSON(http.StatusInternalServerError,
					map[string]string{"error": "internal server error"})
			}

			// Set standard rate-limit headers
			setHeaders(ctx, res)

			if !res.Allowed {
				if c.DeniedHandler != nil {
					return c.DeniedHandler(ctx)
				}
				return ctx.JSON(http.StatusTooManyRequests, map[string]string{
					"error":       "rate limit exceeded",
					"retry_after": fmt.Sprintf("%.0fs", res.RetryAfter.Seconds()),
				})
			}

			return next(ctx)
		}
	}
}

// setHeaders writes standard rate-limit headers to the Echo response.
func setHeaders(c echo.Context, res core.Result) {
	h := c.Response().Header()
	h.Set("RateLimit-Limit", fmt.Sprintf("%d", res.Limit))
	h.Set("RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
	h.Set("RateLimit-Reset", fmt.Sprintf("%d", int(math.Ceil(res.Reset.Seconds()))))
	if !res.Allowed && res.RetryAfter > 0 {
		h.Set("Retry-After", fmt.Sprintf("%d", int(math.Ceil(res.RetryAfter.Seconds()))))
	}
}
