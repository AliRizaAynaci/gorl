package metrics

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPrometheusCollector(t *testing.T) {
	pm := NewPrometheusCollector("test_ns", "test_sub")
	if pm == nil {
		t.Fatal("expected non-nil collector")
	}
	if pm.allow == nil || pm.deny == nil || pm.latency == nil {
		t.Fatal("all metrics should be initialized")
	}
}

func TestPromMetrics_IncAllow(t *testing.T) {
	pm := NewPrometheusCollector("test_allow", "sub")
	pm.IncAllow()
	// No panic means success - Prometheus counter incremented
}

func TestPromMetrics_IncDeny(t *testing.T) {
	pm := NewPrometheusCollector("test_deny", "sub")
	pm.IncDeny()
}

func TestPromMetrics_ObserveLatency(t *testing.T) {
	pm := NewPrometheusCollector("test_latency", "sub")
	pm.ObserveLatency(100 * time.Millisecond)
}

func TestPromMetrics_ImplementsInterface(t *testing.T) {
	var _ core.MetricsCollector = (*PromMetrics)(nil)
}

func TestRegisterPrometheusCollectors(t *testing.T) {
	// Use a custom registry to avoid conflicts with the default registry
	reg := prometheus.NewRegistry()
	pm := NewPrometheusCollector("test_reg", "sub")

	// Register manually with our custom registry to test the collectors work
	err := reg.Register(pm.allow)
	if err != nil {
		t.Fatalf("failed to register allow counter: %v", err)
	}
	err = reg.Register(pm.deny)
	if err != nil {
		t.Fatalf("failed to register deny counter: %v", err)
	}
	err = reg.Register(pm.latency)
	if err != nil {
		t.Fatalf("failed to register latency histogram: %v", err)
	}

	// Verify metrics are gathered
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	if len(mfs) != 3 {
		t.Fatalf("expected 3 metric families, got %d", len(mfs))
	}
}
