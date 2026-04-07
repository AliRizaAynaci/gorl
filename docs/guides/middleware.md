# Middleware Guide

GoRL ships with adapters for `net/http`, Gin, Fiber, and Echo.

## Common Pattern

All middleware adapters do three things:

1. extract a key from the request,
2. call `Allow(ctx, key)`,
3. write rate-limit headers and either forward or deny the request.

Resource-scoped middleware adds one more selection step:

1. extract a resource from the request,
2. extract a key from the request,
3. call `AllowResource(ctx, resource, key)`,
4. write rate-limit headers and either forward or deny the request.

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

### Resource-Scoped `net/http`

```go
resourceLimiter, _ := gorl.NewResourceLimiter(core.ResourceConfig{
    Strategy: core.SlidingWindow,
    DefaultPolicy: core.ResourcePolicy{
        Limit:  100,
        Window: time.Minute,
    },
    Resources: map[string]core.ResourcePolicy{
        "/login":  {Limit: 5, Window: time.Minute},
        "/search": {Limit: 50, Window: time.Second},
    },
})

handler := mw.RateLimitByResource(resourceLimiter, mw.Options{
    KeyFunc:      mw.KeyByIP(),
    ResourceFunc: mw.ResourceByPath(),
}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("ok"))
}))
```

### Important Note

`middleware/http` expects `Options.KeyFunc` to be provided by the caller.

## Gin

```go
r := gin.Default()
r.Use(ginmw.RateLimit(limiter))
```

If `KeyFunc` is omitted, Gin defaults to `c.ClientIP()`.
For resource-scoped limiting, `RateLimitByResource` defaults to `c.FullPath()`
when available and falls back to `c.Request.URL.Path`.

## Fiber

```go
app := fiber.New()
app.Use(fibermw.RateLimit(limiter))
```

If `KeyFunc` is omitted, Fiber defaults to `c.IP()`.
For resource-scoped limiting, `RateLimitByResource` defaults to `c.Path()`.

## Echo

```go
e := echo.New()
e.Use(echomw.RateLimit(limiter))
```

If `KeyFunc` is omitted, Echo defaults to `c.RealIP()`.
For resource-scoped limiting, `RateLimitByResource` defaults to `c.Path()`
when available and falls back to `c.Request().URL.Path`.

## Headers

Middleware adapters currently write these headers from `core.Result`:

- `RateLimit-Limit`
- `RateLimit-Remaining`
- `RateLimit-Reset` when a positive reset duration is available
- `Retry-After` when the request is denied and a positive retry delay is available

This keeps response headers aligned with reliable limiter metadata instead of
forcing zero-value duration headers into every response.

## Custom Error Handling

Each middleware package allows a custom denied or error handler so applications
can standardize response bodies and logging.
