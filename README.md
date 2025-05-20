<p align="center">
  <img src="logo.png" alt="GoRL Logo" width="180"/>
</p>

# GoRL - High-Performance Rate Limiter Library

&#x20;&#x20;

GoRL is a high-performance, extensible rate limiter library for Go. It offers multiple algorithms, pluggable storage backends, and minimal dependencies, making it ideal for both single-instance and distributed systems.

---

## Table of Contents

* [Features](#features)
* [Installation](#installation)
* [Quick Start](#quick-start)
* [Examples](#examples)
* [Benchmarks](#benchmarks)
* [Storage Backends](#storage-backends)
* [Extending GoRL](#extending-gorl)
* [Roadmap](#roadmap)
* [Contributing](#contributing)
* [License](#license)

## Features

* **Algorithms**: Fixed Window, Sliding Window, Token Bucket (with burst), Leaky Bucket
* **Storage**: In-memory, Redis, or any custom store (via `Storage` interface)
* **Fail-Open / Fail-Close**: Configurable policy on backend errors
* **Key Extraction**: Built-in strategies (IP, API key) or custom
* **Minimal Dependencies**: Zero external requirements for in-memory mode
* **Middleware Support**: Ready-made integrations (e.g., Fiber)

## Installation

```bash
go get github.com/AliRizaAynaci/gorl
```

## Quick Start

```go
import (
  "fmt"
  "time"

  "github.com/AliRizaAynaci/gorl"
  "github.com/AliRizaAynaci/gorl/core"
)

func main() {
  limiter, err := gorl.New(core.Config{
    Strategy: core.SlidingWindow,
    KeyBy:    core.KeyByAPIKey,
    Limit:    5,
    Window:   1 * time.Minute,
    RedisURL: "redis://localhost:6379/0",
  })
  if err != nil {
    panic(err)
  }

  for i := 0; i < 10; i++ {
    allowed, _ := limiter.Allow("user-123")
    fmt.Printf("Request #%%d: allowed=%%v\n", i+1, allowed)
  }
}
```

## Examples

### Fiber Middleware (In-Memory Token Bucket)

```go
app.Use(func(c *fiber.Ctx) error {
  allowed, err := limiter.Allow(c.IP())
  if err != nil || !allowed {
    return c.Status(fiber.StatusTooManyRequests).SendString("Rate limit exceeded")
  }
  return c.Next()
})
```

### Docker & Redis

```bash
docker run --name redis-limiter -p 6379:6379 -d redis
```

## Benchmarks

Benchmarks run on AMD Ryzen 7 4800H.

| Algorithm      | Single Key (ns/op, B/op, allocs) | Multi Key (ns/op, B/op, allocs) |
| -------------- | -------------------------------- | ------------------------------- |
| Fixed Window   | 89.2 ns/op, 24 B/op, 1 alloc     | 202.5 ns/op, 30 B/op, 2 allocs  |
| Leaky Bucket   | 333.8 ns/op, 112 B/op, 4 allocs  | 506.4 ns/op, 126 B/op, 5 allocs |
| Sliding Window | 260.5 ns/op, 72 B/op, 3 allocs   | 444.0 ns/op, 86 B/op, 4 allocs  |
| Token Bucket   | 339.6 ns/op, 128 B/op, 4 allocs  | 504.4 ns/op, 126 B/op, 5 allocs |

## Storage Backends

GoRL's pluggable storage layer requires only a minimal key-value interface:

```go
package storage

import "time"

// Storage defines a minimal interface for rate limiter backends.
// Implementations only need to support Get, Set and Incr with TTL.
type Storage interface {
  // Incr atomically increments the value at key by 1, initializing to 1 if missing or expired.
  Incr(key string, ttl time.Duration) (float64, error)

  // Get retrieves the numeric value at key, returning 0 if missing or expired.
  Get(key string) (float64, error)

  // Set stores the numeric value at key with the specified TTL.
  Set(key string, val float64, ttl time.Duration) error
}
```

### Built-in Backends

#### In-Memory Store

The in-memory store (`inmem.NewInMemoryStore()`) is a thread-safe implementation using Go's `sync.Mutex`. It provides:

* **Data Structures**: simple counters with expiration
* **Expiration**: TTL is set on each write; expired entries are lazily removed on access
* **Concurrency**: a mutex protects all operations

```go
store := inmem.NewInMemoryStore()
```

Use case: ideal for single-instance deployments and unit tests.

#### Redis Store

The Redis store (`redis.NewRedisStore(redisURL)`) leverages Redis commands for scalable, distributed rate limiting:

* **Counter**: `INCR` + `EXPIRE`
* **TTL Management**: `EXPIRE` after each write

```go
store := redis.NewRedisStore("redis://localhost:6379/0")
```

Use case: distributed services requiring a centralized store.

### Implementing a Custom Store

To add your own backend (e.g., DynamoDB, NATS KV, SQL), implement the `Storage` interface:

```go
type MyStore struct { /* ... */ }

func (m *MyStore) Incr(key string, ttl time.Duration) (float64, error) { /* ... */ }
func (m *MyStore) Get(key string) (float64, error)       { /* ... */ }
func (m *MyStore) Set(key string, val float64, ttl time.Duration) error { /* ... */ }
```

Then pass your implementation directly to an algorithm constructor from the internal algorithms package:

```go
import (
    "github.com/AliRizaAynaci/gorl/core"
    "github.com/AliRizaAynaci/gorl/internal/algorithms"
)

myStore := &MyStore{ /* init */ }
limiter := algorithms.NewSlidingWindowLimiter(core.Config{
    Limit:  100,
    Window: 1 * time.Minute,
}, myStore)
```

## Extending GoRL

To extend GoRL with custom storage backends, implement the Storage interface as described above and pass your store directly to any algorithm constructor (e.g., algorithms.NewTokenBucketLimiter).


## Contributing

1. Fork the repository
2. Create a branch: `git checkout -b feature/YourFeature`
3. Commit changes: `git commit -m 'Add feature'`
4. Push to branch: `git push origin feature/YourFeature`
5. Submit a Pull Request

Please review our [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.

