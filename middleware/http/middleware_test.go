package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
)

// --- Mock Limiter ---

type mockLimiter struct {
	result core.Result
	err    error
}

func (m *mockLimiter) Allow(_ context.Context, _ string) (core.Result, error) {
	return m.result, m.err
}

func (m *mockLimiter) Close() error { return nil }

// --- Tests ---

func TestRateLimit_Allowed(t *testing.T) {
	limiter := &mockLimiter{result: core.Result{
		Allowed: true, Limit: 10, Remaining: 9,
		Reset: 30 * time.Second,
	}}

	handler := RateLimit(limiter, Options{KeyFunc: KeyByIP()},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}),
	)

	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("RateLimit-Limit") != "10" {
		t.Errorf("expected RateLimit-Limit=10, got %s", rec.Header().Get("RateLimit-Limit"))
	}
	if rec.Header().Get("RateLimit-Remaining") != "9" {
		t.Errorf("expected RateLimit-Remaining=9, got %s", rec.Header().Get("RateLimit-Remaining"))
	}
}

func TestRateLimit_Denied(t *testing.T) {
	limiter := &mockLimiter{result: core.Result{
		Allowed: false, Limit: 10, Remaining: 0,
		Reset: 30 * time.Second, RetryAfter: 15 * time.Second,
	}}

	handler := RateLimit(limiter, Options{KeyFunc: KeyByIP()},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called when denied")
		}),
	)

	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "15" {
		t.Errorf("expected Retry-After=15, got %s", rec.Header().Get("Retry-After"))
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["error"] != "rate limit exceeded" {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestRateLimit_LimiterError(t *testing.T) {
	limiter := &mockLimiter{err: core.ErrBackendUnavailable}

	handler := RateLimit(limiter, Options{KeyFunc: KeyByIP()},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called on error")
		}),
	)

	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestRateLimit_CustomDeniedHandler(t *testing.T) {
	limiter := &mockLimiter{result: core.Result{Allowed: false, Limit: 5}}

	customCalled := false
	handler := RateLimit(limiter, Options{
		KeyFunc: KeyByIP(),
		OnDenied: func(w http.ResponseWriter, r *http.Request, res core.Result) {
			customCalled = true
			w.WriteHeader(http.StatusServiceUnavailable)
		},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !customCalled {
		t.Fatal("custom denied handler was not called")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestKeyByIP_XForwardedFor(t *testing.T) {
	kf := KeyByIP()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")

	key := kf(req)
	if key != "203.0.113.50" {
		t.Errorf("expected first IP from X-Forwarded-For, got %s", key)
	}
}

func TestKeyByIP_XRealIp(t *testing.T) {
	kf := KeyByIP()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-Ip", "10.0.0.42")

	key := kf(req)
	if key != "10.0.0.42" {
		t.Errorf("expected X-Real-Ip value, got %s", key)
	}
}

func TestKeyByIP_RemoteAddr(t *testing.T) {
	kf := KeyByIP()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	key := kf(req)
	if key != "192.168.1.100" {
		t.Errorf("expected IP from RemoteAddr, got %s", key)
	}
}

func TestKeyByHeader(t *testing.T) {
	kf := KeyByHeader("X-API-Key")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "my-secret-key")

	key := kf(req)
	if key != "my-secret-key" {
		t.Errorf("expected header value, got %s", key)
	}
}

func TestKeyByPath(t *testing.T) {
	kf := KeyByPath()
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	key := kf(req)
	if key != "10.0.0.1:/api/users" {
		t.Errorf("expected ip:path key, got %s", key)
	}
}

func TestNewMiddleware_Chaining(t *testing.T) {
	limiter := &mockLimiter{result: core.Result{Allowed: true, Limit: 100, Remaining: 99}}

	rl := NewMiddleware(limiter, Options{KeyFunc: KeyByIP()})
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	handler := rl(inner)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.1.1.1:80"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "hello" {
		t.Errorf("expected 'hello', got %s", rec.Body.String())
	}
}

func TestRateLimit_HeadersDisabled(t *testing.T) {
	limiter := &mockLimiter{result: core.Result{Allowed: true, Limit: 10, Remaining: 9}}

	noHeaders := false
	handler := RateLimit(limiter, Options{
		KeyFunc:    KeyByIP(),
		SetHeaders: &noHeaders,
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:80"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("RateLimit-Limit") != "" {
		t.Error("headers should not be set when disabled")
	}
}

func TestRateLimit_OmitsZeroDurationHeaders(t *testing.T) {
	limiter := &mockLimiter{result: core.Result{
		Allowed: false, Limit: 10, Remaining: 0,
	}}

	handler := RateLimit(limiter, Options{KeyFunc: KeyByIP()},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called when denied")
		}),
	)

	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("RateLimit-Reset") != "" {
		t.Fatalf("expected RateLimit-Reset to be omitted, got %q", rec.Header().Get("RateLimit-Reset"))
	}
	if rec.Header().Get("Retry-After") != "" {
		t.Fatalf("expected Retry-After to be omitted, got %q", rec.Header().Get("Retry-After"))
	}
}
