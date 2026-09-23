# Part 03 — The Truth

> **Prerequisites:** [01-fundamentals.md](01-fundamentals.md) and
> [02-the-buildup.md](02-the-buildup.md).

This is the part the tutorials usually leave out. It is also the part that
answers the question you probably started with: *"yes, but is it actually
faster?"*

---

## 1. None of this is assembly. It's a hint.

`vec/arch_amd64.go` and `vec/arch_arm64.go` contain **no AVX intrinsics and no NEON
intrinsics**. They are ordinary Go. You are not writing vector instructions. You
are arranging arithmetic so that the *Go compiler's* backend (SSA) notices the
shape and generates the code you want.

If the compiler is stubborn, your beautiful four-lane loop compiles to something
ordinary. **The code is a hint, not a guarantee.**

It is a strong enough hint to work here — the measured ~4× speedup in
[../benchmarks.md](../benchmarks.md) is proof the compiler did what was hoped for.
But nothing in the source *forces* it to.

## 2. The build tags are about register count, not extra power

```go
//go:build arm64
```

```go
//go:build amd64
```

```go
//go:build !arm64 && !amd64
```

The three architecture files hold *the same algorithm*, tuned to three different
amounts of spare hardware. ARM64 gets the 2-accumulator, 4-wide version; x86-64
gets the 4-accumulator, 8-wide version; everything else gets the plain loop. The
x86-64 version would still be *correct* on ARM — just tuned for the wrong machine,
and probably slower than the plain loop.

## 3. The binary is the truth, not the source

This section exists because an earlier version of this documentation got it
wrong in a way that is worth understanding.

**The mistake.** Reading the original `sum_arm64.go`, the loop body looked like
this:

```go
for i < limit {              // limit := n - 3
    s0 += xs[i] + xs[i+1]    // touches xs[i],   xs[i+1]
    s1 += xs[i+2] + xs[i+3]  // touches xs[i+2], xs[i+3]
    i += 8                   // <-- advances by 8, but only 4 consumed?!
}
```

Four elements read, but `i` advanced by eight. That looked like a bug that would
silently skip half the array, so the documentation said so.

**It was wrong.** Here is what actually executes. (Labels are exactly as `objdump`
prints them, which is why they say `arch_arm64.go` — the compiler names the file,
not the path.)

```
arch_arm64.go:44   FMOVD (R0)(R3<<3), F2      ; load xs[i]
arch_arm64.go:44   ADD $1, R3, R4
arch_arm64.go:44   FMOVD (R0)(R4<<3), F3      ; load xs[i+1]
arch_arm64.go:44   FADDD F2, F3, F2           ; xs[i] + xs[i+1]
arch_arm64.go:45   ADD $2, R3, R4
arch_arm64.go:45   FMOVD (R0)(R4<<3), F3      ; load xs[i+2]
arch_arm64.go:45   ADD $3, R3, R4
arch_arm64.go:45   FMOVD (R0)(R4<<3), F4      ; load xs[i+3]
arch_arm64.go:45   FADDD F3, F4, F3           ; xs[i+2] + xs[i+3]
arch_arm64.go:44   FADDD F1, F2, F1           ; s0 += ...
arch_arm64.go:45   FADDD F0, F3, F0           ; s1 += ...
arch_arm64.go:46   ADD $4, R3, R3             ; <-- the real increment: i += 4
arch_arm64.go:43   CMP R2, R3
                   BLT -13(PC)                ; loop back
```

The instruction that increments `i` is `ADD $4, R3, R3`. **`i += 4`.** Not 8.

Why? Because nothing in that loop body ever reads `xs[i+4]`. The `+4` written in
the source has **no observable effect** — no load, no store, no branch depends on
it. So the compiler treats it as dead arithmetic and folds it away. The loop does
exactly what it looks like it should do: consumes four elements, advances by four.

> **The source code is a *hint*. The compiled binary is the *truth*.**

That is the single most important lesson in this documentation, and it applies to
any tuned code you will ever read. You cannot audit it by reading it, because the
compiler rewrites it before it runs.

**How to check instead:**

