package echomw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/labstack/echo/v4"
)

type mockLimiter struct {
	result core.Result
	err    error
}

func (m *mockLimiter) Allow(_ context.Context, _ string) (core.Result, error) {
	return m.result, m.err
}
func (m *mockLimiter) Close() error { return nil }

func TestRateLimit_Allowed(t *testing.T) {
	e := echo.New()
	limiter := &mockLimiter{result: core.Result{
		Allowed: true, Limit: 10, Remaining: 9, Reset: 30 * time.Second,
	}}

	handler := RateLimit(limiter)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("RateLimit-Limit") != "10" {
		t.Errorf("expected RateLimit-Limit=10, got %s", rec.Header().Get("RateLimit-Limit"))
	}
}

func TestRateLimit_Denied(t *testing.T) {
	e := echo.New()
	limiter := &mockLimiter{result: core.Result{
		Allowed: false, Limit: 10, Remaining: 0,
		Reset: 30 * time.Second, RetryAfter: 15 * time.Second,
	}}

	handler := RateLimit(limiter)(func(c echo.Context) error {
		t.Fatal("should not reach handler")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "15" {
		t.Errorf("expected Retry-After=15, got %s", rec.Header().Get("Retry-After"))
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["error"] != "rate limit exceeded" {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestRateLimit_CustomKeyFunc(t *testing.T) {
	e := echo.New()
	var capturedKey string
	limiter := &mockLimiter{result: core.Result{Allowed: true, Limit: 10}}

	handler := RateLimit(limiter, Config{
		KeyFunc: func(c echo.Context) string {
			key := c.Request().Header.Get("X-API-Key")
			capturedKey = key
			return key
		},
	})(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "test-key-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler(c)

	if capturedKey != "test-key-123" {
		t.Errorf("expected key 'test-key-123', got '%s'", capturedKey)
	}
}

func TestRateLimit_OmitsZeroDurationHeaders(t *testing.T) {
	e := echo.New()
	limiter := &mockLimiter{result: core.Result{
		Allowed: false, Limit: 10, Remaining: 0,
	}}

	handler := RateLimit(limiter)(func(c echo.Context) error {
		t.Fatal("should not reach handler")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatal(err)
	}

	if rec.Header().Get("RateLimit-Reset") != "" {
		t.Fatalf("expected RateLimit-Reset to be omitted, got %q", rec.Header().Get("RateLimit-Reset"))
	}
	if rec.Header().Get("Retry-After") != "" {
		t.Fatalf("expected Retry-After to be omitted, got %q", rec.Header().Get("Retry-After"))
	}
}
