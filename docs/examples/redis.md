# Redis-backed limiter

Use this example when rate-limit state must be visible to more than one
application process.

## Prerequisites

- Go 1.24 or newer
- Redis reachable at `localhost:6379`

For a disposable local Redis:

```bash
docker run --rm --name gorl-redis -p 6379:6379 redis:7-alpine
```

## Run

In a second terminal:

```bash
go run ./examples/redis
```

```go title="examples/redis/main.go"
--8<-- "examples/redis/main.go"
```

## Expected behavior

The constructor connects to Redis, then a token bucket with capacity four makes
fifteen decisions over roughly fifteen seconds. Multiple runs using the same
Redis database and key observe shared state.

## Production cautions

The example deliberately uses a local unauthenticated URL and `FailOpen: true`.
Do not copy those policy choices blindly. Protect credentials, measure Redis
round-trip latency, monitor evictions, and read [Redis in production](../guides/redis-production.md)
before deployment.
