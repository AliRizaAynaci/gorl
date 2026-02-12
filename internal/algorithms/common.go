// algorithms/common.go
package algorithms

import (
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
)

// failOpenHandler centralizes fail-open logic.
//   - start: timestamp when Allow began (for latency metrics)
//   - err: storage/algorithm error
//   - failOpen: cfg.FailOpen flag
//   - m: metrics collector
//
// Returns (result, done):
//   - done=true: caller should return immediately with (result, retErr)
//   - done=false: no error, continue normal flow
func failOpenHandler(start time.Time, err error, failOpen bool, m core.MetricsCollector, limit int) (core.Result, error, bool) {
	if err == nil {
		return core.Result{}, nil, false
	}
	if failOpen {
		m.ObserveLatency(time.Since(start))
		m.IncAllow()
		return core.Result{Allowed: true, Limit: limit}, nil, true
	}
	return core.Result{Allowed: false, Limit: limit}, err, true
}
