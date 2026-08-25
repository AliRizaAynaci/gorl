# Contributing

Contributions to code, examples, and documentation are welcome. Read the
repository-level [CONTRIBUTING.md](https://github.com/AliRizaAynaci/gorl/blob/main/CONTRIBUTING.md)
for the complete workflow and the [documentation style guide](docs-style.md) for
site-specific conventions.

## Local checks

```bash
go test ./...
python -m pip install -r requirements-docs.txt
mkdocs build --strict
mkdocs serve
```

Open `http://127.0.0.1:8000/gorl/` when MkDocs reports the project subpath, or
use the URL printed by the development server.

## Documentation changes

When adding or changing a runnable example:

1. keep the canonical program under `examples/`,
2. embed it in the relevant page with `pymdownx.snippets`,
3. include prerequisites, a run command, expected behavior, and production
   cautions,
4. run `go test ./...` so every example package compiles,
5. run `mkdocs build --strict` so missing snippets and broken internal links fail.

## Architecture changes

Update the system overview, request lifecycle, package map, or distributed
semantics whenever a code change invalidates their model. Mermaid diagrams must
remain legible on narrow screens and include an accessible title and description
when the syntax supports them.
