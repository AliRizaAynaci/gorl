<p align="center">
  <img src="logo.png" alt="GoRL Logo" width="180"/>
</p>

# GoRL - High-Performance Rate Limiter Library

![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)

## Overview

GoRL is an efficient, scalable, and flexible rate limiting library for Go applications. It supports multiple rate limiting strategies with both in-memory and Redis-backed implementations, ideal for use in high-throughput systems.

## Features

* Multiple rate-limiting algorithms:

  * **Fixed Window**
  * **Sliding Window**
  * **Token Bucket** (supports bursts)
  * **Leaky Bucket**

* Flexible backend support:

  * **In-memory** (ideal for single-instance deployments)
  * **Redis-based** (for distributed, scalable rate limiting)

* Configurable fail-open or fail-close behavior.

* Customizable key extraction (IP, API keys, tokens, custom).

## Installation

```bash
go get github.com/AliRizaAynaci/gorl
```

## Usage

### Basic Example (Redis Sliding Window)

```go
package main

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
		Limit:    3,
		Window:   10 * time.Second,
		RedisURL: "redis://localhost:6379/0",
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow("example-api-key")
		fmt.Printf("Request #%d: allowed=%v, err=%v\n", i+1, allowed, err)
		time.Sleep(2 * time.Second)
	}
}
```

### Fiber Middleware Example (In-Memory Token Bucket)

```go
package main

import (
	"log"
	"time"

	"github.com/AliRizaAynaci/gorl"
	"github.com/AliRizaAynaci/gorl/core"
	"github.com/gofiber/fiber/v2"
)

func main() {
	limiter, err := gorl.New(core.Config{
		Strategy: core.TokenBucket,
		KeyBy:    core.KeyByIP,
		Limit:    5,
		Window:   10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		allowed, err := limiter.Allow(c.IP())
		if err != nil || !allowed {
			return c.Status(fiber.StatusTooManyRequests).SendString("Rate limit exceeded")
		}
		return c.Next()
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome!")
	})

	log.Fatal(app.Listen(":3000"))
}
```

## Docker (Redis Examples)

To run Redis-based examples, you can quickly spin up a Redis instance using Docker:

```bash
docker run --name redis-limiter -p 6379:6379 -d redis
```

## Benchmark Results

### In-memory Benchmark Results

| Algorithm              | Operations | Avg Time per Op (ns) | Bytes per Op | Allocations per Op |
| ---------------------- | ---------- | -------------------- | ------------ | ------------------ |
| Fixed Window Limiter   | 696 ops    | 1,855,605            | 362          | 7                  |
| Fixed Window Limiter   | 639 ops    | 1,643,390            | 376          | 8                  |
| Leaky Bucket Limiter   | 513 ops    | 2,400,053            | 1,746        | 30                 |
| Leaky Bucket Limiter   | 475 ops    | 2,480,942            | 1,696        | 27                 |
| Sliding Window Limiter | 376 ops    | 2,880,348            | 21,283       | 312                |
| Sliding Window Limiter | 580 ops    | 2,210,054            | 654          | 16                 |
| Token Bucket Limiter   | 482 ops    | 2,183,965            | 1,763        | 31                 |
| Token Bucket Limiter   | 547 ops    | 2,163,840            | 1,692        | 27                 |

### Redis Benchmark Results

| Algorithm              | Operations | Avg Time per Op (ns) | Bytes per Op | Allocations per Op |
| ---------------------- | ---------- | -------------------- | ------------ | ------------------ |
| Fixed Window Limiter   | 772 ops    | 1,520,749            | 360          | 7                  |
| Fixed Window Limiter   | 795 ops    | 1,456,494            | 370          | 8                  |
| Leaky Bucket Limiter   | 572 ops    | 2,165,656            | 1,742        | 30                 |
| Leaky Bucket Limiter   | 553 ops    | 2,160,602            | 1,690        | 27                 |
| Sliding Window Limiter | 511 ops    | 2,476,893            | 25,085       | 379                |
| Sliding Window Limiter | 501 ops    | 2,129,440            | 658          | 17                 |
| Token Bucket Limiter   | 524 ops    | 2,227,303            | 1,763        | 31                 |
| Token Bucket Limiter   | 525 ops    | 2,204,126            | 1,694        | 27                 |

Benchmarks conducted on AMD Ryzen 7 4800H CPU.

## Advanced Usage: Storage Layer

GoRL uses a pluggable storage layer. You can use the built-in in-memory or Redis backends, or bring your own by implementing the `storage.Storage` interface.

### Built-in Store Implementations

```go
import (
    "github.com/AliRizaAynaci/gorl/core"
    "github.com/AliRizaAynaci/gorl/internal/algorithms"
    "github.com/AliRizaAynaci/gorl/storage/inmem"
    "github.com/AliRizaAynaci/gorl/storage/redis"
)
```

#### In-memory:

```go
store := inmem.NewInMemoryStore()
```

#### Redis:

```go
store := redis.NewRedisStore("redis://localhost:6379/0")
```

You can then pass your custom store manually with a specific limiter algorithm:

```go
limiter := algorithms.NewFixedWindowLimiter(core.Config{
    Limit:  5,
    Window: 10 * time.Second,
}, store)
```

> Note: If you’re using the top-level `gorl.New()` function, it automatically selects Redis or In-Memory store based on `core.Config.RedisURL`.

### Writing Your Own Store Backend

To implement a custom backend (e.g., DynamoDB, NATS, SQL), implement the following interface:

```go
type Storage interface {
    Incr(key string, ttl time.Duration) (float64, error)
    Get(key string) (float64, error)
    Set(key string, val float64, ttl time.Duration) error

    AppendList(key string, value int64, ttl time.Duration) error
    GetList(key string) ([]int64, error)
    TrimList(key string, count int) error

    ZAdd(key string, score float64, member int64, ttl time.Duration) error
    ZRemRangeByScore(key string, min, max float64) error
    ZCard(key string) (int64, error)
    ZRangeByScore(key string, min, max float64) ([]int64, error)

    HMSet(key string, fields map[string]float64, ttl time.Duration) error
    HMGet(key string, fields ...string) (map[string]float64, error)
}
```

This design gives you full control over how data is stored, enabling integration with your own distributed cache, message broker, or database.

## Contributing

Contributions are welcomed! Please open an issue to discuss changes or submit a pull request with your improvements.

## License

GoRL is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
