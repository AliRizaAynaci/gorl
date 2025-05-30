// algorithms/common.go
package algorithms

import (
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

// failOpenHandler centralizes fail-open logic.
//   - start: timestamp when Allow began (for latency metrics)
//   - err: storage/algorithm error
//   - failOpen: cfg.FailOpen flag
//   - m: metrics collector
//
// Returns (allowed, retErr, done):
//   - done=true: caller should return immediately with (allowed, retErr)
//   - done=false: no error, continue normal flow
func failOpenHandler(start time.Time, err error, failOpen bool, m core.MetricsCollector) (bool, error, bool) {
	if err == nil {
		return false, nil, false
	}
	if failOpen {
		m.ObserveLatency(time.Since(start))
		m.IncAllow()
		return true, nil, true
	}
	return false, err, true
}
