package core

import "time"

// MetricsCollector is an abstraction for recording rate-limit metrics
type MetricsCollector interface {
	IncAllow()                            // increment allowed requests counter
	IncDeny()                             // increment denied requests counter
	ObserveLatency(elapsed time.Duration) // observe request processing duration
}

// NoopMetrics is a no-op implementation of MetricsCollector.
type NoopMetrics struct{}

func (_ *NoopMetrics) IncAllow()                      {}
func (_ *NoopMetrics) IncDeny()                       {}
func (_ *NoopMetrics) ObserveLatency(_ time.Duration) {}
