# Configuration-file example

Load resource policies from a version-controlled YAML or JSON document.

## Prerequisites

- Go 1.24 or newer
- no external service for the bundled YAML, because it omits `redis_url`

## Run

```bash
go run ./examples/configuration_file \
  -config examples/configuration_file/limits.yaml
```

```yaml title="examples/configuration_file/limits.yaml"
--8<-- "examples/configuration_file/limits.yaml"
```

```go title="examples/configuration_file/main.go"
--8<-- "examples/configuration_file/main.go"
```

## Expected behavior

The file selects sliding-window behavior and a five-per-minute `login` policy.
The program prints an allowed first decision with four remaining requests.

## Production cautions

The loader reads once and does not watch the file. Validate configuration during
deployment, protect Redis credentials if a URL is included, and design an
explicit limiter replacement lifecycle before attempting live reloads.
