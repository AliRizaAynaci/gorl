# Middleware Guide

GoRL ships with adapters for `net/http`, Gin, Fiber, and Echo.

## Common Pattern

All middleware adapters do three things:

1. extract a key from the request,
2. call `Allow(ctx, key)`,
3. write rate-limit headers and either forward or deny the request.

## `net/http`

```go
package main

import (
    "net/http"
    "time"

    "github.com/AliRizaAynaci/gorl/v2"
    "github.com/AliRizaAynaci/gorl/v2/core"
    mw "github.com/AliRizaAynaci/gorl/v2/middleware/http"
)

func main() {
    limiter, _ := gorl.New(core.Config{
        Strategy: core.SlidingWindow,
        Limit:    10,
        Window:   time.Minute,
    })

    handler := mw.RateLimit(limiter, mw.Options{
        KeyFunc: mw.KeyByIP(),
    }, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("ok"))
    }))

    http.ListenAndServe(":8080", handler)
}
```

### Built-in Key Extractors

- `mw.KeyByIP()`
- `mw.KeyByHeader("X-API-Key")`
- `mw.KeyByPath()`

### Important Note

`middleware/http` expects `Options.KeyFunc` to be provided by the caller.

## Gin

```go
r := gin.Default()
r.Use(ginmw.RateLimit(limiter))
```

If `KeyFunc` is omitted, Gin defaults to `c.ClientIP()`.

## Fiber

```go
app := fiber.New()
app.Use(fibermw.RateLimit(limiter))
```

If `KeyFunc` is omitted, Fiber defaults to `c.IP()`.

## Echo

```go
e := echo.New()
e.Use(echomw.RateLimit(limiter))
```

If `KeyFunc` is omitted, Echo defaults to `c.RealIP()`.

## Headers

Middleware adapters currently write these headers from `core.Result`:

- `RateLimit-Limit`
- `RateLimit-Remaining`
- `RateLimit-Reset`
- `Retry-After` when the request is denied

Because these values come directly from algorithm results, header quality
depends on the current metadata behavior of each limiter implementation.

## Custom Error Handling

Each middleware package allows a custom denied or error handler so applications
can standardize response bodies and logging.
