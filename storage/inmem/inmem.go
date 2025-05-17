package inmem

import (
	"errors"
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/storage"
)

type inMemoryStore struct {
	data     map[string]*item
	listData map[string][]int64
	listTTL  map[string]time.Time

	zsetData map[string][]zsetEntry
	zsetTTL  map[string]time.Time

	hashData map[string]map[string]float64
	hashTTL  map[string]time.Time

	mu sync.Mutex
}

type item struct {
	value     float64
	expiresAt time.Time
}

type zsetEntry struct {
	score  float64
	member int64
}

// Constructor
func NewInMemoryStore() storage.Storage {
	return &inMemoryStore{
		data:     make(map[string]*item),
		listData: make(map[string][]int64),
		listTTL:  make(map[string]time.Time),
		zsetData: make(map[string][]zsetEntry),
		zsetTTL:  make(map[string]time.Time),
		hashData: make(map[string]map[string]float64),
		hashTTL:  make(map[string]time.Time),
	}
}

// --- Counter ops ---

func (s *inMemoryStore) Incr(key string, ttl time.Duration) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.data[key]
	now := time.Now()
	if !ok || it.expiresAt.Before(now) {
		s.data[key] = &item{value: 1, expiresAt: now.Add(ttl)}
		return 1, nil
	}
	it.value++
	return it.value, nil
}

func (s *inMemoryStore) Get(key string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.data[key]
	now := time.Now()
	if !ok || it.expiresAt.Before(now) {
		return 0, errors.New("not found or expired")
	}
	return it.value, nil
}

func (s *inMemoryStore) Set(key string, val float64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = &item{value: val, expiresAt: time.Now().Add(ttl)}
	return nil
}

// --- List ops (int64 olarak kalabilir) ---

func (s *inMemoryStore) AppendList(key string, value int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	expire := now.Add(ttl)
	if t, ok := s.listTTL[key]; ok && t.Before(now) {
		s.listData[key] = nil
	}
	s.listData[key] = append(s.listData[key], value)
	s.listTTL[key] = expire
	return nil
}

func (s *inMemoryStore) GetList(key string) ([]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if t, ok := s.listTTL[key]; ok && t.Before(now) {
		delete(s.listData, key)
		delete(s.listTTL, key)
		return nil, errors.New("not found or expired")
	}
	vals, ok := s.listData[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return vals, nil
}

func (s *inMemoryStore) TrimList(key string, count int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if t, ok := s.listTTL[key]; ok && t.Before(now) {
		delete(s.listData, key)
		delete(s.listTTL, key)
		return errors.New("not found or expired")
	}
	vals, ok := s.listData[key]
	if !ok {
		return errors.New("not found")
	}
	if len(vals) > count {
		s.listData[key] = vals[len(vals)-count:]
	}
	return nil
}

// --- Sorted set ops (very basic, int64 olarak bırakılabilir) ---

func (s *inMemoryStore) ZAdd(key string, score float64, member int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	expire := now.Add(ttl)
	if t, ok := s.zsetTTL[key]; ok && t.Before(now) {
		s.zsetData[key] = nil
	}
	s.zsetData[key] = append(s.zsetData[key], zsetEntry{score, member})
	s.zsetTTL[key] = expire
	return nil
}

func (s *inMemoryStore) ZRemRangeByScore(key string, min, max float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, ok := s.zsetData[key]
	if !ok {
		return errors.New("not found")
	}
	var filtered []zsetEntry
	for _, entry := range entries {
		if entry.score < min || entry.score > max {
			filtered = append(filtered, entry)
		}
	}
	s.zsetData[key] = filtered
	return nil
}

func (s *inMemoryStore) ZCard(key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if t, ok := s.zsetTTL[key]; ok && t.Before(now) {
		delete(s.zsetData, key)
		delete(s.zsetTTL, key)
		return 0, errors.New("not found or expired")
	}
	entries, ok := s.zsetData[key]
	if !ok {
		return 0, errors.New("not found")
	}
	return int64(len(entries)), nil
}

func (s *inMemoryStore) ZRangeByScore(key string, min, max float64) ([]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, ok := s.zsetData[key]
	if !ok {
		return nil, errors.New("not found")
	}
	var result []int64
	for _, entry := range entries {
		if entry.score >= min && entry.score <= max {
			result = append(result, entry.member)
		}
	}
	return result, nil
}

// --- Hash ops ---

func (s *inMemoryStore) HMSet(key string, fields map[string]float64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	expire := now.Add(ttl)
	if t, ok := s.hashTTL[key]; ok && t.Before(now) {
		s.hashData[key] = nil
	}
	if s.hashData[key] == nil {
		s.hashData[key] = make(map[string]float64)
	}
	for k, v := range fields {
		s.hashData[key][k] = v
	}
	s.hashTTL[key] = expire
	return nil
}

func (s *inMemoryStore) HMGet(key string, fields ...string) (map[string]float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if t, ok := s.hashTTL[key]; ok && t.Before(now) {
		delete(s.hashData, key)
		delete(s.hashTTL, key)
		return nil, errors.New("not found or expired")
	}
	m, ok := s.hashData[key]
	if !ok {
		return nil, errors.New("not found")
	}
	res := make(map[string]float64)
	for _, f := range fields {
		res[f] = m[f]
	}
	return res, nil
}
