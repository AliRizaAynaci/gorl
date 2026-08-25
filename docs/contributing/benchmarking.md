# Benchmarking Methodology

This page defines how GoRL benchmark numbers are produced. Every performance
figure published in the README or in `docs/` follows this procedure, so a reader
can reproduce it and a reviewer can challenge it.

## Tooling

Comparisons use [`benchstat`][benchstat], the standard Go tool for summarizing
benchmark samples:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

`benchstat` reports the median of the samples together with the spread, and
marks differences that are not statistically significant as `~`. Do not compute
averages by hand, and do not quote a single run.

[benchstat]: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat

## Reporting rules

- Report the **median** of at least 10 samples. One run is noise.
- Always name the machine, the Go version, and the exact command.
- For an A/B claim, name the **two git refs** that were compared.
- If `benchstat` prints `~`, the honest wording is "no measurable change", not
  a percentage.
- Treat every figure as comparative evidence on one machine, never as a latency
  guarantee.

## The control group rule

At least one benchmark in every comparison must be one the change **cannot**
affect. `BenchmarkFixedWindow_*` is the standing control for anything touching
the generic limiter path: the fixed window uses the store's atomic `Incr` and
takes no limiter-level lock.

If the control moves between the two sides, the machine drifted (thermal
throttling, background load) and the run is invalid. This is not hypothetical:
an uninterleaved run on an Apple M4 showed the lock-free fixed window
"regressing" by 70%, which was entirely machine state.

## The interleaving rule

Do not collect all `before` samples and then all `after` samples. Alternate them
so drift hits both sides equally:

```bash
for round in $(seq 1 10); do
  (cd "$BEFORE_TREE" && go test ./internal/algorithms \
    -run=^$ -bench="$PATTERN" -benchmem -benchtime=1s -count=1 >> before.txt)
  go test ./internal/algorithms \
    -run=^$ -bench="$PATTERN" -benchmem -benchtime=1s -count=1 >> after.txt
done

benchstat before.txt after.txt
```

Build `$BEFORE_TREE` with `git worktree add` at the baseline ref. When the
baseline predates a benchmark you want to compare, copy the current benchmark
harness (`benchmark_helpers_test.go`) into it so both sides run identical
measurement code and only the implementation differs.

## In-memory benchmarks

```bash
go test ./internal/algorithms \
  -run=^$ \
  -bench='^Benchmark(FixedWindow|SlidingWindow|TokenBucket|LeakyBucket)_' \
  -benchmem \
  -benchtime=1s \
  -count=10
```

Benchmark shapes and what each one answers:

| Suffix | Question it answers |
| --- | --- |
| `_SingleKey` | Sequential cost of one decision. |
| `_MultiKey` | Sequential cost across 1024 distinct keys. |
| `_DeniedSingleKey` | Cost of the rejection path, which skips the write. |
| `_ParallelSingleKey` | Contention cost when every goroutine shares one key. |
| `_ParallelMultiKey` | Whether independent keys can use available CPU. |

The parallel pair is the meaningful one for concurrency work. `ParallelSingleKey`
is the price of correctness — one key must stay serialized — so it is expected
to stay flat. `ParallelMultiKey` is where a concurrency change should show up.

## Redis benchmarks

Redis benchmarks need a reachable server and are addressed by the `Redis_`
prefix:

```bash
docker run --rm -d -p 6379:6379 --name gorl-bench redis:7-alpine

GORL_REDIS_URL=redis://127.0.0.1:6379/0 \
go test ./internal/algorithms \
  -run=^$ \
  -bench='^BenchmarkRedis_' \
  -benchmem \
  -benchtime=1s \
  -count=10

docker rm -f gorl-bench
```

Record the Redis server version alongside the results. Redis figures are
dominated by round-trip latency, so they are only comparable within one host and
one network path; never compare them against the in-memory table.

## What CI measures

The `Run algorithm benchmarks` step in `.github/workflows/ci.yml` executes the
suite with `-count=3` and uploads `benchmark-results.txt` as an artifact. That
artifact is a **trend signal**, not a comparison basis: it is a shared runner
with no control over neighbors, and `-count=3` is below the reporting threshold
above. Published numbers come from a local run following this page.
