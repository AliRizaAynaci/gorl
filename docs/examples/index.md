# Runnable examples

Every example is a standalone `main` package under `examples/` and is compiled
by `go test ./...`. The documentation embeds those canonical files at build
time, so the displayed program cannot silently drift from the repository.

## Choose a task

| Task | Example | External service | Typical next step |
| --- | --- | --- | --- |
| Try one process | [In-memory limiter](in-memory.md) | None | Choose a stable key |
| Share state across processes | [Redis-backed limiter](redis.md) | Redis | Review production guidance |
| Set policies by operation | [Resource-scoped limits](resource-scoped.md) | None | Design default policy |
| Compose tenant/user identity | [Custom key extractor](custom-extractor.md) | None | Normalize authenticated IDs |
| Protect standard library HTTP | [net/http](net-http.md) | None | Configure proxy trust |
| Protect Gin | [Gin](gin.md) | None | Customize key and handlers |
| Protect Fiber | [Fiber](fiber.md) | None | Customize key and handlers |
| Protect Echo | [Echo](echo.md) | None | Customize key and handlers |
| Load YAML or JSON | [Configuration file](configuration-file.md) | None | Add deployment validation |
| Export metrics | [Prometheus](prometheus.md) | None | Add alerts and dashboards |

## Compile everything

From the repository root:

```bash
go test ./...
```

This compiles every example package but does not start servers or require Redis.
Redis integration tests run when `GORL_REDIS_URL` is configured by the test
environment.

## Example contract

Each page records:

- prerequisites,
- the exact command to run,
- expected behavior,
- production cautions.

Examples favor clarity over application framework structure. Production code
should construct the limiter in application startup, inject it where needed,
and close it during graceful shutdown.
