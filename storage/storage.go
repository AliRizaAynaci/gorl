package storage

import "time"

type Storage interface {
	// Simple counter ops
	Incr(key string, ttl time.Duration) (float64, error)
	Get(key string) (float64, error)
	Set(key string, val float64, ttl time.Duration) error

	// List ops (for sliding window)
	AppendList(key string, value int64, ttl time.Duration) error
	GetList(key string) ([]int64, error)
	TrimList(key string, count int) error

	// Sorted set ops (for precise sliding window)
	ZAdd(key string, score float64, member int64, ttl time.Duration) error
	ZRemRangeByScore(key string, min, max float64) error
	ZCard(key string) (int64, error)
	ZRangeByScore(key string, min, max float64) ([]int64, error)

	// Hash ops (for complex state)
	HMSet(key string, fields map[string]float64, ttl time.Duration) error
	HMGet(key string, fields ...string) (map[string]float64, error)
}
