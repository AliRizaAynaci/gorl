# Prometheus metrics

Expose allow, deny, and limiter-latency metrics beside a protected endpoint.

## Prerequisites

- Go 1.24 or newer
- port `8080` available

## Run

```bash
go run ./examples/prometheus
```

```go title="examples/prometheus/main.go"
--8<-- "examples/prometheus/main.go"
```

Generate decisions, then inspect metrics:

```bash
curl http://localhost:8080/api/
curl http://localhost:8080/metrics | grep '^gorl_example_'
```

## Expected behavior

Requests to `/api/` update:

```text
gorl_example_allow_total
gorl_example_deny_total
gorl_example_request_duration_seconds
```

The `/metrics` endpoint itself is not protected in this example.

## Production cautions

Register each collector name once; the helper uses the default registry and
panics on duplicate registration. Restrict metrics endpoint access at the
network or application layer. Monitor Redis separately because the built-in
collector does not expose backend availability or error counters.
