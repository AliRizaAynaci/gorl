package metrics

import (
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/prometheus/client_golang/prometheus"
)

// PromMetrics is an adapter to expose rate limiter metrics to Prometheus.
type PromMetrics struct {
	allow   prometheus.Counter
	deny    prometheus.Counter
	latency prometheus.Histogram
}

// NewPrometheusCollector creates a PromMetrics instance with the specified namespace and subsystem.
func NewPrometheusCollector(namespace, subsystem string) *PromMetrics {
	return &PromMetrics{
		allow: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "allow_total",
			Help:      "Total number of allowed requests",
		}),
		deny: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "deny_total",
			Help:      "Total number of denied requests",
		}),
		latency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "Histogram of request processing durations",
			// Adjust buckets to fit expected latency distribution as needed
		}),
	}
}

// RegisterPrometheusCollectors  registers the PromMetrics collectors with the default Prometheus registry.
func RegisterPrometheusCollectors(m *PromMetrics) {
	prometheus.MustRegister(m.allow, m.deny, m.latency)
}

// IncAllow increments the allowed requests counter.
func (m *PromMetrics) IncAllow() {
	m.allow.Inc()
}

// IncDeny increments the denied requests counter.
func (m *PromMetrics) IncDeny() {
	m.deny.Inc()
}

// ObserveLatency observes a request processing duration in seconds.
func (m *PromMetrics) ObserveLatency(d time.Duration) {
	m.latency.Observe(d.Seconds())
}

// Ensure PromMetrics implements core.MetricsCollector.
var _ core.MetricsCollector = (*PromMetrics)(nil)
