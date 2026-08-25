# `net/http` middleware

Protect standard-library handlers while keeping key extraction and denial
responses configurable.

## Prerequisites

- Go 1.24 or newer
- port `8080` available

## Run

```bash
go run ./examples/http
```

```go title="examples/http/main.go"
--8<-- "examples/http/main.go"
```

Exercise the endpoint from another terminal:

```bash
for i in 1 2 3 4 5 6; do curl -i http://localhost:8080/api/demo; done
```

## Expected behavior

The first five requests from one IP are allowed inside the thirty-second sliding
window. The sixth receives HTTP 429. Responses include limit and remaining
headers; positive reset and retry durations add their corresponding headers.

## Production cautions

`KeyByIP()` reads forwarding headers without validating the proxy. Normalize
them at a trusted edge or provide a topology-aware key function. Always provide
`Options.KeyFunc`; it is required by the standard-library adapter.
