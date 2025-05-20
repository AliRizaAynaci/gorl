package inmem

import (
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/storage"
)

type inMemoryStore struct {
	mu   sync.Mutex
	data map[string]*item
}

type item struct {
	value     float64
	expiresAt time.Time
}

// NewInMemoryStore returns a storage that only supports Incr, Get, Set.
func NewInMemoryStore() storage.Storage {
	return &inMemoryStore{
		data: make(map[string]*item),
	}
}

// Incr atomically increments the value at key by 1.
// If missing or expired, initializes to 1 with the given TTL.
func (s *inMemoryStore) Incr(key string, ttl time.Duration) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	it, ok := s.data[key]
	if !ok || it.expiresAt.Before(now) {
		s.data[key] = &item{value: 1, expiresAt: now.Add(ttl)}
		return 1, nil
	}
	it.value++
	return it.value, nil
}

// Get retrieves the current value at key, or 0 if missing/expired.
func (s *inMemoryStore) Get(key string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	it, ok := s.data[key]
	if !ok || it.expiresAt.Before(now) {
		return 0, nil
	}
	return it.value, nil
}

// Set stores the given value at key with TTL.
func (s *inMemoryStore) Set(key string, val float64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &item{value: val, expiresAt: time.Now().Add(ttl)}
	return nil
}
