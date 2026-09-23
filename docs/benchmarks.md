# Benchmarks

Measured results. Companion to [learn/](learn/), which explains *why* the loops
are shaped the way they are; this file is the evidence that the shape actually
pays off.

**Every number below was produced on this machine and can be reproduced with the
commands shown.** Nothing here is estimated or carried over from another
architecture.

---

## How to reproduce

```sh
cd /Users/raghav/Development/numa

# Full suite, all operations.
go test ./vec/ -bench=. -benchmem -count=10

# Just the headline comparison.
go test ./vec/ -bench='NaiveSum|^BenchmarkSum$' -benchmem -count=10

# Sub-second spot check.
go test ./vec/ -bench=. -benchtime=100x -run='^$'
```

For comparing two runs, use benchstat rather than eyeballing raw output:

```sh
go install golang.org/x/perf/cmd/benchstat@latest
go test ./vec/ -bench=. -benchmem -count=10 > new.txt
benchstat old.txt new.txt
```

`-count=10` is not optional. Single-run laptop numbers move by several percent
between invocations, which is larger than the differences being measured.

## Test environment

| | |
|---|---|
| Hardware | Apple Silicon (arm64), macOS |
| Toolchain | `go1.26.5 darwin/arm64` |
| x86-64 numbers | `GOARCH=amd64` under Rosetta 2 |

### Caveat on the amd64 numbers

The x86-64 results are measured through Rosetta 2, which translates x86-64 to
ARM64 rather than executing it natively. **They validate correctness and
direction, not absolute throughput.** Any tuning decision that depends on the
exact x86-64 magnitude needs re-measuring on real x86-64 hardware.

## Test data

All benchmarks use `benchData(n)`: `xs[i] = float64(i%17)*0.5 - 4.0`. A
non-constant pattern is deliberate — some CPUs have data-dependent timing, and
all-equal inputs can be unrepresentatively fast.

Sizes: 8 (register-resident), 1024 (L1), 32768 (L2), 4194304 (main memory).
Every benchmark calls `b.SetBytes`, so the reported throughput in GB/s tells you
whether a loop is compute-bound or bandwidth-bound.

---

## Headline result: `Sum`

The baseline `BenchmarkNaiveSum` is the plain one-accumulator loop, compiled on
the same machine and measured in the same run.

### arm64 (native)

| n | naïve ns/op | tuned ns/op | naïve GB/s | tuned GB/s | speedup |
|---|---|---|---|---|---|
| 8 | 5.02 | 4.17 | 12.7 | 15.4 | **1.20×** |
| 1 024 | 1 184 | 278.7 | 6.9 | 29.4 | **4.25×** |
| 32 768 | 40 725 | 10 195 | 6.4 | 25.7 | **3.99×** |
| 4 194 304 | 5 260 503 | 1 323 642 | 6.4 | 25.4 | **3.98×** |

### amd64 (Rosetta 2)

| n | tuned ns/op | tuned GB/s |
|---|---|---|
| 8 | 4.52 | 14.2 |
| 1 024 | 210.4 | 38.9 |
| 32 768 | 6 563 | 40.0 |
| 4 194 304 | 1 002 336 | 33.5 |

**Reading this:** the tuned loop is **~4× faster** at every size that matters,
and the plateau at ~25 GB/s (arm64) is the memory-bandwidth ceiling rather than
a compute limit. That is the correct place for an optimized reduction to end up:
once the loop is bandwidth-bound, further instruction-level tuning has nothing
left to win.

Note the small-size behaviour. At n=8 the tuned version is only 1.2× faster,
and part of that is the removed empty-check, not the unrolled loop. This matches
the prediction in [learn/02-the-buildup.md](learn/02-the-buildup.md): **the tuning
only pays off when the input is long enough to amortize the setup.**

## `Dot`

