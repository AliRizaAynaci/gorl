# Resource-scoped limits

Use one strategy and backend while assigning different capacities to named
application operations.

## Prerequisites

- Go 1.24 or newer
- no external service

## Run

```bash
go run ./examples/resource_scoped
```

```go title="examples/resource_scoped/main.go"
--8<-- "examples/resource_scoped/main.go"
```

## Expected behavior

The `login` resource allows five requests per minute and denies the sixth for
`user-123`. The `search` resource has its own, larger policy and separate state.
Any resource not listed in `Resources` uses `DefaultPolicy`.

## Production cautions

Resource names are application contracts. Prefer stable identifiers such as
route patterns or operation names, not raw URLs containing IDs. Review the
fallback policy: a typo in a resource name selects the default rather than
returning an unknown-resource error.
