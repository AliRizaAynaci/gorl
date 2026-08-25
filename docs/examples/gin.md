# Gin middleware

Use the Gin adapter as a global or route-group middleware.

## Prerequisites

- Go 1.24 or newer
- port `8080` available

## Run

```bash
go run ./examples/gin
```

```go title="examples/gin/main.go"
--8<-- "examples/gin/main.go"
```

```bash
for i in 1 2 3 4 5 6; do curl -i http://localhost:8080/; done
```

## Expected behavior

Gin uses `ClientIP()` as the default key. Five requests are allowed per thirty-
second sliding window; the next request receives a JSON 429 response.

## Production cautions

Configure Gin's trusted proxies to match the deployment, and consider a custom
key based on authenticated client identity. For resource-scoped middleware,
GoRL prefers `FullPath()` so parameterized routes share the intended policy.
