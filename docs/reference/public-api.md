# Public API Reference

This page summarizes the main public contracts exposed by the library.

## Constructor

### `gorl.New(cfg core.Config) (core.Limiter, error)`

Creates a limiter by:

- validating config,
- defaulting metrics to `NoopMetrics`,
- choosing storage based on `RedisURL`,
- selecting the requested strategy from the internal registry.

## `core.Config`

```go
type Config struct {
    Strategy  StrategyType
    Limit     int
    Window    time.Duration
    RedisURL  string
    FailOpen  bool
    Metrics MetricsCollector
}
```

### Fields

- `Strategy`
- `Limit`
- `Window`
- `RedisURL`
- `FailOpen`
- `Metrics`

`Config` now contains only constructor-level runtime settings. Request key
selection belongs to the caller or to middleware adapters.

## `core.Limiter`

```go
type Limiter interface {
    Allow(ctx context.Context, key string) (Result, error)
    Close() error
}
```

## `core.Result`

```go
type Result struct {
    Allowed    bool
    Limit      int
    Remaining  int
    Reset      time.Duration
    RetryAfter time.Duration
}
```

### Semantics

- `Allowed`: whether the request may proceed
- `Limit`: configured capacity
- `Remaining`: remaining capacity reported by the algorithm
- `Reset`: time until the algorithm reports a reset/refill boundary
- `RetryAfter`: suggested wait time when denied

Consumers should note that metadata quality depends on the current algorithm
implementation.

## Strategies

Available strategy constants:

- `core.FixedWindow`
- `core.SlidingWindow`
- `core.TokenBucket`
- `core.LeakyBucket`

## Metrics

`core.MetricsCollector` is optional and allows applications to attach external
observability without changing limiter behavior.

## Middleware Packages

Public middleware packages:

- `middleware/http`
- `middleware/gin`
- `middleware/fiber`
- `middleware/echo`

These packages wrap `core.Limiter` rather than exposing a separate rate-limit
engine.

## Key Selection

GoRL does not derive request identity inside `gorl.New`.

- If you call the limiter directly, you provide the key in `Allow(ctx, key)`.
- If you use middleware, the middleware package decides the key via its
  configurable `KeyFunc`.