```sh
go test -c -o /tmp/numa.test .
go tool objdump -s 'numa\.Sum' /tmp/numa.test
```

(Reproduce this yourself rather than trusting the offsets above — they depend on
the compiler version.)

And confirm it behaviourally: `TestSumKnownValues` in `vec/sum_test.go` pins the
exact result for `xs[i] = i` at lengths from 1 to 100, where the true sum
`n(n-1)/2` is exactly representable. That test exists *because* of this mistake.

**The general lesson for testing tuned code:** the tests in this package are
property-based rather than value-based wherever possible.
`TestSumConsumesEveryElement` perturbs every index in turn and requires the result
to move. That catches a skipped-block bug structurally, without anyone having to
spot the arithmetic by hand.

## 4. "Optimized" is a measurement, not a property

You cannot know which version is fastest by *reading* it. You can only know by
timing it:

```
BenchmarkNaiveSum/4194304-8    5260503 ns/op    6378.56 MB/s
BenchmarkSum/4194304-8         1326591 ns/op    25293.72 MB/s
```

**3.98× faster.** That number did not exist before `bench_test.go` was written, and
neither did any other conclusion in this documentation.

But note what else the measurement revealed, which no amount of reading would
have:

| operation | naïve | tuned | speedup |
|---|---|---|---|
| `Sum` (n=4M) | 6.4 GB/s | **25.4 GB/s** | **3.98×** |
| `Dot` (n=4M) | 9.6 GB/s | **38.4 GB/s** | **4.00×** |
| `Min` (n=4M) | 4.8 GB/s | **9.5 GB/s** | **1.97×** |
| `Add` (n=4M) | 37.0 GB/s | **47.0 GB/s** | **1.27×** |
| `Dot` (n=8) | 25.7 GB/s | 20.6 GB/s | **0.80× — slower!** |

Four things to notice, because they *are* the lesson:

1. **~4× is real** for the reductions, at every size that matters.
2. **`Add` only gained 1.27×**, not 4× — because it had no dependency chain to
   fix. The same tuning applied to a loop that wasn't broken bought almost
   nothing. This is the strongest evidence that the explanation in
   [02-the-buildup.md](02-the-buildup.md) is the correct one: if the theory were
   wrong, `Add` would have gained 4× too.
3. **`Dot` at n=8 got 25 % slower.** The tuning has a cost, and on short slices you
   pay it without collecting the benefit.
4. **`Min` gains ~2×, not ~4×**, because a compare cannot be shortened the way
   `s += a + b` shortens an add.

### Two things the measurements got wrong

This is the part worth internalising, because it happened twice.

**First:** an earlier version of this documentation said the reductions "plateau at
the memory-bandwidth ceiling", inferred from the shape of the curve. It was never
measured. When it finally was, by writing a loop with the *same shape* that moves
the same bytes but does no real arithmetic, the answer came out differently:

```
ReadOnlyTuned (4 chains, multiply by zero)   37.8 GB/s
Sum           (4 chains, real adds)          25.4 GB/s
```

Same loads, same bytes, same unrolling — 49 % apart. So `Sum` is **not**
bandwidth-bound. It is bound by floating-point add latency. The plateau was where
*this loop shape* tops out, not where the memory system does. And that error had a
consequence: it was being used to argue that SIMD would win nothing, when in fact
it is exactly what would help.

**Second:** `Min` was reported here as a failure — no faster than a naive loop —
because the NaN check was a second pass over the input. Written that way, it cost
2× and cancelled the entire benefit of the tuned scan. Folding the check into the
scan loop made it nearly free, and `Min` went from 4.68 GB/s to 9.48 GB/s.

Both mistakes had the same root cause: **a conclusion reached by reasoning about
the code instead of measuring it.** The plateau *looked* like a bandwidth ceiling.
The NaN check *looked* like it had to cost a pass. Neither survived contact with a
benchmark.

> **Reading code tells you what someone intended. Measuring tells you what
> happened.** They are different activities, and only one of them is evidence.

> **Being "optimized" is not a property a function has.** It is a relationship
> between a function and a workload. `Dot` is 3.81× faster at n=4M and 0.80×
> *slower* at n=8. Both statements are true.

