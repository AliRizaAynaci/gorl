<p align="center">
  <img src="logo.png" alt="GoRL Logo" width="180"/>
</p>

# GoRL - High-Performance Rate Limiter Library

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/AliRizaAynaci/gorl/actions) [![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)


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
    fmt.Printf("Request #%d: allowed=%v\n", i+1, allowed)
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

| Algorithm      | In-Memory (ns/op) | Redis (ns/op) |
| -------------- | ----------------- | ------------- |
| Fixed Window   | 1.6M              | 1.5M          |
| Sliding Window | 2.3M              | 2.4M          |
| Token Bucket   | 2.2M              | 2.2M          |
| Leaky Bucket   | 2.4M              | 2.2M          |

## Storage Backends

GoRL's pluggable storage layer allows you to seamlessly switch between built-in backends or integrate your own storage solutions. The storage interface is defined in `storage/storage.go`:

```go
package storage

import "time"

// Storage defines the methods required for a rate limiter backend.
type Storage interface {
    // Counter operations
    Incr(key string, ttl time.Duration) (float64, error)
    Get(key string) (float64, error)
    Set(key string, val float64, ttl time.Duration) error

    // List operations (for sliding window)
    AppendList(key string, value int64, ttl time.Duration) error
    GetList(key string) ([]int64, error)
    TrimList(key string, count int) error

    // Sorted set operations (for precise sliding window)
    ZAdd(key string, score float64, member int64, ttl time.Duration) error
    ZRemRangeByScore(key string, min, max float64) error
    ZCard(key string) (int64, error)
    ZRangeByScore(key string, min, max float64) ([]int64, error)

    // Hash operations (for complex state)
    HMSet(key string, fields map[string]float64, ttl time.Duration) error
    HMGet(key string, fields ...string) (map[string]float64, error)
}
```

### Built-in Backends

#### In-Memory Store

The in-memory store (`inmem.NewInMemoryStore()`) is a thread-safe implementation using Go's `sync.Mutex`. It provides:

* **Data Structures**:

  * `map[string]*item` for counters with expiration.
  * `map[string][]int64` for lists.
  * `map[string][]zsetEntry` for sorted sets.
  * `map[string]map[string]float64` for hash fields.
* **Expiration**: TTL is set on each write; expired entries are lazily removed on access.
* **Concurrency**: A single mutex protects all operations.

```go
store := inmem.NewInMemoryStore()
```

Use case: ideal for single-instance deployments and unit tests.

#### Redis Store

The Redis store (`redis.NewRedisStore(redisURL)`) leverages Redis commands to support scalable, distributed rate limiting:

* **Counter**: `INCR` + `EXPIRE`
* **List**: `RPUSH`, `LRANGE`, `LTRIM`
* **Sorted Set**: `ZADD`, `ZREMRANGEBYSCORE`, `ZCARD`, `ZRANGEBYSCORE`
* **Hash**: `HSET`, `HMGET`
* **TTL Management**: `EXPIRE` is called after each write.

```go
store := redis.NewRedisStore("redis://localhost:6379/0")
```

Use case: distributed services requiring a centralized store.

### Implementing a Custom Store

To add your own backend (e.g., DynamoDB, NATS KV, SQL), implement the `Storage` interface:

```go
type MyStore struct {
    // internal client or connection
}

func (m *MyStore) Incr(key string, ttl time.Duration) (float64, error) { /* ... */ }
func (m *MyStore) Get(key string) (float64, error) { /* ... */ }
// implement remaining methods...
```

Then pass your implementation to any limiter constructor:

```go
myStore := &MyStore{ /* init */ }
limiter := algorithms.NewSlidingWindowLimiter(core.Config{
    Limit:  100,
    Window: 1 * time.Minute,
}, myStore)
```

### Choosing a Backend

* **In-Memory**: Best for single-node, low-latency scenarios and unit tests.
* **Redis**: Ideal for distributed environments and when persistence or high availability is needed.
* **Custom**: Tailor to specific infrastructure (e.g., cloud services, proprietary caches).

## Extending GoRL

Implement the `Storage` interface in `storage/storage.go`:

```go
type Storage interface {
  Incr(key string, ttl time.Duration) (float64, error)
  /* ... */
}
```

## Roadmap

* [ ] JetStream KV support
* [ ] Prometheus monitoring middleware

## Contributing

1. Fork the repository
2. Create a branch: `git checkout -b feature/YourFeature`
3. Commit changes: `git commit -m 'Add feature'`
4. Push to branch: `git push origin feature/YourFeature`
5. Submit a Pull Request

Please review our [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.

