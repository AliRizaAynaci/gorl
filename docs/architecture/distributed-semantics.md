# Distributed Semantics

This page defines the repository's current stance on multi-instance behavior.
It is intentionally conservative: the table below describes what the codebase
can reliably guarantee today, not what Redis could support in theory.

## Support Matrix

| Backend | Strategy | Multi-instance status | Notes |
| --- | --- | --- | --- |
| `storage/inmem` | all strategies | not applicable | State is local to one process. |
| `storage/redis` | `FixedWindow` | supported shared-state option today | Uses bucketed keys, so separate instances converge on the same window key. |
| `storage/redis` | `SlidingWindow` | not distributed-safe today | Requires multi-key state transitions that are not atomic across processes. |
| `storage/redis` | `TokenBucket` | not distributed-safe today | Refill and consume logic reads and writes multiple values over multiple steps. |
| `storage/redis` | `LeakyBucket` | not distributed-safe today | Queue drain and increment logic is not executed as one atomic state transition. |

## What "Supported Shared-State Option" Means

For `Redis + FixedWindow`, the current implementation is the best fit in this
repository for multiple application instances sharing a limiter key space.

That does not mean every Redis operation is bundled into a single script. It
means the algorithm's current model aligns with the store's available
primitives well enough that the repository documents it as the supported Redis
path today.

## What Is Not Guaranteed Today

GoRL does not currently claim strict distributed correctness for these Redis
paths:

- `SlidingWindow`
- `TokenBucket`
- `LeakyBucket`

These algorithms maintain richer state than a single shared counter and need a
stronger atomic execution path to produce reliable multi-instance behavior.

## Recommended Deployment Guidance

- Use in-memory storage only for single-process applications, development, and
  tests.
- Use Redis with `FixedWindow` when you need a supported shared-state option
  today.
- Avoid using Redis with `SlidingWindow`, `TokenBucket`, or `LeakyBucket` for
  strict multi-instance enforcement until an atomic Redis execution path lands.

## Testing Strategy

The repository currently treats distributed correctness as a separate concern
from single-process correctness.

- Unit tests validate limiter behavior in-process.
- Redis-backed algorithm safety should be proven with targeted integration
  tests under concurrent multi-instance access.
- Future Redis-specific work should verify both correctness and metadata output
  under contention.

## Planned Direction

The next step for stronger Redis semantics is a Lua-scripted execution path for
algorithms that need multi-key state transitions. That work is intentionally
separate from this document so the current guarantees remain clear even before
the implementation changes land.
