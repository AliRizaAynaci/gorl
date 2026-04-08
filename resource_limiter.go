package gorl

import (
	"context"
	"fmt"
	"sync"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage"
)

type resourceRouter struct {
	defaultLimiter core.Limiter
	limiters       map[string]core.Limiter
	store          storage.Storage
	closeOnce      sync.Once
	closeErr       error
}

type evalScriptCapable interface {
	EvalScript(ctx context.Context, name string, keys []string, args ...int64) ([]int64, error)
}

type sharedStore struct {
	storage.Storage
}

func (s sharedStore) Close() error {
	return nil
}

type sharedScriptStore struct {
	storage.Storage
	runner evalScriptCapable
}

func (s sharedScriptStore) Close() error {
	return nil
}

func (s sharedScriptStore) EvalScript(ctx context.Context, name string, keys []string, args ...int64) ([]int64, error) {
	return s.runner.EvalScript(ctx, name, keys, args...)
}

func newResourceRouter(
	cfg core.ResourceConfig,
	store storage.Storage,
	constructor func(core.Config, storage.Storage) core.Limiter,
) core.ResourceLimiter {
	defaultLimiter := constructor(resourceConfigToCore(cfg, cfg.DefaultPolicy), wrapSharedStore(store))
	limiters := make(map[string]core.Limiter, len(cfg.Resources))
	for resource, policy := range cfg.Resources {
		limiters[resource] = constructor(resourceConfigToCore(cfg, policy), wrapSharedStore(store))
	}

	return &resourceRouter{
		defaultLimiter: defaultLimiter,
		limiters:       limiters,
		store:          store,
	}
}

func (r *resourceRouter) AllowResource(ctx context.Context, resource, key string) (core.Result, error) {
	limiter, ok := r.limiters[resource]
	if !ok {
		limiter = r.defaultLimiter
	}
	return limiter.Allow(ctx, buildResourceKey(resource, key))
}

func (r *resourceRouter) Close() error {
	r.closeOnce.Do(func() {
		r.closeErr = r.store.Close()
	})
	return r.closeErr
}

func resourceConfigToCore(cfg core.ResourceConfig, policy core.ResourcePolicy) core.Config {
	return core.Config{
		Strategy: cfg.Strategy,
		Limit:    policy.Limit,
		Window:   policy.Window,
		RedisURL: cfg.RedisURL,
		FailOpen: cfg.FailOpen,
		Metrics:  cfg.Metrics,
	}
}

func wrapSharedStore(store storage.Storage) storage.Storage {
	if runner, ok := store.(evalScriptCapable); ok {
		return sharedScriptStore{Storage: store, runner: runner}
	}
	return sharedStore{Storage: store}
}

func buildResourceKey(resource, key string) string {
	return fmt.Sprintf("%d:%s:%s", len(resource), resource, key)
}
