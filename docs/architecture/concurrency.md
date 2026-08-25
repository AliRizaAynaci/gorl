<div class="gorl-lock-page" markdown>

<section class="gorl-lock-hero">
<div class="gorl-lock-hero__copy">
<p class="gorl-lock-kicker">v2.2.1 · internal concurrency</p>
<h1>One key should not stop another.</h1>
<p>
GoRL keeps same-key state transitions serialized while letting unrelated
in-process keys advance through separate lock lanes. The public API,
rate-limit semantics, and Redis state format stay unchanged. In the
measured parallel multi-key workload, median time per operation fell by
61.1% to 66.7%, depending on the algorithm.
</p>
</div>
<div class="gorl-lock-release-rail" aria-label="Concurrency implementation release path">
<div class="gorl-lock-release">
<span class="gorl-lock-release__tag">v2.2.0</span>
<strong>Mixed baseline</strong>
<span>Token and leaky buckets used one lock; sliding window still had a concurrency gap.</span>
</div>
<div class="gorl-lock-release gorl-lock-release--checkpoint">
<span class="gorl-lock-release__tag">safety checkpoint</span>
<strong>Correct first</strong>
<span>Sliding window gained serialization so concurrent requests could not over-admit.</span>
</div>
<div class="gorl-lock-release gorl-lock-release--current">
<span class="gorl-lock-release__tag">v2.2.1</span>
<strong>Correct and concurrent</strong>
<span>All three generic stateful limiters use the same bounded sharded-lock model.</span>
</div>
</div>
</section>

## From one gate to independent lanes

The safety baseline placed one mutex around each limiter's complete generic
state transition. That protected correctness, but a slow operation for
`tenant-a` also held up unrelated work for `tenant-b` and `tenant-c`.

<div class="gorl-lock-board">
<section class="gorl-lock-panel gorl-lock-panel--before" aria-label="Before: unrelated keys wait at one limiter-wide mutex">
<header>
<div>
<span class="gorl-lock-panel__eyebrow">Before · safety baseline</span>
<p class="gorl-lock-panel__name">One limiter, one gate</p>
</div>
<span class="gorl-lock-panel__state">serialized</span>
</header>
<div class="gorl-lock-diagram gorl-lock-diagram--single">
<div class="gorl-lock-requests" aria-label="Incoming keys">
<code>tenant-a</code>
<code>tenant-b</code>
<code>tenant-c</code>
</div>
<div class="gorl-lock-paths" aria-hidden="true"><span></span><span></span><span></span></div>
<div class="gorl-lock-gate gorl-lock-gate--single"><span>one<br>mutex</span></div>
<div class="gorl-lock-outcomes">
<span class="gorl-lock-outcome gorl-lock-outcome--run">running</span>
<span class="gorl-lock-outcome gorl-lock-outcome--wait">waiting</span>
<span class="gorl-lock-outcome gorl-lock-outcome--wait">waiting</span>
</div>
</div>
<p>Correct, but independent keys share the same contention point.</p>
</section>
<section class="gorl-lock-panel gorl-lock-panel--after" aria-label="After: unrelated keys advance through separate lock shards">
<header>
<div>
<span class="gorl-lock-panel__eyebrow">After · v2.2.1</span>
<p class="gorl-lock-panel__name">One limiter, 256 lanes</p>
</div>
<span class="gorl-lock-panel__state">concurrent</span>
</header>
<div class="gorl-lock-diagram gorl-lock-diagram--sharded">
<div class="gorl-lock-requests" aria-label="Incoming keys">
<code>tenant-a</code>
<code>tenant-b</code>
<code>tenant-c</code>
</div>
<div class="gorl-lock-paths" aria-hidden="true">
<span><i></i></span><span><i></i></span><span><i></i></span>
</div>
<div class="gorl-lock-gates" aria-label="Illustrative shard assignments">
<span>shard 042</span>
<span>shard 187</span>
<span>shard 091</span>
</div>
<div class="gorl-lock-outcomes">
<span class="gorl-lock-outcome gorl-lock-outcome--run">running</span>
<span class="gorl-lock-outcome gorl-lock-outcome--run">running</span>
<span class="gorl-lock-outcome gorl-lock-outcome--run">running</span>
</div>
</div>
<p>Shard numbers are illustrative; each limiter receives a process-local random hash seed.</p>
</section>
</div>

