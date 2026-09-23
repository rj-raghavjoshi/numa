// Package vec provides hardware-aware numerical primitives for high-throughput
// time-series analysis and financial engineering.
//
// It is the implementation package for the numa engine. Most users should import
// github.com/rj-raghavjoshi/numa, which re-exports this package's API under
// shorter names; import vec directly only if you want to avoid that indirection.
//
// # What this package optimizes for
//
// Every reduction here walks a slice and produces a single number. That shape
// has a specific performance problem, and every tuned loop in this package is an
// answer to it.
//
// The problem is the dependency chain. The obvious implementation:
//
//	var total float64
//	for _, v := range xs {
//	    total += v
//	}
//
// has a loop-carried dependency: every iteration's add needs the previous
// iteration's add to finish. A modern CPU executes several instructions
// concurrently, but it cannot overlap two instructions when the second needs the
// first one's result. So the machine stalls, over and over, once per element.
//
// # The fix, in three parts
//
//  1. Multiple accumulators. Keeping several independent partial results creates
//     that many independent dependency chains, so the CPU always has a chain it
//     can make progress on.
//
//  2. Accumulating a pair at a time. Writing `s += a + b` rather than
//     `s += a; s += b` halves the number of links in each chain, because the
//     `a + b` result is a fresh temporary that nothing is waiting on.
//
//  3. A tail loop. The unrolled main loop consumes a fixed number of elements
//     per iteration, so the leftover elements are handled one at a time.
//
// # Not every operation benefits equally
//
// The size of the win depends on what the loop is waiting for:
//
//   - Reductions (Sum, Dot, SumSq) gain the most, roughly 4x, because they are
//     throughput-bound and the dependency chain is the bottleneck.
//   - Scans (Min, Max, MinMax) gain far less and run at a fraction of the
//     throughput, because a compare is inherently latency-bound and there is no
//     pairwise trick that shortens the chain.
//   - Elementwise maps (Add, Sub, Mul, Scale) gain only ~1.3x, because they have
//     no loop-carried dependency to fix in the first place and are already bound
//     by memory bandwidth.
//
// See ../docs/benchmarks.md for the measurements behind those claims.
//
// # Numerics: results are reassociated
//
// Floating-point addition is not associative: (a+b)+c can differ from a+(b+c).
// Because the tuned reductions reassociate the input, they can return a result
// that differs in the last few bits from a plain left-to-right loop. The
// difference is ordinarily smaller than the naive loop's own error relative to
// the exact answer, because summing several independent partial results keeps
// large and small magnitudes from being repeatedly mixed. But the difference is
// real, and it is a deliberate design decision rather than an accident.
//
// Two consequences worth knowing:
//
//   - Results are not guaranteed to be bit-identical across architectures. The
//     tuned loop shape differs per GOARCH by design.
//   - If you need a reproducible audit trail, this package cannot provide one as
//     currently designed.
//
// # Architecture selection
//
// The tuned loops are chosen at compile time by build tags. The number of
// accumulators differs between architectures because the number of available
// floating-point registers differs; using more accumulators than the machine has
// registers to hold causes spills, which defeats the entire purpose.
package vec

// Version is the current semantic release of the numa engine.
const Version = "0.1.0"
