// Package fibermw provides a ready-to-use GoRL rate-limiting middleware for Fiber.
//
// Usage:
//
//	limiter, _ := gorl.New(core.Config{...})
//	app := fiber.New()
//	app.Use(fibermw.RateLimit(limiter))
package fibermw

import (
	"fmt"
	"math"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/gofiber/fiber/v2"
)

// Config configures the rate-limiting middleware.
type Config struct {
	// KeyFunc extracts the rate-limiting key from the Fiber context.
	// Defaults to c.IP() if nil.
	KeyFunc func(c *fiber.Ctx) string

	// ResourceFunc extracts the resource identifier from the Fiber context.
	// Used by RateLimitByResource. Defaults to the request path.
	ResourceFunc func(c *fiber.Ctx) string

	// DeniedHandler is called when a request is rate-limited.
	// Defaults to a JSON 429 response if nil.
	DeniedHandler fiber.Handler

	// ErrorHandler is called when the limiter returns an error.
	// Defaults to a JSON 500 response if nil.
	ErrorHandler func(c *fiber.Ctx, err error) error
}

// configDefault fills in zero-value fields with sensible defaults.
func configDefault(cfg ...Config) Config {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.KeyFunc == nil {
		c.KeyFunc = func(ctx *fiber.Ctx) string {
			return ctx.IP()
		}
	}
	if c.ResourceFunc == nil {
		c.ResourceFunc = func(ctx *fiber.Ctx) string {
			return ctx.Path()
		}
	}
	return c
}

// RateLimit returns a Fiber middleware that applies rate limiting.
//
// It extracts a key per request, calls limiter.Allow, sets standard
// RateLimit-* headers, and either passes the request through or
// returns 429 Too Many Requests.
func RateLimit(limiter core.Limiter, cfg ...Config) fiber.Handler {
	c := configDefault(cfg...)

	return func(ctx *fiber.Ctx) error {
		key := c.KeyFunc(ctx)

		res, err := limiter.Allow(ctx.UserContext(), key)
		if err != nil {
			if c.ErrorHandler != nil {
				return c.ErrorHandler(ctx, err)
			}
			return ctx.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "internal server error"})
		}

		// Set standard rate-limit headers
		setHeaders(ctx, res)

		if !res.Allowed {
			if c.DeniedHandler != nil {
				return c.DeniedHandler(ctx)
			}
			return ctx.Status(fiber.StatusTooManyRequests).
				JSON(fiber.Map{
					"error":       "rate limit exceeded",
					"retry_after": fmt.Sprintf("%.0fs", res.RetryAfter.Seconds()),
				})
		}

		return ctx.Next()
	}
}

// RateLimitByResource returns a Fiber middleware that applies resource-scoped rate limiting.
func RateLimitByResource(limiter core.ResourceLimiter, cfg ...Config) fiber.Handler {
	c := configDefault(cfg...)

	return func(ctx *fiber.Ctx) error {
		key := c.KeyFunc(ctx)
		resource := c.ResourceFunc(ctx)

		res, err := limiter.AllowResource(ctx.UserContext(), resource, key)
		if err != nil {
			if c.ErrorHandler != nil {
				return c.ErrorHandler(ctx, err)
			}
			return ctx.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "internal server error"})
		}

		setHeaders(ctx, res)

		if !res.Allowed {
			if c.DeniedHandler != nil {
				return c.DeniedHandler(ctx)
			}
			return ctx.Status(fiber.StatusTooManyRequests).
				JSON(fiber.Map{
					"error":       "rate limit exceeded",
					"retry_after": fmt.Sprintf("%.0fs", res.RetryAfter.Seconds()),
				})
		}

		return ctx.Next()
	}
}

// setHeaders writes standard rate-limit headers to the Fiber response.
func setHeaders(c *fiber.Ctx, res core.Result) {
	c.Set("RateLimit-Limit", fmt.Sprintf("%d", res.Limit))
	c.Set("RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
	if res.Reset > 0 {
		c.Set("RateLimit-Reset", fmt.Sprintf("%d", int(math.Ceil(res.Reset.Seconds()))))
	}
	if !res.Allowed && res.RetryAfter > 0 {
		c.Set("Retry-After", fmt.Sprintf("%d", int(math.Ceil(res.RetryAfter.Seconds()))))
	}
}
