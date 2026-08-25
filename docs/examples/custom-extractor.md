# Custom key extractor

This direct-API example composes a tenant ID and user ID before calling the
limiter.

## Prerequisites

- Go 1.24 or newer
- no external service

## Run

```bash
go run ./examples/custom_extractor
```

```go title="examples/custom_extractor/main.go"
--8<-- "examples/custom_extractor/main.go"
```

## Expected behavior

`team-a:user-123` and `team-b:user-456` consume independent leaky-bucket state.
The third request for the first composite key is denied after that key has
filled its capacity.

## Production cautions

Build keys from authenticated, immutable identifiers and define an unambiguous
encoding. Avoid logging secrets or raw access tokens. For arbitrary components,
prefer a length-prefix or another collision-safe encoding over ambiguous string
concatenation.
