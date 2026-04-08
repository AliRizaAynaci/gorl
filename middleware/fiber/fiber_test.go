package fibermw

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/gofiber/fiber/v2"
)

type mockLimiter struct {
	result core.Result
	err    error
}

func (m *mockLimiter) Allow(_ context.Context, _ string) (core.Result, error) {
	return m.result, m.err
}
func (m *mockLimiter) Close() error { return nil }

type mockResourceLimiter struct {
	result   core.Result
	err      error
	resource string
	key      string
}

func (m *mockResourceLimiter) AllowResource(_ context.Context, resource, key string) (core.Result, error) {
	m.resource = resource
	m.key = key
	return m.result, m.err
}
func (m *mockResourceLimiter) Close() error { return nil }

func TestRateLimit_Allowed(t *testing.T) {
	app := fiber.New()
	limiter := &mockLimiter{result: core.Result{
		Allowed: true, Limit: 10, Remaining: 9, Reset: 30 * time.Second,
	}}
	app.Use(RateLimit(limiter))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("RateLimit-Limit") != "10" {
		t.Errorf("expected RateLimit-Limit=10, got %s", resp.Header.Get("RateLimit-Limit"))
	}
}

func TestRateLimit_Denied(t *testing.T) {
	app := fiber.New()
	limiter := &mockLimiter{result: core.Result{
		Allowed: false, Limit: 10, Remaining: 0,
		Reset: 30 * time.Second, RetryAfter: 15 * time.Second,
	}}
	app.Use(RateLimit(limiter))
	app.Get("/", func(c *fiber.Ctx) error {
		t.Fatal("should not reach handler")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") != "15" {
		t.Errorf("expected Retry-After=15, got %s", resp.Header.Get("Retry-After"))
	}

	body, _ := io.ReadAll(resp.Body)
	var m map[string]string
	json.Unmarshal(body, &m)
	if m["error"] != "rate limit exceeded" {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestRateLimit_CustomKeyFunc(t *testing.T) {
	app := fiber.New()
	var capturedKey string
	limiter := &mockLimiter{result: core.Result{Allowed: true, Limit: 10}}

	app.Use(RateLimit(limiter, Config{
		KeyFunc: func(c *fiber.Ctx) string {
			key := c.Get("X-API-Key")
			capturedKey = key
			return key
		},
	}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "test-key-123")
	app.Test(req)

	if capturedKey != "test-key-123" {
		t.Errorf("expected key 'test-key-123', got '%s'", capturedKey)
	}
}

func TestRateLimit_OmitsZeroDurationHeaders(t *testing.T) {
	app := fiber.New()
	limiter := &mockLimiter{result: core.Result{
		Allowed: false, Limit: 10, Remaining: 0,
	}}
	app.Use(RateLimit(limiter))
	app.Get("/", func(c *fiber.Ctx) error {
		t.Fatal("should not reach handler")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Header.Get("RateLimit-Reset") != "" {
		t.Fatalf("expected RateLimit-Reset to be omitted, got %q", resp.Header.Get("RateLimit-Reset"))
	}
	if resp.Header.Get("Retry-After") != "" {
		t.Fatalf("expected Retry-After to be omitted, got %q", resp.Header.Get("Retry-After"))
	}
}

func TestRateLimitByResource_UsesRequestPath(t *testing.T) {
	app := fiber.New()
	limiter := &mockResourceLimiter{result: core.Result{Allowed: true, Limit: 10, Remaining: 9}}

	app.Use(RateLimitByResource(limiter))
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if limiter.resource != "/users/42" {
		t.Fatalf("expected resource /users/42, got %q", limiter.resource)
	}
	if limiter.key == "" {
		t.Fatal("expected key to be populated")
	}
}
