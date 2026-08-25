# Echo middleware

Use the Echo adapter as standard middleware around application routes.

## Prerequisites

- Go 1.24 or newer
- port `8080` available

## Run

```bash
go run ./examples/echo
```

```go title="examples/echo/main.go"
--8<-- "examples/echo/main.go"
```

```bash
for i in 1 2 3 4 5 6; do curl -i http://localhost:8080/; done
```

## Expected behavior

Echo uses `RealIP()` as the default key. Five requests spend the initial token
bucket capacity; the next request is denied until a token refills.

## Production cautions

Configure Echo's proxy/IP extraction for your network and use authenticated
identity where appropriate. Resource-aware middleware prefers the matched Echo
route path and falls back to the raw request path.
