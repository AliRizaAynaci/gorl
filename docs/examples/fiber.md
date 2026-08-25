# Fiber middleware

Use the Fiber adapter to apply a limiter before application handlers.

## Prerequisites

- Go 1.24 or newer
- port `3000` available

## Run

```bash
go run ./examples/fiber
```

```go title="examples/fiber/main.go"
--8<-- "examples/fiber/main.go"
```

```bash
for i in 1 2 3 4 5 6; do curl -i http://localhost:3000/; done
```

## Expected behavior

Fiber uses `c.IP()` as the default key. Five requests are allowed in a thirty-
second fixed window; the next receives a JSON 429 response.

## Production cautions

Verify Fiber proxy settings and forwarded-IP behavior in the actual deployment.
The resource-scoped default uses `c.Path()`; normalize high-cardinality paths or
supply a stable `ResourceFunc` when URLs contain identifiers.
