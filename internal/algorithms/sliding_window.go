package algorithms

import (
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

type SlidingWindowLimiter struct {
	limit  int
	window time.Duration
	store  storage.Storage
	prefix string
}

func NewSlidingWindowLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &SlidingWindowLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  store,
	}
}

func (s *SlidingWindowLimiter) Allow(key string) (bool, error) {
	storageKey := s.prefix + ":" + key

	// Get current window start time
	now := time.Now()
	currentWindowStart := now.Add(-s.window).UnixNano()

	// Retrieve timestamps of previous requests in the window
	timestamps, err := s.store.GetList(storageKey)
	if err != nil {
		// If we can't get the list, create a new one
		timestamps = []int64{}
	}

	// Filter out only timestamps that are still within the window
	var validTimestamps []int64
	for _, ts := range timestamps {
		if ts >= currentWindowStart {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if adding one more request would exceed the limit
	allowed := len(validTimestamps) < s.limit

	// Only add the current timestamp if the request is allowed
	if allowed {
		timestamp := now.UnixNano()
		// Add current timestamp to window
		err := s.store.AppendList(storageKey, timestamp, s.window)
		if err != nil {
			return false, err
		}
	}

	return allowed, nil
}
