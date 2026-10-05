// Package series provides stateful streaming computations over a sequence of
// values: rolling windows, exponential smoothers, and the incremental primitives
// the indicator layer is built from.
//
// # Relationship to vec
//
// The two packages answer different questions about the same data.
//
// [github.com/rj-raghavjoshi/numa/vec] is for *batch* work: it has the whole slice
// and computes over it, tuned per architecture. Package series is for *streaming*
// work: it receives one value at a time and maintains enough state to answer in
// O(1) per element, regardless of the window length.
//
// For a full-history computation the batch form is usually faster, because it can
// keep several independent accumulators and the compiler can schedule whole loops.
// The streaming form wins when the answer is needed at each step and the alternative
// is recomputing over the window, which is O(window) per element. BenchmarkSMA
// against BenchmarkNaiveSMA in this package's benchmark file measures exactly that
// difference.
//
// Neither is a wrapper around the other; they are the same mathematics arranged for
// two different access patterns.
//
// # The Roller contract
//
// A [Roller] receives values with Push and returns the current value of its
// computation:
//
//	Warmup() int             elements needed before the output is meaningful
//	Push(v float64) float64  consume one value, return the current output
//	Reset()                  return to the initial state, without reallocating
//
// Push returns NaN until Warmup() elements have been pushed, and NaN propagates:
// a NaN input makes the affected outputs NaN, matching the "NaN wins" policy of the
// vec scans. This is deliberate, and it means a caller can treat NaN uniformly as
// "not available yet" and "upstream error".
//
// Reset exists so one roller can process many series without allocating. Combined
// with [ApplyTo] this makes a roller reusable in a parameter sweep, which is the
// pattern the package was built for.
//
// # Interface dispatch
//
// [ApplyTo] and [Apply] take a [Roller], so each element costs an interface method
// call. That is the right price for the convenience of writing one driver over every
// roller. A caller in a genuinely hot loop should call Push on the concrete type
// directly, where the compiler can inline it:
//
//	s := series.NewSMA(20)
//	for _, v := range xs {
//	    out = append(out, s.Push(v))
//	}
//
// Both forms are tested to produce identical results.
package series
