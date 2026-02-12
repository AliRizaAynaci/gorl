package algorithms

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/storage"
)

// mockMetrics records calls for assertion.
type mockMetrics struct {
	allows    int
	denies    int
	latencies int
}

func (m *mockMetrics) IncAllow()                      { m.allows++ }
func (m *mockMetrics) IncDeny()                       { m.denies++ }
func (m *mockMetrics) ObserveLatency(_ time.Duration) { m.latencies++ }

// failingStore always returns an error on every operation.
type failingStore struct{}

func (s *failingStore) Incr(_ context.Context, _ string, _ time.Duration) (float64, error) {
	return 0, fmt.Errorf("store unavailable")
}
func (s *failingStore) Get(_ context.Context, _ string) (float64, error) {
	return 0, fmt.Errorf("store unavailable")
}
func (s *failingStore) Set(_ context.Context, _ string, _ float64, _ time.Duration) error {
	return fmt.Errorf("store unavailable")
}
func (s *failingStore) Close() error { return nil }

// Ensure failingStore implements storage.Storage.
var _ storage.Storage = (*failingStore)(nil)

// failOnSetStore fails only on Set, succeeds on Get and Incr.
type failOnSetStore struct {
	data map[string]float64
}

func newFailOnSetStore() *failOnSetStore {
	return &failOnSetStore{data: make(map[string]float64)}
}

func (s *failOnSetStore) Incr(_ context.Context, key string, _ time.Duration) (float64, error) {
	s.data[key]++
	return s.data[key], nil
}
func (s *failOnSetStore) Get(_ context.Context, key string) (float64, error) {
	return s.data[key], nil
}
func (s *failOnSetStore) Set(_ context.Context, _ string, _ float64, _ time.Duration) error {
	return fmt.Errorf("set failed")
}
func (s *failOnSetStore) Close() error { return nil }

var _ storage.Storage = (*failOnSetStore)(nil)

// setFailAfterNStore succeeds for the first N Set calls, then fails.
type setFailAfterNStore struct {
	data      map[string]float64
	failAfter int
	setCalls  int
}

func (s *setFailAfterNStore) Incr(_ context.Context, key string, _ time.Duration) (float64, error) {
	s.data[key]++
	return s.data[key], nil
}
func (s *setFailAfterNStore) Get(_ context.Context, key string) (float64, error) {
	return s.data[key], nil
}
func (s *setFailAfterNStore) Set(_ context.Context, key string, val float64, _ time.Duration) error {
	s.setCalls++
	if s.setCalls > s.failAfter {
		return fmt.Errorf("set failed after %d calls", s.failAfter)
	}
	s.data[key] = val
	return nil
}
func (s *setFailAfterNStore) Close() error { return nil }

var _ storage.Storage = (*setFailAfterNStore)(nil)

// incrFailStore fails on Incr but succeeds on Get/Set.
type incrFailStore struct {
	data map[string]float64
}

func (s *incrFailStore) Incr(_ context.Context, _ string, _ time.Duration) (float64, error) {
	return 0, fmt.Errorf("incr failed")
}
func (s *incrFailStore) Get(_ context.Context, key string) (float64, error) {
	return s.data[key], nil
}
func (s *incrFailStore) Set(_ context.Context, key string, val float64, _ time.Duration) error {
	s.data[key] = val
	return nil
}
func (s *incrFailStore) Close() error { return nil }

var _ storage.Storage = (*incrFailStore)(nil)

// getFailAfterNStore succeeds for the first N Get calls, then fails.
type getFailAfterNStore struct {
	data      map[string]float64
	failAfter int
	getCalls  int
}

func (s *getFailAfterNStore) Incr(_ context.Context, key string, _ time.Duration) (float64, error) {
	s.data[key]++
	return s.data[key], nil
}
func (s *getFailAfterNStore) Get(_ context.Context, key string) (float64, error) {
	s.getCalls++
	if s.getCalls > s.failAfter {
		return 0, fmt.Errorf("get failed after %d calls", s.failAfter)
	}
	return s.data[key], nil
}
func (s *getFailAfterNStore) Set(_ context.Context, key string, val float64, _ time.Duration) error {
	s.data[key] = val
	return nil
}
func (s *getFailAfterNStore) Close() error { return nil }

var _ storage.Storage = (*getFailAfterNStore)(nil)
