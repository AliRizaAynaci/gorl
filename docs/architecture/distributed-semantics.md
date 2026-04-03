# Distributed Semantics

This page defines the repository's current stance on multi-instance behavior.
It is intentionally conservative: the table below describes what the codebase
guarantees today when applications use the built-in Redis backend.

## Support Matrix

| Backend | Strategy | Multi-instance status | Notes |
| --- | --- | --- | --- |
| `storage/inmem` | all strategies | not applicable | State is local to one process. |
| `storage/redis` | `FixedWindow` | supported atomic shared-state path | Uses an atomic Lua counter+TTL transition. |
| `storage/redis` | `SlidingWindow` | supported atomic shared-state path | Uses a Lua-scripted multi-key state transition. |
| `storage/redis` | `TokenBucket` | supported atomic shared-state path | Uses a Lua-scripted refill+consume transition. |
| `storage/redis` | `LeakyBucket` | supported atomic shared-state path | Uses a Lua-scripted drain+enqueue transition. |

## What "Supported Atomic Shared-State Path" Means

When `gorl.New` selects `storage/redis`, each built-in limiter decision is
executed in Redis as one atomic operation.

- `FixedWindow` uses an atomic counter script.
- `SlidingWindow`, `TokenBucket`, and `LeakyBucket` use algorithm-specific Lua
  scripts.
- Multi-key scripts use Redis hash tags so the related keys stay in the same
  hash slot.

## Scope Of The Guarantee

These guarantees assume:

- all application instances talk to the same Redis deployment,
- Redis provides normal script atomicity for the target keys,
- the application uses the built-in `storage/redis` backend rather than a
  custom store.

This page does not claim to characterize replica lag, cross-region topologies,
or custom failover behavior outside Redis' normal command guarantees.

## Recommended Deployment Guidance

- Use in-memory storage only for single-process applications, development, and
  tests.
- Use Redis when you need a shared limiter state across application instances.
- Prefer the built-in constructor path so the limiters can detect the Redis
  store's atomic script capability automatically.
- Treat custom storage backends as separate integrations with their own
  correctness story.

## Testing Strategy

The repository currently treats distributed correctness as a separate concern
from single-process correctness.

- Unit tests validate limiter behavior in-process.
- Redis-backed safety is verified with targeted integration tests under
  concurrent multi-instance access.
- Metadata behavior is verified for the Redis script path as well as the
  generic in-process path.
- Benchmarking should compare Redis paths before and after script changes using
  the same container, same benchmark command, and repeated runs.
