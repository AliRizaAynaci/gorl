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

Setting `RedisURL` selects the Redis backend, but distributed guarantees still
depend on the chosen strategy. See
[Distributed Semantics](../architecture/distributed-semantics.md).

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
- `Remaining`: remaining whole-request capacity after the current decision
- `Reset`: time until the limiter fully resets or refills if no more requests arrive
- `RetryAfter`: earliest reliable delay before a denied request may be allowed

Middleware adapters should emit duration-based headers only when these values
are positive and reliable for the current result.

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
