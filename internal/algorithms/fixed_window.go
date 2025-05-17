package algorithms

import (
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

type FixedWindowLimiter struct {
	limit  int
	window time.Duration
	store  storage.Storage
	prefix string
}

func NewFixedWindowLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &FixedWindowLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  store,
	}
}

func (f *FixedWindowLimiter) Allow(key string) (bool, error) {
	storageKey := f.prefix + ":" + key

	count, err := f.store.Incr(storageKey, f.window)
	if err != nil {
		return false, err
	}
	if count > float64(f.limit) {
		return false, nil
	}
	return true, nil
}