Full numbers and methodology: [../benchmarks.md](../benchmarks.md).

## 5. Optimizing changed the arithmetic, too

This is the part that genuinely matters and is easy to miss.

Plain summation adds the numbers in order:

```
(((a + b) + c) + d) + e ...
```

The four-lane version adds them like a tree, then combines:

```
lane0 = pairs from the left
lane1 = different pairs
...
total = (lane0 + lane1) + (lane2 + lane3)
```

Floating-point numbers are **not associative**. `(a + b) + c` can differ from
`a + (b + c)` in the last couple of bits. So the fast version and the plain version
can return answers that differ slightly.

That's usually fine — often *better*, because the tree shape keeps big and small
numbers from mixing badly (that's the "numerically stable" claim in the README).
But it is a **real behavioural difference**, and it's the kind of thing that matters
a lot in the finance and trading domain this project targets.

> **You're not just making it faster. You're changing what "sum" means at the last
> bit of precision. That trade has to be a decision, not an accident.**

Two consequences:

* Results are **not guaranteed to be bit-identical across architectures**, because
  arm64 and x86-64 run different loop shapes by design.
* If you need a reproducible audit trail, this package **cannot provide one** as
  currently designed. That's listed as an open decision in
  [../next-steps.md](../next-steps.md).

## 6. Short lists are slower this way

All this bookkeeping only pays off when the list is long. For a three-element
slice, the plain loop wins. That's exactly why `Sum` in `vec/sum.go` checks:

```go
if len(xs) == 0 {
    return 0.0
}
```

...and why the tuned loops bother having a **tail loop** for leftovers. A tuned
implementation is only tuned *in its sweet spot*.

---

## The cheat sheet

| # | Idea | One-line why |
|---|---|---|
| 1 | CPU = robot, one instruction at a time | It's the whole mental model |
| 2 | Memory is slow, thinking is fast | So stalls are the enemy |
| 3 | Pipelines overlap instructions | But dependencies drain them |
| 4 | One accumulator = one long dependency chain | Everyone waits in line |
| 5 | Multiple accumulators | Independent chains, robot always busy |
| 6 | `s0 += a + b` not `s0 += a; s0 += b` | Halves the chain through `s0` |
| 7 | Pairing `xs[i], xs[i+1]` | Suggests one wide memory load |
| 8 | Separate registers | Writes to *different* pockets can't collide |
| 9 | 2 lanes on ARM64, 4 on x86-64 | Different machines, different register counts |
| 10 | Tree at the very end | Fewer, more parallel finishing adds |
| 11 | Tail loop + `n - 3` / `n - 7` | Consume leftovers without reading past the end |
| 12 | It's a *hint* to the compiler, not assembly | You're shaping SSA, not writing SIMD |
| 13 | It changes floating-point results | Non-associativity is real |
| 14 | **The binary is the truth, not the source** | The compiler deletes code you wrote |
| 15 | "Optimized" means *measured*, not *read* | Benchmarks or it didn't happen |
| 16 | Know what the loop *waits on* first | Reductions ≠ scans ≠ elementwise maps |
| 17 | Tuned code can be *slower* when short | n=8 `Dot` is 0.80× — measure your sizes |

---

## The one sentence

If you forget everything else:

> **A modern CPU is a very fast worker that spends most of its time waiting — for
> memory, and for the instruction right before it. Optimizing code is almost
> always just re-arranging the work so it has something else to do while it
> waits.**

Read `vec/arch_arm64.go` again with that sentence in your head. Every odd-looking line
in it is an answer to the question: *"what could the robot do while it's waiting
for the last thing I told it to do?"*

That's it. That's why this code looks like this.

---

## Where to go next

| document | purpose |
|---|---|
| [../design.md](../design.md) | Reference: the dispatch pattern, how to add an operation, invariants |
| [../benchmarks.md](../benchmarks.md) | Raw measurements and reproduction commands |
| [../next-steps.md](../next-steps.md) | What is verified, what is open, what needs deciding |
| [../../README.md](../../README.md) | The API surface |
