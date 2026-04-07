<p align="center">
  <img src="logo.png" alt="GoRL Logo" width="180"/>
</p>

# GoRL - High-Performance Rate Limiter Library

GoRL is a high-performance, extensible rate limiter library for Go. It supports multiple algorithms, pluggable storage backends, a metrics collector abstraction, and minimal dependencies for both single-instance deployments and Redis-backed shared-state deployments.

---

## Table of Contents

* [Features](#features)
* [Installation](#installation)
* [Quick Start](#quick-start)
* [Resource-Scoped Limits](#resource-scoped-limits)
* [Docs](#docs)
* [Usage Examples](#usage-examples)
* [Observability](#observability)
* [Benchmarks](#benchmarks)
* [Storage Backends](#storage-backends)
* [Extending GoRL](#extending-gorl)
* [Contributing](#contributing)
* [License](#license)

## Features

* **Algorithms**: Fixed Window, Sliding Window, Token Bucket, Leaky Bucket
* **Storage**: In-memory, Redis, or any custom store (via `Storage` interface)
* **Atomic Redis Execution**: Built-in Redis-backed limiters use Lua-scripted state transitions
* **Fail-Open / Fail-Close**: Configurable policy on backend errors
* **Key Extraction**: Built-in strategies (IP, API key) or custom
* **Resource-Scoped Policies**: Optional per-resource overrides while keeping a shared store and strategy
* **Metrics Collector**: Optional abstraction for counters and histograms, zero-cost when unused
* **Minimal Dependencies**: Zero external requirements for in-memory mode
* **Middleware Support**: Built-in middleware for `net/http`, Fiber, Gin, and Echo

## Installation

```bash
go get github.com/AliRizaAynaci/gorl/v2
```

## Quick Start

```go
import (
  "context"
  "fmt"
  "time"

  "github.com/AliRizaAynaci/gorl/v2"
  "github.com/AliRizaAynaci/gorl/v2/core"
)

func main() {
  limiter, err := gorl.New(core.Config{
    Strategy: core.SlidingWindow,
    Limit:    5,
    Window:   1 * time.Minute,
  })
  if err != nil {
    panic(err)
  }
  defer limiter.Close()

  ctx := context.Background()
  for i := 1; i <= 10; i++ {
    res, _ := limiter.Allow(ctx, "user-123")
    fmt.Printf("Request #%d: allowed=%v, remaining=%d\n", i, res.Allowed, res.Remaining)
  }
}
```

## Resource-Scoped Limits

Existing `v2` usage stays exactly the same. If you want per-resource policies,
you can opt into the additive resource-scoped API:

```go
resourceLimiter, err := gorl.NewResourceLimiter(core.ResourceConfig{
  Strategy: core.SlidingWindow,
  DefaultPolicy: core.ResourcePolicy{
    Limit:  100,
    Window: time.Minute,
  },
  Resources: map[string]core.ResourcePolicy{
    "login": {
      Limit:  5,
      Window: time.Minute,
    },
    "search": {
      Limit:  50,
      Window: time.Second,
    },
  },
})
if err != nil {
  panic(err)
}
defer resourceLimiter.Close()

res, err := resourceLimiter.AllowResource(context.Background(), "login", "user-123")
if err != nil {
  panic(err)
}

fmt.Println(res.Allowed, res.Remaining)
```

Unknown resources use the configured `DefaultPolicy`, so named overrides are
optional rather than required.

### Load Resource Config from JSON or YAML

```go
import (
  "github.com/AliRizaAynaci/gorl/v2"
  "github.com/AliRizaAynaci/gorl/v2/config"
)

cfg, err := config.LoadResourceConfig("limits.yaml")
if err != nil {
  panic(err)
}

resourceLimiter, err := gorl.NewResourceLimiter(cfg)
if err != nil {
  panic(err)
}
defer resourceLimiter.Close()
```

## Docs

Additional library documentation is available under [docs/README.md](docs/README.md).

Recommended entry points:

- [Getting Started](docs/guides/getting-started.md)
- [System Overview](docs/architecture/system-overview.md)
- [Distributed Semantics](docs/architecture/distributed-semantics.md)
- [Middleware Guide](docs/guides/middleware.md)
- [Public API Reference](docs/reference/public-api.md)

## Usage Examples

### HTTP Middleware (Built-in)

GoRL ships with a ready-to-use `net/http` middleware under `middleware/http`.

**Basic Usage (handler wrapping):**

```go
import (
  "net/http"

  "github.com/AliRizaAynaci/gorl/v2"
  "github.com/AliRizaAynaci/gorl/v2/core"
  mw "github.com/AliRizaAynaci/gorl/v2/middleware/http"
)

limiter, _ := gorl.New(core.Config{
  Strategy: core.SlidingWindow,
  Limit:    10,
  Window:   1 * time.Minute,
})

mux := http.NewServeMux()
mux.Handle("/api/", mw.RateLimit(limiter, mw.Options{
  KeyFunc: mw.KeyByIP(),
}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("OK"))
})))

http.ListenAndServe(":8080", mux)
```

**Middleware Chaining:**

```go
rl := mw.NewMiddleware(limiter, mw.Options{
  KeyFunc: mw.KeyByHeader("X-API-Key"),
})

mux.Handle("/api/", rl(myHandler))
```

The middleware always sets `RateLimit-Limit` and `RateLimit-Remaining`, and
adds `RateLimit-Reset` and `Retry-After` when the limiter returns a reliable
duration.

**Available Key Extractors:**
- `mw.KeyByIP()` — client IP (supports `X-Forwarded-For`, `X-Real-Ip`)
- `mw.KeyByHeader("X-API-Key")` — any request header
- `mw.KeyByPath()` — IP + request path (per-endpoint limiting)

### HTTP Middleware (Resource-Scoped)

```go
resourceLimiter, _ := gorl.NewResourceLimiter(core.ResourceConfig{
  Strategy: core.SlidingWindow,
  DefaultPolicy: core.ResourcePolicy{
    Limit:  100,
    Window: time.Minute,
  },
  Resources: map[string]core.ResourcePolicy{
    "/login": {Limit: 5, Window: time.Minute},
    "/search": {Limit: 50, Window: time.Second},
  },
})

mux.Handle("/", mw.RateLimitByResource(resourceLimiter, mw.Options{
  KeyFunc:      mw.KeyByIP(),
  ResourceFunc: mw.ResourceByPath(),
}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("OK"))
})))
```

### Fiber

```go
import (
  "github.com/gofiber/fiber/v2"
  "github.com/AliRizaAynaci/gorl/v2"
  "github.com/AliRizaAynaci/gorl/v2/core"
  fibermw "github.com/AliRizaAynaci/gorl/v2/middleware/fiber"
)

limiter, _ := gorl.New(core.Config{
  Strategy: core.FixedWindow, Limit: 100, Window: time.Minute,
})

app := fiber.New()
app.Use(fibermw.RateLimit(limiter)) // key defaults to c.IP()
app.Listen(":3000")
```

### Gin

```go
import (
  "github.com/gin-gonic/gin"
  "github.com/AliRizaAynaci/gorl/v2"
  "github.com/AliRizaAynaci/gorl/v2/core"
  ginmw "github.com/AliRizaAynaci/gorl/v2/middleware/gin"
)

limiter, _ := gorl.New(core.Config{
  Strategy: core.SlidingWindow, Limit: 100, Window: time.Minute,
})

r := gin.Default()
r.Use(ginmw.RateLimit(limiter)) // key defaults to c.ClientIP()
r.Run(":8080")
```

### Echo

```go
import (
  "github.com/labstack/echo/v4"
  "github.com/AliRizaAynaci/gorl/v2"
  "github.com/AliRizaAynaci/gorl/v2/core"
  echomw "github.com/AliRizaAynaci/gorl/v2/middleware/echo"
)

limiter, _ := gorl.New(core.Config{
  Strategy: core.TokenBucket, Limit: 100, Window: time.Minute,
})

e := echo.New()
e.Use(echomw.RateLimit(limiter)) // key defaults to c.RealIP()
e.Start(":8080")
```

> All framework middlewares set `RateLimit-Limit` and `RateLimit-Remaining`,
> and add duration-based headers when reliable timing data is available.
> Pass a custom `Config{KeyFunc: ..., ResourceFunc: ...}` to override the default
> key or resource extraction behavior.

### Docker & Redis Backend

```bash
docker run --name redis-limiter -p 6379:6379 -d redis
```

```go
limiter, err := gorl.New(core.Config{
  Strategy: core.TokenBucket,
  Limit:    100,
  Window:   1 * time.Minute,
  RedisURL: "redis://localhost:6379/0",
})
if err != nil {
  panic(err)
}
```

## Observability

GoRL provides an optional metrics collector abstraction. Below is an example integrating Prometheus:

```go
import (
  "log"
  "net/http"
  "time"

  "github.com/AliRizaAynaci/gorl/v2"
  "github.com/AliRizaAynaci/gorl/v2/core"
  "github.com/AliRizaAynaci/gorl/v2/metrics"
  mw "github.com/AliRizaAynaci/gorl/v2/middleware/http"
  "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
  // Create and register Prometheus collector
  pm := metrics.NewPrometheusCollector("gorl", "sliding_window")
  metrics.RegisterPrometheusCollectors(pm)

  // Initialize limiter with metrics enabled
  limiter, err := gorl.New(core.Config{
    Strategy: core.SlidingWindow,
    Limit:    5,
    Window:   1 * time.Minute,
    RedisURL: "redis://localhost:6379/0",
    Metrics:  pm,
  })
  if err != nil {
    log.Fatal(err)
  }
  defer limiter.Close()

  // Expose Prometheus metrics endpoint
  http.Handle("/metrics", promhttp.Handler())

  // Application handler with rate limiting middleware
  http.Handle("/api", mw.RateLimitFunc(limiter, mw.Options{
    KeyFunc: mw.KeyByHeader("X-API-Key"),
  }, func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("OK"))
  }))

  log.Println("Listening on :8080")
  log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Benchmarks

Benchmarks below are averages of 3 runs on Apple M4 using:

```bash
go test ./internal/algorithms -run=^$ -bench=. -benchmem -benchtime=1s -count=3
```

Redis results were measured against a local `redis:7-alpine` container and
reflect the Lua-backed atomic execution path.

### In-Memory Backend

| Algorithm      | Single Key (ns/op, B/op, allocs)     | Multi Key (ns/op, B/op, allocs)      |
| -------------- | ------------------------------------ | ------------------------------------ |
| Fixed Window   | 217.2 ns/op, 64 B/op, 4 allocs/op    | 268.8 ns/op, 86 B/op, 5 allocs/op    |
| Sliding Window | 394.8 ns/op, 168 B/op, 9 allocs/op   | 504.2 ns/op, 182 B/op, 10 allocs/op  |
| Token Bucket   | 467.6 ns/op, 272 B/op, 12 allocs/op  | 546.0 ns/op, 300 B/op, 13 allocs/op  |
| Leaky Bucket   | 474.7 ns/op, 272 B/op, 12 allocs/op  | 570.0 ns/op, 286 B/op, 13 allocs/op  |

### Redis Backend

| Algorithm      | Single Key (ns/op, B/op, allocs)        | Multi Key (ns/op, B/op, allocs)         |
| -------------- | --------------------------------------- | --------------------------------------- |
| Fixed Window   | 100797.0 ns/op, 416 B/op, 17 allocs/op | 103031.7 ns/op, 452 B/op, 17 allocs/op |
| Sliding Window | 106871.3 ns/op, 912 B/op, 32 allocs/op | 118876.3 ns/op, 970.3 B/op, 33 allocs/op |
| Token Bucket   | 107571.0 ns/op, 800 B/op, 28 allocs/op | 108520.0 ns/op, 861 B/op, 29 allocs/op |
| Leaky Bucket   | 103766.0 ns/op, 800 B/op, 28 allocs/op | 111682.3 ns/op, 859 B/op, 29 allocs/op |

### Redis: Before Lua vs After Lua

| Algorithm      | Single Key                             | Multi Key                              |
| -------------- | -------------------------------------- | -------------------------------------- |
| Fixed Window   | `178022.0 -> 100797.0 ns/op` `43.4%` faster | `186592.7 -> 103031.7 ns/op` `44.8%` faster |
| Sliding Window | `457973.7 -> 106871.3 ns/op` `76.7%` faster | `462304.7 -> 118876.3 ns/op` `74.3%` faster |
| Token Bucket   | `374951.7 -> 107571.0 ns/op` `71.3%` faster | `399340.7 -> 108520.0 ns/op` `72.8%` faster |
| Leaky Bucket   | `386675.0 -> 103766.0 ns/op` `73.2%` faster | `470074.0 -> 111682.3 ns/op` `76.2%` faster |

These comparisons use the same benchmark command, the same local Redis
container setup, and 3-run averages before and after the Lua migration.

## Storage Backends

GoRL's storage layer uses a minimal key-value interface.

```go
package storage

import (
  "context"
  "time"
)

type Storage interface {
  // Incr atomically increments the value at key by 1, initializing to 1 if missing or expired.
  Incr(ctx context.Context, key string, ttl time.Duration) (float64, error)

  // Get retrieves the numeric value at key, returning 0 if missing or expired.
  Get(ctx context.Context, key string) (float64, error)

  // Set stores the numeric value at key with the specified TTL.
  Set(ctx context.Context, key string, val float64, ttl time.Duration) error

  // Close releases any resources held by the storage backend.
  Close() error
}
```

### In-Memory Store

Lock-free implementation using `sync.Map` and `sync/atomic`:

```go
store := inmem.NewInMemoryStore()
```

* **Use case**: single-instance and unit tests
* **Expiration**: TTL on each write, background GC cleanup
* **Concurrency**: lock-free via atomic CAS operations

### Redis Store

Scalable store leveraging Redis commands:

```go
store := redis.NewRedisStore("redis://localhost:6379/0")
```

* **Execution**: fixed window uses an atomic counter+TTL script; the other built-in algorithms use algorithm-specific Lua scripts
* **TTL Management**: handled inside the Redis script path
* **Use case**: shared state across services
* **Atomicity**: built-in algorithms use Redis Lua scripts for atomic execution

Current distributed guarantees depend on the selected algorithm.

| Backend + Strategy | Multi-instance status |
| --- | --- |
| In-memory + any strategy | single-process only |
| Redis + Fixed Window | supported atomic shared-state path |
| Redis + Sliding Window | supported atomic shared-state path |
| Redis + Token Bucket | supported atomic shared-state path |
| Redis + Leaky Bucket | supported atomic shared-state path |

See [docs/architecture/distributed-semantics.md](docs/architecture/distributed-semantics.md)
for the current support matrix and planned direction.

## Custom Storage Backend

By default, `gorl.New(cfg core.Config)` wires up:

* **Redis** (if `cfg.RedisURL` is set)
* **In-memory** (otherwise)

To add any other storage backend (JetStream, DynamoDB, etc.) without forking the repo, follow these steps:

1. **Create** a sub-package `github.com/AliRizaAynaci/gorl/v2/storage/yourmodule` and implement the `storage.Storage` interface:

   ```go
   // github.com/AliRizaAynaci/gorl/v2/storage/yourmodule/store.go
   package yourmodule

   import (
     "context"
     "time"
     "github.com/AliRizaAynaci/gorl/v2/storage"
   )

   // YourModuleStore holds your connection fields.
   type YourModuleStore struct {
     // e.g. client, context
   }

   // NewYourModuleStore constructs your store with any parameters.
   func NewYourModuleStore(/* params */) *YourModuleStore {
     return &YourModuleStore{/* initialize fields */}
   }

   func (s *YourModuleStore) Incr(ctx context.Context, key string, ttl time.Duration) (float64, error) {
     // increment logic
   }
   func (s *YourModuleStore) Get(ctx context.Context, key string) (float64, error) {
     // get logic
   }
   func (s *YourModuleStore) Set(ctx context.Context, key string, val float64, ttl time.Duration) error {
     // set logic
   }
   func (s *YourModuleStore) Close() error {
     // cleanup logic
   }
   ```

2. **Extend** `core.Config` in `gorl/core/config.go`:

   ```go
   type Config struct {
     Strategy      StrategyType
     Limit         float64
     Window        time.Duration
     RedisURL      string
     YourModuleURL string // ← new field
     Metrics       Metrics
   }
   ```

3. **Wire** your store in `gorl/limiter.go`:

   ```go
   func New(cfg core.Config) (core.Limiter, error) {
     if cfg.Metrics == nil {
       cfg.Metrics = &core.NoopMetrics{}
     }

     var store storage.Storage
     switch {
     case cfg.YourModuleURL != "":
       store = yourmodule.NewYourModuleStore(cfg.YourModuleURL)
     case cfg.RedisURL != "":
       store = redis.NewRedisStore(cfg.RedisURL)
     default:
       store = inmem.NewInMemoryStore()
     }

     constructor, ok := strategyRegistry[cfg.Strategy]
     if !ok {
       return nil, core.ErrUnknownStrategy
     }
     return constructor(cfg, store), nil
   }
   ```

4. **Use** your custom backend:

   ```go
   import (
     "log"
     "time"
     "github.com/AliRizaAynaci/gorl/v2"
     "github.com/AliRizaAynaci/gorl/v2/core"
   )

   cfg := core.Config{
     Strategy:      core.SlidingWindow,
     Limit:         100,
     Window:        time.Minute,
     YourModuleURL: "your-backend://connection-string",
   }
   limiter, err := gorl.New(cfg)
   if err != nil {
     log.Fatal(err)
   }
   ```

> **Note:** After implementing and wiring up your custom storage backend, open a Pull Request against the `main` branch to merge these changes into the GoRL repository before using it in production.

## Key Selection

GoRL accepts a rate-limit key as the second argument to `Allow(ctx, key)`.

- In direct library usage, your application builds and passes that key.
- In middleware usage, the middleware's `KeyFunc` determines the key.

Example:

```go
key := tenantID + ":" + userID
res, err := limiter.Allow(ctx, key)
```

## Resource Selection

Resource-scoped limiters add a second routing dimension on top of keys:

- `resource` selects which policy should be applied
- `key` selects which identity is counted within that policy

Example:

```go
res, err := resourceLimiter.AllowResource(ctx, "github_api", "tenant:acme")
```


## Contributing

1. Fork the repository
2. Create a branch: `git checkout -b feature/YourFeature`
3. Commit changes: `git commit -m "Add feature"`
4. Push to branch: `git push origin feature/YourFeature`
5. Submit a Pull Request

Please review our [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.
