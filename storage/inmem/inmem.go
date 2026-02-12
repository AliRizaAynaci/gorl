// Package inmem provides an in-memory storage implementation for the rate limiter.
package inmem

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/storage"
)

const defaultGCInterval = 1 * time.Minute

type inMemoryStore struct {
	data sync.Map // Map of string -> *item
	done chan struct{}
}

type item struct {
	value     uint64 // stores math.Float64bits(val), updated atomically
	expiresAt int64  // UnixNano, fixed at creation
}

// NewInMemoryStore returns a storage with a background garbage collector
// that cleans up expired entries every minute.
func NewInMemoryStore() storage.Storage {
	s := &inMemoryStore{
		done: make(chan struct{}),
	}
	go s.gc(defaultGCInterval)
	return s
}

// gc periodically removes expired entries from the store.
func (s *inMemoryStore) gc(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.removeExpired()
		}
	}
}

// removeExpired deletes all entries whose TTL has passed.
func (s *inMemoryStore) removeExpired() {
	now := time.Now().UnixNano()
	s.data.Range(func(key, value any) bool {
		it := value.(*item)
		if it.expiresAt < now {
			// Try to delete if it's still the same item (CompareAndDelete)
			// Go 1.20+ supports CompareAndDelete
			s.data.CompareAndDelete(key, value)
		}
		return true
	})
}

// Incr atomically increments the value at key by 1.
// If missing or expired, initializes to 1 with the given TTL.
func (s *inMemoryStore) Incr(_ context.Context, key string, ttl time.Duration) (float64, error) {
	for {
		val, loaded := s.data.Load(key)
		if !loaded {
			// Try to initialize
			newItem := &item{
				value:     math.Float64bits(1.0),
				expiresAt: time.Now().Add(ttl).UnixNano(),
			}
			actual, loaded := s.data.LoadOrStore(key, newItem)
			if !loaded {
				return 1.0, nil
			}
			val = actual // Use the item that won the race
		}

		it := val.(*item)
		now := time.Now().UnixNano()

		// Check expiry
		if it.expiresAt < now {
			newItem := &item{
				value:     math.Float64bits(1.0),
				expiresAt: time.Now().Add(ttl).UnixNano(),
			}
			// Atomic replacement
			if s.data.CompareAndSwap(key, val, newItem) {
				return 1.0, nil
			}
			continue // Reload and retry
		}

		// Valid item, atomic increment loop
		for {
			oldBits := atomic.LoadUint64(&it.value)
			oldVal := math.Float64frombits(oldBits)
			newVal := oldVal + 1.0
			newBits := math.Float64bits(newVal)

			if atomic.CompareAndSwapUint64(&it.value, oldBits, newBits) {
				return newVal, nil
			}
			// Spin loop for value update
		}
	}
}

// Get retrieves the current value at key, or 0 if missing/expired.
func (s *inMemoryStore) Get(_ context.Context, key string) (float64, error) {
	val, ok := s.data.Load(key)
	if !ok {
		return 0, nil
	}

	it := val.(*item)
	now := time.Now().UnixNano()
	if it.expiresAt < now {
		return 0, nil
	}

	bits := atomic.LoadUint64(&it.value)
	return math.Float64frombits(bits), nil
}

// Set stores the given value at key with TTL.
func (s *inMemoryStore) Set(_ context.Context, key string, val float64, ttl time.Duration) error {
	newItem := &item{
		value:     math.Float64bits(val),
		expiresAt: time.Now().Add(ttl).UnixNano(),
	}
	s.data.Store(key, newItem)
	return nil
}

// Close stops the background garbage collector goroutine.
func (s *inMemoryStore) Close() error {
	close(s.done)
	return nil
}
