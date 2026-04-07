# Getting Started

This guide walks through the minimum setup needed to use GoRL in an
application.

## Install

```bash
go get github.com/AliRizaAynaci/gorl/v2
```

## Create a Limiter

```go
package main

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
        Window:   time.Minute,
    })
    if err != nil {
        panic(err)
    }
    defer limiter.Close()

    res, err := limiter.Allow(context.Background(), "user-123")
    if err != nil {
        panic(err)
    }

    fmt.Println(res.Allowed, res.Remaining)
}
```

## Choose a Strategy

GoRL currently supports:

- `core.FixedWindow`
- `core.SlidingWindow`
- `core.TokenBucket`
- `core.LeakyBucket`

## Choose a Storage Backend

The constructor chooses storage like this:

- default: in-memory store
- when `RedisURL` is set: Redis store

```go
limiter, err := gorl.New(core.Config{
    Strategy: core.TokenBucket,
    Limit:    100,
    Window:   time.Minute,
    RedisURL: "redis://localhost:6379/0",
})
```

## Runtime Model

The core runtime call is:

```go
res, err := limiter.Allow(ctx, key)
```

Where:

- `ctx` carries cancellation and deadlines,
- `key` is the rate-limit identity chosen by your application,
- `res` describes whether the request is allowed and how much capacity remains.

## Resource-Scoped Runtime Model

If you need different policies for different resources while keeping the same
strategy and store selection, use the additive resource-scoped API:

```go
resourceLimiter, err := gorl.NewResourceLimiter(core.ResourceConfig{
    Strategy: core.SlidingWindow,
    DefaultPolicy: core.ResourcePolicy{
        Limit:  100,
        Window: time.Minute,
    },
    Resources: map[string]core.ResourcePolicy{
        "login":  {Limit: 5, Window: time.Minute},
        "search": {Limit: 50, Window: time.Second},
    },
})
if err != nil {
    panic(err)
}
defer resourceLimiter.Close()

res, err := resourceLimiter.AllowResource(ctx, "login", "user-123")
```

Where:

- `resource` selects the policy,
- `key` selects the identity counted within that policy,
- unknown resources fall back to `DefaultPolicy`.

## Fail-Open Behavior

Set `FailOpen: true` if you prefer requests to pass when the backend is
unavailable.

```go
limiter, err := gorl.New(core.Config{
    Strategy: core.FixedWindow,
    Limit:    50,
    Window:   time.Minute,
    RedisURL: "redis://localhost:6379/0",
    FailOpen: true,
})
```

## Current Notes

GoRL keeps request key selection outside the top-level constructor.

- In direct usage, your code builds the key passed to `Allow(ctx, key)`.
- In middleware usage, the adapter's key extraction function builds the key.
- Resource-scoped usage is optional and does not change existing `v2` callers.

## Load Config from JSON or YAML

The optional `config` package can load `core.ResourceConfig` from disk:

```go
cfg, err := config.LoadResourceConfig("limits.yaml")
if err != nil {
    panic(err)
}

resourceLimiter, err := gorl.NewResourceLimiter(cfg)
if err != nil {
    panic(err)
}
```

## Next Reading

- [Middleware Guide](./middleware.md)
- [Storage and Observability](./storage-and-observability.md)
- [Distributed Semantics](../architecture/distributed-semantics.md)
- [System Overview](../architecture/system-overview.md)
