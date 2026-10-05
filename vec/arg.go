package vec

// ---------------------------------------------------------------------------
// Arg reductions: the *index* of an extreme rather than the extreme itself.
//
// # Why these are arch-tagged
//
// An arg reduction is a scan: one output, a loop-carried compare. It is
// latency-bound, and the only available parallelism is independent partial scans
// whose chain length is divided by the accumulator count -- exactly like Min and
// Max. So ArgMin and ArgMax have the same per-architecture tuning (two chains on
// arm64, four on x86-64) and a naive generic reference.
//
// The measurement is why this is not merely a stylistic choice. A first version
// fused the NaN check into a single-chain loop and came out at **0.51x** -- half the
// throughput of a plain loop -- because the NaN compare and the ordering compare
// contend for the same compare port on a chain that has nothing to hide them behind.
// Splitting into independent chains recovers it. See docs/benchmarks.md.
//
// # NaN policy
//
// The arg reductions return -1 when the input contains NaN. That is the same
// "NaN wins" policy as [Min] and [Max], expressed in a domain that has no NaN
// index: an index cannot be NaN, so the only honest answers are "here it is" or
// "not determinable". Returning a plausible index computed from the non-NaN
// elements would hide the error, which is exactly what the policy exists to prevent.
//
// Note that -1 is therefore returned for two distinct reasons, an empty input and
// a NaN-containing one. Callers who must distinguish them can check the length and
// use [HasNaN]. Keeping one sentinel rather than two was chosen because a caller
// who ignores the distinction is more likely to be the bug.
//
// # Tie resolution
//
// Ties resolve to the first occurrence. That is a real constraint on the tuned
// implementations rather than a free property of them: with independent partial
// scans, the same extreme value can be found at a different index in each chain, so
// the combine step must break an equality by index. A combine that only compared
// values would return the wrong index for a tie across chains, which
// TestArgReductionsTiesResolveFirst catches.
// ---------------------------------------------------------------------------

// ArgMin returns the index of the smallest element of xs, or -1 if xs is empty or
// contains NaN. Ties resolve to the first occurrence.
func ArgMin(xs []float64) int {
	if len(xs) == 0 {
		return -1
	}
	return argMinArch(xs)
}

// ArgMax returns the index of the largest element of xs, or -1 if xs is empty or
// contains NaN. Ties resolve to the first occurrence.
func ArgMax(xs []float64) int {
	if len(xs) == 0 {
		return -1
	}
	return argMaxArch(xs)
}

// ArgMinMax returns the indices of the smallest and largest elements of xs in a
// single pass.
//
// It returns (-1, -1) for an empty input or if any element is NaN. Callers that
// need both indices should prefer this over calling [ArgMin] and [ArgMax], which
// read the input twice.
func ArgMinMax(xs []float64) (loIdx, hiIdx int) {
	if len(xs) == 0 {
		return -1, -1
	}
	return argMinMaxArch(xs)
}

// HasNaN reports whether any element of xs is NaN.
//
// It is the underlying test for the arg reductions' -1 sentinel, and it is exported
// because callers of ArgMin and ArgMax need exactly this to tell the two meanings
// of -1 apart.
//
// The loop returns early rather than running to completion. That is the right trade
// here: NaN is the rare case, so the common path is a full scan with a
// perfectly-predicted not-taken branch, while the rare path saves the rest of it.
func HasNaN(xs []float64) bool {
	for _, v := range xs {
		if v != v {
			return true
		}
	}
	return false
}
