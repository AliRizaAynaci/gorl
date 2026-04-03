# Request Lifecycle

This page shows the typical execution path for a request that goes through a
GoRL middleware adapter and into a limiter implementation.

## Middleware-to-Limiter Flow

```mermaid
sequenceDiagram
    participant Client
    participant Middleware
    participant Limiter as core.Limiter
    participant Algorithm
    participant Storage
    participant Metrics
    participant Handler

    Client->>Middleware: HTTP request
    Middleware->>Middleware: Extract rate-limit key
    Middleware->>Limiter: Allow(ctx, key)
    Limiter->>Algorithm: Strategy-specific Allow
    Algorithm->>Storage: Get / Set / Incr
    Storage-->>Algorithm: Current state
    Algorithm->>Metrics: Record latency / allow / deny
    Algorithm-->>Limiter: core.Result
    Limiter-->>Middleware: core.Result
    Middleware->>Middleware: Set RateLimit headers

    alt Allowed
        Middleware->>Handler: Forward request
        Handler-->>Client: Normal response
    else Denied
        Middleware-->>Client: 429 response
    else Limiter error
        Middleware-->>Client: 500 or custom error response
    end
```

## Step-by-Step Behavior

1. The middleware extracts a key from the incoming request.
2. The middleware calls `Allow(ctx, key)` on the configured limiter.
3. The algorithm reads and updates state through `storage.Storage`.
4. The algorithm builds a `core.Result`.
5. The middleware writes headers from that result.
6. The request is either forwarded, denied, or failed with an internal error.

## Important Implementation Notes

- `middleware/http` requires `Options.KeyFunc`; the framework-specific
  middleware packages provide a default key extractor when one is omitted.
- `FailOpen` and `FailClose` behavior is enforced inside the algorithm layer.
- The in-memory store handles TTL cleanup internally with a background GC loop.
- Redis-backed behavior depends on the storage backend plus the algorithm's
  state transition strategy.
