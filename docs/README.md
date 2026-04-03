# GoRL Documentation

This directory contains library-focused documentation for GoRL.

## Start Here

- [Getting Started](./guides/getting-started.md)
- [System Overview](./architecture/system-overview.md)
- [Request Lifecycle](./architecture/request-lifecycle.md)
- [Middleware Guide](./guides/middleware.md)
- [Storage and Observability](./guides/storage-and-observability.md)
- [Public API Reference](./reference/public-api.md)
- [Package Map](./architecture/package-map.md)

## Recommended Reading Order

1. Read [Getting Started](./guides/getting-started.md) for installation and the
   core runtime model.
2. Read [System Overview](./architecture/system-overview.md) to understand how
   the packages connect.
3. Read [Middleware Guide](./guides/middleware.md) if you plan to use GoRL in a
   web service.
4. Read [Storage and Observability](./guides/storage-and-observability.md) if
   you need Redis or Prometheus integration.
5. Use [Public API Reference](./reference/public-api.md) as the package-level
   lookup page while implementing.

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
