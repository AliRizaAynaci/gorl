# 60-second quickstart

This path uses the in-memory backend, so it needs only Go and the GoRL module.

## 1. Install

```bash
go get github.com/AliRizaAynaci/gorl/v2
```

## 2. Run the canonical example

From the repository root:

```bash
go run ./examples/inmemory
```

The complete program is compiled by `go test ./...`:

```go title="examples/inmemory/main.go"
--8<-- "examples/inmemory/main.go"
```

## 3. Read the decisions

The token bucket begins full with three tokens. The first requests consume that
capacity; later requests are denied until time has replenished enough tokens.
Each line reports:

- `allowed`: whether application work may proceed,
- `remaining`: whole-request capacity after the decision,
- `retry_after`: the earliest reliable delay after a denial,
- `err`: a backend or algorithm failure, if one occurred.

## Move toward production

Before deploying, decide:

1. which application identity becomes the [key](../concepts/keys-and-resources.md),
2. which [algorithm](../concepts/algorithms.md) matches the traffic contract,
3. whether state is process-local or [shared through Redis](redis-production.md),
4. whether a backend failure should [fail open or fail closed](failure-policy.md),
5. where `Close` belongs in the application shutdown path.

!!! warning "Process-local means process-local"
    Two application instances with in-memory limiters enforce two independent
    capacities. Use Redis when the limit must be shared.