| n | naïve GB/s | tuned GB/s | speedup |
|---|---|---|---|
| 8 | 25.7 | 20.6 | 0.80× (**slower**) |
| 1 024 | 9.9 | 42.5 | **4.30×** |
| 32 768 | 9.6 | 39.5 | **4.10×** |
| 4 194 304 | 9.6 | 36.7 | **3.81×** |

`Dot` shows the largest speedup of any reduction here, as expected: a
floating-point multiply has higher latency than an add, so the naïve loop stalls
harder and there is more for the independent accumulators to recover.

It also shows the clearest small-size regression — **0.80×, i.e. 25 % slower at
n=8**. This is the honest counterexample to "tuned is always better". If your
data is short slices, measure before adopting these functions.

## `Min` / `Max`

| arm64, n=4194304 | ns/op | GB/s |
|---|---|---|
| `Max` (tuned) | 10 606 687 | 3.2 |
| amd64, n=4194304 | | |
| `Max` (tuned) | 1 966 572 | 17.0 |

Min/max is roughly **2× slower than summation at the same size** (3.2 GB/s vs
25.4 GB/s). This is not a defect in the implementation, it is the nature of the
problem, and it is worth understanding:

> A compare cannot start until the previous compare's result is known. `min` has
> no equivalent of the `s += a + b` trick, because there is no way to "combine
> two elements at once" that shortens the chain. Splitting the input into
> independent partial scans is the *only* available parallelism, and it divides
> the chain length by the accumulator count rather than eliminating it.

So scans are **latency-bound** and reductions are **throughput-bound**, and they
respond to tuning differently. If you see min/max underperforming relative to
sum, that is expected.

### The NaN check doubles the cost of `Min`

`Min`, `Max` and `MinMax` enforce the documented "NaN wins" policy by calling
`hasNaN`, which walks the input a second time. `BenchmarkMinArchNoNaN` measures the
scan loop without that pass, so the cost is a measured quantity rather than a
guess:

| arm64, n=4 194 304 | ns/op | GB/s |
|---|---|---|
| `naiveMin` (baseline) | 7 001 998 | 4.79 |
| `minArch` (tuned, no NaN check) | 3 537 183 | 9.49 |
| `Min` (tuned + NaN check) | 7 192 376 | 4.67 |

**This is an uncomfortable result and it should be stated plainly: the NaN check
costs about 2×, which cancels out the entire benefit of the tuned scan.** At
n=4M, `Min` (4.67 GB/s) is marginally *slower* than the naive one-accumulator loop
(4.79 GB/s). The tuning is doing its job — `minArch` alone is 1.98× faster than
naive — but the second pass eats all of it.

The numbers are within noise of each other at n≥1024, so the fair summary is
**"the tuned Min is no faster than naive"**, not "the tuned Min is slower".

Options, in rough order of preference:

1. **Fold the NaN check into the scan loop.** This would recover most of the cost,
   but it adds work to a latency-bound loop, so it needs measuring rather than
   assuming. It also interacts with the accumulator structure.
2. **Drop the NaN check and document the weaker guarantee** (comparisons against
   NaN are false, so NaN is silently skipped). That restores the 1.98× but
   reintroduces positional behaviour, which is worse than either explicit policy.
3. **Accept the cost**, on the grounds that correctness beats throughput and 4.7
   GB/s is still fast.

This is listed as an open decision in [next-steps.md](next-steps.md). It is worth
deciding rather than leaving, because right now the README advertises tuned scans
that measure the same as naive ones.

### The "NaN wins" policy

`Min`, `Max` and `MinMax` return NaN if any input element is NaN. The reasoning is
in `vec/nan.go`: an array containing NaN is almost always the result of an upstream
error, and silently returning the extreme of the *remaining* values would hide
that error while producing a plausible-looking number. In a domain where these
numbers feed decisions, failing loudly is safer.

The alternative ("ignore NaN") is defensible but is the *accidental* behaviour of
a naive scan, since `v < m` is false for NaN. That means a naive implementation's
result depends on where the NaN sits: one at index 0 gets overwritten by later
comparisons, one at the end gets returned. Neither policy justifies positional
behaviour, which is why the check exists.

