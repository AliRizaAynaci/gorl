# Documentation Style Guide

This page defines a lightweight standard for future documentation additions.

## Goals

- Keep docs useful for both users and maintainers.
- Prefer accuracy over marketing language.
- Document current behavior first, future intent second.

## File Organization

- `docs/README.md` is the index page.
- `docs/architecture/` contains system and package-level explanations.
- `docs/guides/` contains task-oriented documentation.
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

## Maintenance Rules

- Update docs when public behavior changes.
- Avoid copying the same explanation across many pages.
- Link to the canonical page instead of repeating large sections.
