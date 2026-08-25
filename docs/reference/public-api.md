# Public API Reference

This page summarizes the main public contracts exposed by the library. Use
[pkg.go.dev](https://pkg.go.dev/github.com/AliRizaAynaci/gorl/v2) for the exact
package index and source-linked declarations.

## Constructor

### `gorl.New(cfg core.Config) (core.Limiter, error)`

Creates a limiter by:

- validating config,
- defaulting metrics to `NoopMetrics`,
- choosing storage based on `RedisURL`,
- selecting the requested strategy from the internal registry.

Validation requires `Limit > 0` and `Window > 0`. An unsupported strategy
returns `core.ErrUnknownStrategy`. If `RedisURL` is set, an invalid URL or failed
startup ping is returned during construction.

### `gorl.NewResourceLimiter(cfg core.ResourceConfig) (core.ResourceLimiter, error)`

Creates a resource-scoped limiter by:

- validating the default and named resource policies,
- defaulting metrics to `NoopMetrics`,
- choosing storage based on `RedisURL`,
- creating per-resource child limiters that share one storage backend,
- falling back to `DefaultPolicy` for resources not present in `Resources`.

The default policy and every named override require positive limit and window
values. Empty resource names are rejected.

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

Setting `RedisURL` selects the Redis backend and enables the built-in Redis
atomic execution path for the built-in strategies. See
[Distributed Semantics](../architecture/distributed-semantics.md).

## `core.ResourcePolicy`

```go
type ResourcePolicy struct {
    Limit  int
    Window time.Duration
}
```

## `core.ResourceConfig`

```go
type ResourceConfig struct {
    Strategy      StrategyType
    DefaultPolicy ResourcePolicy
    Resources     map[string]ResourcePolicy
    RedisURL      string
    FailOpen      bool
    Metrics       MetricsCollector
}
```

### Semantics

- Existing `core.Config` users do not need to change anything.
- `DefaultPolicy` is required and is used as the fallback for unknown resources.
- `Resources` contains optional per-resource overrides.
- All resources under the same `ResourceConfig` use the same strategy and store selection.

## `core.Limiter`

```go
type Limiter interface {
    Allow(ctx context.Context, key string) (Result, error)
    Close() error
}
```

## `core.ResourceLimiter`

```go
type ResourceLimiter interface {
    AllowResource(ctx context.Context, resource, key string) (Result, error)
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

## Resource Selection

GoRL also supports optional resource-scoped limiting via `core.ResourceLimiter`.

- `resource` selects which policy should be applied.
- `key` selects which identity should be counted under that policy.
- Middleware adapters expose a separate resource function when using the resource-scoped flow.

## Config Loader

The optional `config` package provides:

```go
config.LoadResourceConfig(path string) (core.ResourceConfig, error)
```

It supports `.json`, `.yaml`, and `.yml` files and converts duration strings
such as `1s`, `30s`, and `1m` into `time.Duration`.

The loader accepts either:

- a flat top-level object, or
- a nested `gorl` root object for namespaced configs.

## Errors

The core package exposes sentinel errors:

- `core.ErrBackendUnavailable`,
- `core.ErrConfigInvalid`,
- `core.ErrUnknownStrategy`.

Configuration validation wraps `ErrConfigInvalid`, so callers can use
`errors.Is`. Backend implementations may return contextual errors from their
underlying operations; middleware should log them through a custom error handler
without exposing backend details to clients.

## Package index

| Package | Primary public role |
| --- | --- |
| `gorl` | `New` and `NewResourceLimiter` constructors |
| `core` | Configuration, strategy, result, limiter, and metrics contracts |
| `config` | JSON/YAML resource configuration loader |
| `middleware/http` | Standard-library HTTP adapter and extractors |
| `middleware/gin` | Gin adapter |
| `middleware/fiber` | Fiber adapter |
| `middleware/echo` | Echo adapter |
| `metrics` | Prometheus collector implementation |
| `storage` | Minimal custom storage interface |
| `storage/inmem` | Bundled process-local storage |
| `storage/redis` | Bundled Redis storage and client access |

For exact signatures and Go documentation, open the
[GoRL v2 module on pkg.go.dev](https://pkg.go.dev/github.com/AliRizaAynaci/gorl/v2).
