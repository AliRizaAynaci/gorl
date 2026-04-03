# Storage and Observability

This guide covers storage backend selection and metrics integration.

## Storage Abstraction

All algorithms operate through the `storage.Storage` interface:

```go
type Storage interface {
    Incr(ctx context.Context, key string, ttl time.Duration) (float64, error)
    Get(ctx context.Context, key string) (float64, error)
    Set(ctx context.Context, key string, val float64, ttl time.Duration) error
    Close() error
}
```

## In-Memory Backend

The in-memory backend is the default.

Characteristics:

- no external dependency,
- local to a single process,
- background cleanup for expired entries,
- good fit for development, tests, or single-instance services.

## Redis Backend

Set `RedisURL` in `core.Config` to use Redis.

```go
limiter, err := gorl.New(core.Config{
    Strategy: core.FixedWindow,
    Limit:    100,
    Window:   time.Minute,
    RedisURL: "redis://localhost:6379/0",
})
```

Characteristics:

- good for shared state across services,
- selected automatically by the top-level constructor,
- depends on `go-redis/v9`.

### Current Caveat

The storage backend exposes atomic Redis primitives, but not every algorithm
performs its full state transition atomically across multiple processes. That
matters if you need strict distributed correctness.

## Metrics Interface

Algorithms accept any implementation of `core.MetricsCollector`.

```go
type MetricsCollector interface {
    IncAllow()
    IncDeny()
    ObserveLatency(elapsed time.Duration)
}
```

If omitted, `core.NoopMetrics` is used.

## Prometheus Integration

The repository includes a Prometheus adapter in `metrics/prometheus.go`.

```go
pm := metrics.NewPrometheusCollector("gorl", "sliding_window")
metrics.RegisterPrometheusCollectors(pm)

limiter, err := gorl.New(core.Config{
    Strategy: core.SlidingWindow,
    Limit:    5,
    Window:   time.Minute,
    Metrics:  pm,
})
```

## Operational Advice

- Use the in-memory store for local development and fast tests.
- Use Redis only after deciding what consistency guarantees you need.
- Keep metrics optional at first; add them once you need production visibility.
- Treat `FailOpen` as an application policy decision, not just a technical one.