## How a key chooses its lock

<div class="gorl-lock-formula" aria-label="Key-to-lock-shard selection flow">
<div><span>input</span><code>tenant-a:user-123</code></div>
<b aria-hidden="true">→</b>
<div><span>hash</span><code>maphash.String(seed, key)</code></div>
<b aria-hidden="true">→</b>
<div><span>mask</span><code>hash &amp; 255</code></div>
<b aria-hidden="true">→</b>
<div><span>lock</span><code>mutex[index]</code></div>
</div>

The locker creates one random `maphash` seed when the generic limiter is
constructed. For that limiter and process lifetime, the same key produces the
same hash and therefore the same index.

`256` is a power of two, so `hash & 255` selects the lower eight bits and is
equivalent to `hash % 256`. Two different keys can land on the same index. That
collision is safe: those keys briefly serialize, while the rate-limit result
remains correct.

This design deliberately avoids a mutex map keyed by caller input. Memory stays
bounded at about 2 KB per generic stateful limiter and does not grow with key
cardinality. Redis-backed limiters do not allocate these locks because their
Lua transitions are already atomic.

## What remains invariant

| Invariant | Guarantee |
| --- | --- |
| Same key, one transition | Concurrent calls for one key still execute their multi-step state update serially. |
| Collisions stay safe | A shard collision can reduce parallelism, but it cannot admit requests above the configured limit. |
| Redis bypasses the locker | The bundled Redis backend continues to execute each decision in an atomic Lua script. |
| No public contract changes | Constructors, configuration, result metadata, middleware, and storage keys remain compatible. |

## Measured effect

Apple M4, Go 1.26.0, in-memory store. Ten interleaved samples per side compared
with `benchstat`; each cell is the median with its spread. `~` means the
difference is not statistically significant. The full procedure, including why
the runs are interleaved, is in
[Benchmarking methodology](../contributing/benchmarking.md).

| Benchmark | Before | After | Change |
| --- | ---: | ---: | ---: |
| Fixed window, parallel single key *(control)* | 229.8 ns ± 8% | 227.9 ns ± 10% | ~ (p=0.986) |
| Fixed window, parallel multi-key *(control)* | 95.8 ns ± 38% | 101.0 ns ± 37% | ~ (p=0.780) |
| Sliding window, parallel single key | 707.3 ns ± 24% | 749.8 ns ± 36% | ~ (p=0.353) |
| Sliding window, parallel multi-key | 787.6 ns ± 23% | 262.4 ns ± 24% | **−66.7%** (p=0.000) |
| Token bucket, parallel single key | 745.4 ns ± 21% | 758.6 ns ± 23% | ~ (p=0.971) |
| Token bucket, parallel multi-key | 823.1 ns ± 44% | 320.3 ns ± 21% | **−61.1%** (p=0.000) |
| Leaky bucket, parallel single key | 715.9 ns ± 17% | 734.8 ns ± 20% | ~ (p=0.853) |
| Leaky bucket, parallel multi-key | 813.4 ns ± 24% | 315.4 ns ± 23% | **−61.2%** (p=0.000) |

Bytes and allocations per operation are byte-identical on both sides for every
benchmark, so the change costs no additional garbage.

Three rows carry the argument:

- **The control rows do not move.** The fixed window takes no limiter-level lock,
  so this change cannot affect it. Its flat result is what makes the other rows
  trustworthy; a control that moves means the machine drifted, not that the code
  improved.
- **Single-key rows are `~`.** One key must stay serialized for correctness, and
  hashing the key to a shard costs nothing measurable.
- **Multi-key rows drop to roughly a third of their original cost.** This is the
  case the change targets: independent keys no longer queue behind each other.

Treat these figures as comparative evidence on one machine, not as a latency
guarantee for another machine or workload.

## Upgrade to v2.2.1

!!! tip "Drop-in patch release"
    ```bash
    go get github.com/AliRizaAynaci/gorl/v2@v2.2.1
    ```

    No migration is required. Applications do not need new options, different
    keys, or middleware changes. Run the normal test suite for your service and
    deploy the patch release through the same process used for `v2.2.0`.

</div>
