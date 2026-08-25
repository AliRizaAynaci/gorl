# Documentation Style Guide

This page defines a lightweight standard for future documentation additions.

## Goals

- Keep docs useful for both users and maintainers.
- Prefer accuracy over marketing language.
- Document current behavior first, future intent second.

## File Organization

- `docs/README.md` is the index page.
- `docs/index.md` is the rendered site home page.
- `docs/concepts/` explains stable mental models and semantics.
- `docs/architecture/` contains system and package-level explanations.
- `docs/guides/` contains task-oriented documentation.
- `docs/examples/` explains canonical programs stored under `examples/`.
- `docs/reference/` contains public contract summaries.
- `docs/contributing/` contains documentation process notes.

## Writing Rules

- Start each page with a short statement of purpose.
- Prefer short sections over long prose blocks.
- Keep code samples minimal and runnable where practical.
- When behavior has caveats, call them out explicitly.

## Diagram Rules

- Use Mermaid for high-level architecture and flow diagrams.
- Keep one main message per diagram.
- Prefer left-to-right or top-to-bottom layouts with stable naming.
- Add `accTitle` and `accDescr` metadata when supported by the diagram syntax.
- Verify diagrams in both theme palettes and at mobile width.
- Use SVG under `docs/assets/diagrams/` only when Mermaid cannot express the
  diagram clearly.

## Maintenance Rules

- Update docs when public behavior changes.
- Avoid copying the same explanation across many pages.
- Link to the canonical page instead of repeating large sections.
- Embed runnable programs from `examples/` with `pymdownx.snippets`; do not paste
  a second copy into the page.
- Run `go test ./...` and `mkdocs build --strict` before opening a pull request.
