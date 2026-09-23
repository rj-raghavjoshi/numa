package vec

// # NaN policy
//
// [Min], [Max] and [MinMax] return NaN if any element of the input is NaN. This is
// the "NaN wins" policy, chosen deliberately over "ignore NaN".
//
// The reasoning: an array containing NaN is almost always the result of an
// upstream error (a 0/0, an inf-inf, a failed parse). Silently returning the
// extreme of the *remaining* values would hide that error and produce a
// plausible-looking number, which is worse than propagating it in a domain where
// the numbers feed decisions. Failing loudly is the safer default.
//
// The alternative is not merely a different opinion, it is worse in a specific
// way: a naive scan's behaviour is *positional*. Since `v < m` is false for NaN, a
// NaN at index 0 gets overwritten by later comparisons while a NaN at the last
// index gets returned. Neither policy justifies a result that depends on where the
// bad value sits, which is why the check exists rather than being left implicit.
//
// # The cost
//
// NaN detection is a separate pass over the input, and it costs about 2x. Because
// the scans are latency-bound, this consumes the entire benefit of the tuned scan
// loop (measured 0.98x vs the naive loop at n=4M, where the scan loop alone is
// 1.98x). This is a known and currently unresolved trade — correctness over
// throughput. See ../docs/next-steps.md for the open decision and the options.
//
// If you want NaN-skipping semantics, filter the input first with a predicate.
// That keeps the policy visible at the call site rather than buried in the
// reduction, and it costs a pass either way.

// hasNaN reports whether xs contains a NaN.
//
// The comparison `v != v` is the standard idiom for NaN detection: it is the only
// value for which it is true. A dedicated math.IsNaN call compiles to the same
// thing but is a function call in the source, and this loop is hot enough that the
// distinction is worth keeping visible.
//
// The loop is written to be vectorizable: it is a pure map from input to a boolean
// accumulator, with no early exit, so the compiler can process it in wide chunks.
// An early-return version would serialise on a branch, and this pass is already the
// most expensive thing in [Min].
func hasNaN(xs []float64) bool {
	found := false
	for _, v := range xs {
		found = found || v != v
	}
	return found
}
