package algorithms

import (
	"fmt"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
)

// TestFailOpenHandler_NoError verifies that the failOpenHandler helper returns 'false' for done
// when there is no error, allowing normal flow to continue.
func TestFailOpenHandler_NoError(t *testing.T) {
	m := &core.NoopMetrics{}
	_, _, done := failOpenHandler(time.Now(), nil, true, m, 10)
	if done {
		t.Fatal("should not be done when no error")
	}
}

// TestFailOpenHandler_ErrorFailOpen validates that when an error occurs and FailOpen is true,
// the handler returns 'true' for done, allowing the request, and suppressing the error.
func TestFailOpenHandler_ErrorFailOpen(t *testing.T) {
	m := &mockMetrics{}
	res, err, done := failOpenHandler(time.Now(), fmt.Errorf("storage error"), true, m, 10)
	if !done {
		t.Fatal("should be done")
	}
	if !res.Allowed {
		t.Fatal("fail-open should allow")
	}
	if err != nil {
		t.Fatal("fail-open should not return error")
	}
	if m.allows != 1 {
		t.Error("should record allow metric")
	}
}

// TestFailOpenHandler_ErrorFailClosed ensures that when an error occurs and FailOpen is false,
// the handler returns 'true' for done, denying the request, and returning the error.
func TestFailOpenHandler_ErrorFailClosed(t *testing.T) {
	m := &core.NoopMetrics{}
	res, err, done := failOpenHandler(time.Now(), fmt.Errorf("storage error"), false, m, 10)
	if !done {
		t.Fatal("should be done")
	}
	if res.Allowed {
		t.Fatal("fail-closed should deny")
	}
	if err == nil {
		t.Fatal("fail-closed should return error")
	}
}