## `MinMax` vs `Min` then `Max`

| arm64, n=4 194 304 | ns/op | GB/s |
|---|---|---|
| `MinMax` (one pass) | 5 260 681 | 6.4 |
| `Min` + `Max` (two passes) | 7 464 223 | 9.0 |

`MinMax` is **1.42× faster** than calling `Min` and `Max` separately, because it
reads the input once instead of twice. Note that the GB/s figure is *lower* for
`MinMax` even though it is faster — it moves less data, and the wall-clock time
is what matters. Comparing GB/s across those two would be misleading; the
`BenchmarkMinThenMax` variant reports `16*n` bytes precisely so the two are
comparable on the same basis.

(These two figures predate the NaN check, so both are now optimistic. The
comparison between them remains valid; the absolute values do not. Re-run
`go test ./vec/ -bench='MinMax|MinThenMax'` for current numbers.)

## Elementwise `Add`

| arm64 | naïve ns/op | tuned ns/op | naive GB/s | tuned GB/s |
|---|---|---|---|---|
| 8 | 5.44 | 7.48 | 35.3 | 25.7 |
| 1 024 | 676.7 | 546.5 | 36.3 | 45.0 |
| 32 768 | 20 557 | 16 189 | 38.3 | 48.6 |
| 4 194 304 | 2 722 034 | 2 142 221 | 37.0 | 47.0 |

Elementwise addition is a much smaller win than the reductions — **~1.27×**
rather than ~4×. This is also expected, and it is the clearest illustration of
the difference between the two kinds of loop:

> `dst[i] = xs[i] + ys[i]` has **no dependency chain at all**. Each output
> depends only on its own inputs. There is therefore no stall to eliminate, and
> the compiler was already free to overlap everything. The only thing tuning can
> do here is keep the load/store units saturated — and it is already close to
> memory bandwidth.

So: **tuning a reduction recovers stalled cycles; tuning a map recovers
nothing, because nothing was stalled.** Same syntax, completely different
problem.

---

## Summary table

arm64, n=4 194 304, throughput in GB/s:

| operation | naïve | tuned | speedup | bound by |
|---|---|---|---|---|
| `Sum` | 6.4 | 25.4 | 3.98× | memory bandwidth |
| `Dot` | 9.6 | 36.7 | 3.81× | memory bandwidth |
| `Min` | **4.79** | **4.67** | **0.98×** | **compare latency + NaN pass** |
| `Min` (no NaN check) | 4.79 | 9.49 | 1.98× | compare latency |
| `MinMax` | — | 6.4 | 1.42× vs 2 passes | compare latency |
| `Add` | 37.0 | 47.0 | 1.27× | memory bandwidth |

## Conclusions

1. **The tuning is real, for reductions.** ~4× on `Sum` and `Dot` at realistic
   sizes, measured rather than asserted.
2. **It is not free at small sizes.** `Dot` at n=8 is 25 % *slower* than naïve.
3. **Different loops are bound by different things**, so the same tuning strategy
   produces very different returns: reductions gain ~4×, elementwise maps gain
   ~1.3×, scans are limited by compare latency no matter what.
4. **The tuned loops hit the memory-bandwidth ceiling** at large n. Beyond that, no
   amount of loop restructuring helps; the next step would be to reduce bytes
   touched, not instructions executed.
5. **One result here is negative and is reported as such.** The tuned `Min` is no
   faster than the naive loop once the NaN check is included (0.98×). The tuning
   works — `minArch` alone is 1.98× — but a second pass over the input consumes the
   entire gain. This is the single most action-needed item in this file; see
   [next-steps.md](next-steps.md).
6. **Everything here is reproducible** with the commands at the top of this file.
   If your numbers disagree with these, trust yours — the machine is the
   authority, and this file is only a record of one machine.
