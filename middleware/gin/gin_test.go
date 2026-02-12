package ginmw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/gin-gonic/gin"
)

type mockLimiter struct {
	result core.Result
	err    error
}

func (m *mockLimiter) Allow(_ context.Context, _ string) (core.Result, error) {
	return m.result, m.err
}
func (m *mockLimiter) Close() error { return nil }

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimit_Allowed(t *testing.T) {
	limiter := &mockLimiter{result: core.Result{
		Allowed: true, Limit: 10, Remaining: 9, Reset: 30 * time.Second,
	}}
	r := gin.New()
	r.Use(RateLimit(limiter))
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("RateLimit-Limit") != "10" {
		t.Errorf("expected RateLimit-Limit=10, got %s", rec.Header().Get("RateLimit-Limit"))
	}
}

func TestRateLimit_Denied(t *testing.T) {
	limiter := &mockLimiter{result: core.Result{
		Allowed: false, Limit: 10, Remaining: 0,
		Reset: 30 * time.Second, RetryAfter: 15 * time.Second,
	}}
	r := gin.New()
	r.Use(RateLimit(limiter))
	r.GET("/", func(c *gin.Context) { t.Fatal("should not reach handler") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

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
	var capturedKey string
	limiter := &mockLimiter{result: core.Result{Allowed: true, Limit: 10}}
	r := gin.New()
	r.Use(RateLimit(limiter, Config{
		KeyFunc: func(c *gin.Context) string {
			key := c.GetHeader("X-API-Key")
			capturedKey = key
			return key
		},
	}))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "test-key-123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if capturedKey != "test-key-123" {
		t.Errorf("expected key 'test-key-123', got '%s'", capturedKey)
	}
}
