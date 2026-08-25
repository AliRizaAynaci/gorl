# Configuration files

The `config` package loads resource-scoped policies from JSON or YAML into
`core.ResourceConfig`. It does not load the simpler `core.Config` shape.

## Canonical YAML

```yaml title="examples/configuration_file/limits.yaml"
--8<-- "examples/configuration_file/limits.yaml"
```

The same fields can be expressed as JSON. Both formats accept either the
namespaced `gorl` object shown above or a flat object containing `strategy`,
`default`, and `resources` directly.

## Load and run

```go title="examples/configuration_file/main.go"
--8<-- "examples/configuration_file/main.go"
```

Run it from the repository root:

```bash
go run ./examples/configuration_file \
  -config examples/configuration_file/limits.yaml
```

Expected first decision:

```text
allowed=true limit=5 remaining=4
```

## Schema

| Field | Required | Meaning |
| --- | --- | --- |
| `strategy` | Yes | `fixed_window`, `sliding_window`, `token_bucket`, or `leaky_bucket` |
| `redis_url` | No | Selects the bundled Redis backend when non-empty |
| `fail_open` | No | Defaults to `false` |
| `default.limit` | Yes | Positive fallback capacity |
| `default.window` | Yes | Positive Go duration string |
| `resources` | No | Map of resource name to policy override |

Windows use Go duration syntax such as `250ms`, `30s`, `1m`, or `2h`. A bare
number such as `60` is invalid.

## Validation and errors

Loading fails when:

- the extension is not `.json`, `.yaml`, or `.yml`,
- the file cannot be read,
- JSON or YAML is malformed,
- a duration cannot be parsed,
- a limit or window is not positive,
- a resource name is empty.

An unknown strategy is detected later by `gorl.NewResourceLimiter`, after the
configuration document has been converted and validated.

!!! warning "Configuration is loaded once"
    `LoadResourceConfig` does not watch the file. To reload policies, the
    application must load a new config, construct a replacement limiter, route
    new traffic to it, and close the old limiter after in-flight work drains.

## Production handling

- Treat configuration as startup input and fail deployment early on invalid
  policy.
- Do not log Redis credentials embedded in `redis_url`.
- Review the default policy as carefully as named overrides because unknown
  resources silently use it.
- Keep configuration examples in tests or executable examples so field names
  cannot drift unnoticed.
