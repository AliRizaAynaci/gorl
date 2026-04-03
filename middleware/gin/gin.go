// Package ginmw provides a ready-to-use GoRL rate-limiting middleware for Gin.
//
// Usage:
//
//	limiter, _ := gorl.New(core.Config{...})
//	r := gin.Default()
//	r.Use(ginmw.RateLimit(limiter))
package ginmw

import (
	"fmt"
	"math"
	"net/http"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/gin-gonic/gin"
)

// Config configures the rate-limiting middleware.
type Config struct {
	// KeyFunc extracts the rate-limiting key from the Gin context.
	// Defaults to c.ClientIP() if nil.
	KeyFunc func(c *gin.Context) string

	// DeniedHandler is called when a request is rate-limited.
	// Defaults to a JSON 429 response if nil.
	DeniedHandler gin.HandlerFunc

	// ErrorHandler is called when the limiter returns an error.
	// Defaults to a JSON 500 response if nil.
	ErrorHandler func(c *gin.Context, err error)
}

// configDefault fills in zero-value fields with sensible defaults.
func configDefault(cfg ...Config) Config {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.KeyFunc == nil {
		c.KeyFunc = func(ctx *gin.Context) string {
			return ctx.ClientIP()
		}
	}
	return c
}

// RateLimit returns a Gin middleware that applies rate limiting.
//
// It extracts a key per request, calls limiter.Allow, sets standard
// RateLimit-* headers, and either passes the request through or
// returns 429 Too Many Requests.
func RateLimit(limiter core.Limiter, cfg ...Config) gin.HandlerFunc {
	c := configDefault(cfg...)

	return func(ctx *gin.Context) {
		key := c.KeyFunc(ctx)

		res, err := limiter.Allow(ctx.Request.Context(), key)
		if err != nil {
			if c.ErrorHandler != nil {
				c.ErrorHandler(ctx, err)
				ctx.Abort()
				return
			}
			ctx.AbortWithStatusJSON(http.StatusInternalServerError,
				gin.H{"error": "internal server error"})
			return
		}

		// Set standard rate-limit headers
		setHeaders(ctx, res)

		if !res.Allowed {
			if c.DeniedHandler != nil {
				c.DeniedHandler(ctx)
				ctx.Abort()
				return
			}
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": fmt.Sprintf("%.0fs", res.RetryAfter.Seconds()),
			})
			return
		}

		ctx.Next()
	}
}

// setHeaders writes standard rate-limit headers to the Gin response.
func setHeaders(c *gin.Context, res core.Result) {
	c.Header("RateLimit-Limit", fmt.Sprintf("%d", res.Limit))
	c.Header("RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
	if res.Reset > 0 {
		c.Header("RateLimit-Reset", fmt.Sprintf("%d", int(math.Ceil(res.Reset.Seconds()))))
	}
	if !res.Allowed && res.RetryAfter > 0 {
		c.Header("Retry-After", fmt.Sprintf("%d", int(math.Ceil(res.RetryAfter.Seconds()))))
	}
}
