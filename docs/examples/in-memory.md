# In-memory limiter

Use this example to see a token bucket refill without running an external
service.

## Prerequisites

- Go 1.24 or newer
- repository dependencies downloaded with `go mod download`

## Run

```bash
go run ./examples/inmemory
```

```go title="examples/inmemory/main.go"
--8<-- "examples/inmemory/main.go"
```

## Expected behavior

The bucket has capacity three over ten seconds and begins full. Early requests
are allowed, the bucket becomes empty, and later requests become allowed as
tokens refill. The program runs for about fifteen seconds and prints timing,
remaining capacity, and retry delay for every decision.

## Production cautions

!!! warning "State is not shared"
    Each process has an independent budget. Scaling from one process to three
    can expose roughly three times the intended aggregate capacity.

Reuse one limiter instead of constructing one per request, keep key cardinality
bounded, and call `Close` so the in-memory cleanup goroutine stops.
