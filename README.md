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

### In-memory Performance

| Algorithm          | Operations/sec | Latency per op | Allocations |
| ------------------ | -------------- | -------------- | ----------- |
| **Fixed Window**   | 15,321,034     | 77.12 ns/op    | 0 allocs/op |
| **Leaky Bucket**   | 12,973,014     | 92.89 ns/op    | 0 allocs/op |
| **Sliding Window** | 244,675        | 5,424 ns/op    | 0 allocs/op |
| **Token Bucket**   | 13,772,462     | 87.39 ns/op    | 0 allocs/op |

### Redis-based Performance

| Algorithm          | Operations/sec | Latency per op | Allocations  |
| ------------------ | -------------- | -------------- | ------------ |
| **Fixed Window**   | 8,180          | 138,133 ns/op  | 9 allocs/op  |
| **Leaky Bucket**   | 6,999          | 169,325 ns/op  | 13 allocs/op |
| **Sliding Window** | 7,101          | 154,330 ns/op  | 31 allocs/op |
| **Token Bucket**   | 7,183          | 170,546 ns/op  | 14 allocs/op |

Benchmarks conducted on AMD EPYC 7763 CPU (64-core).

## Contributing

Contributions are welcomed! Please open an issue to discuss changes or submit a pull request with your improvements.

## License

GoRL is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
