# GoRL Documentation

This directory contains the English Markdown source for the searchable
[GoRL documentation site](https://alirizaaynaci.github.io/gorl/). The files also
remain readable directly from a clone when working offline.

## Start Here

- [Getting Started](./guides/getting-started.md)
- [60-second Quickstart](./guides/quickstart.md)
- [Choose an Algorithm](./concepts/algorithms.md)
- [Keys and Resources](./concepts/keys-and-resources.md)
- [Results and HTTP Headers](./concepts/results-and-headers.md)
- [System Overview](./architecture/system-overview.md)
- [Distributed Semantics](./architecture/distributed-semantics.md)
- [Request Lifecycle](./architecture/request-lifecycle.md)
- [Middleware Guide](./guides/middleware.md)
- [Storage and Observability](./guides/storage-and-observability.md)
- [Redis in Production](./guides/redis-production.md)
- [Troubleshooting](./guides/troubleshooting.md)
- [Runnable Examples](./examples/index.md)
- [Public API Reference](./reference/public-api.md)
- [Package Map](./architecture/package-map.md)

## Recommended Reading Order

1. Read [Getting Started](./guides/getting-started.md) for installation and the
   core runtime model.
2. Read [System Overview](./architecture/system-overview.md) to understand how
   the packages connect.
3. Read [Distributed Semantics](./architecture/distributed-semantics.md) before
   using Redis in a multi-instance deployment.
4. Read [Middleware Guide](./guides/middleware.md) if you plan to use GoRL in a
   web service.
5. Read [Storage and Observability](./guides/storage-and-observability.md) if
   you need Redis or Prometheus integration.
6. Use [Public API Reference](./reference/public-api.md) as the package-level
   lookup page while implementing.

## Build Locally

From the repository root:

```bash
python3 -m venv .venv-docs
source .venv-docs/bin/activate
python -m pip install --requirement requirements-docs.txt
mkdocs serve
```

Validate exactly as CI does:

```bash
go test ./...
mkdocs build --strict
```

## Documentation Conventions

- Architecture pages prefer Mermaid diagrams for high-level flows.
- Guides describe current repository behavior as implemented today.
- When runtime behavior and public API do not fully align, the docs call that
  out explicitly instead of smoothing over the gap.
- Keep examples small and executable where possible.

## Intended Audience

This documentation is written for:

- application developers integrating the library,
- maintainers changing internal behavior,
- contributors needing a fast package-by-package orientation.
