# Troubleshooting

Start from the observed decision, then inspect identity, policy, backend, and
middleware in that order.

## Every caller shares one limit

**Symptoms:** unrelated users consume the same capacity.

**Checks:**

- Log a hashed or non-sensitive representation of the key.
- Verify required authentication headers exist before `KeyByHeader` runs.
- Remember that a missing header produces the empty key.
- Check whether a constant key was accidentally used.

See [Keys and resources](../concepts/keys-and-resources.md).

## Clients bypass an IP limit

**Symptoms:** clients rotate a forwarding header or appear under unexpected IPs.

`middleware/http.KeyByIP()` trusts `X-Forwarded-For` and `X-Real-Ip` values. Make
the edge proxy strip client-supplied forwarding headers and write canonical
ones, or provide a custom `KeyFunc` that trusts only your network topology.

## Limits multiply after scaling out

**Symptoms:** two replicas allow roughly twice the configured traffic.

The in-memory backend is local to each process. Configure the same `RedisURL` on
every instance when the budget must be shared.

## Redis construction fails even with fail-open

**Symptoms:** application startup fails on `gorl.New` while `FailOpen` is true.

Fail-open applies to runtime storage operations. The Redis constructor still
parses the URL and pings Redis. Check DNS, credentials, database selection,
network policy, TLS requirements, and the two-second startup timeout.

## Requests return HTTP 500 instead of 429

HTTP 500 means the limiter returned an execution error under fail-closed policy.
HTTP 429 means the limiter successfully evaluated the policy and denied the
request. Add a custom error handler to log the underlying error without exposing
credentials or backend details to clients.

## `RateLimit-Reset` is missing

The middleware emits timing headers only for positive durations. A missing
header means the result did not contain a positive reliable duration. Do not
replace it with an assumed zero.

## Prometheus registration panics

`RegisterPrometheusCollectors` uses `prometheus.MustRegister` on the default
registry. Registering another collector with the same namespace, subsystem, and
metric names in the same process panics.

Construct and register the collector once during startup. Tests that create
multiple applications in one process should isolate registries or avoid repeated
global registration.

## Configuration file is rejected

- Confirm the extension is `.json`, `.yaml`, or `.yml`.
- Use Go duration strings such as `30s` or `1m`.
- Set positive `limit` and `window` values for the default and every override.
- Do not use an empty resource name.
- Confirm the document is either flat or nested once under `gorl`.

## Behavior changes during a rolling deployment

Check that all replicas use the same strategy, policy, key construction, and
resource naming. Strategy changes use a different state prefix, and mixed policy
values can make shared Redis state appear inconsistent during the rollout.

## Minimal diagnostic record

For a reproducible report, capture:

- GoRL version and Go version,
- strategy, limit, window, and failure policy,
- backend type without credentials,
- single-instance or multi-instance topology,
- normalized resource and non-sensitive key shape,
- `core.Result`, returned error, and relevant response headers,
- a minimal runnable example or failing test.
