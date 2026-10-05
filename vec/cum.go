package vec

import "math"

// ---------------------------------------------------------------------------
// Cumulative (prefix/scan) operations.
//
// Each of these produces one output per input, where output i depends on every
// input up to and including i. That is a scan, not a map: there is a genuine
// loop-carried dependency, so the naive loop stalls on it once per element.
//
// # Why this file is not arch-tagged
//
// The arch-tagged reductions vary the number of independent accumulators, because
// that count is a property of the register file. A prefix scan cannot use
// independent accumulators at all: output i needs the running value, so there is
// exactly one chain by definition. What shortens it is not more registers but
// *blocking* -- computing a short local prefix inside a block and then adding the
// carried total to each of its elements. The carry chain then advances once per
// block instead of once per element, and the block size is a fixed algorithmic
// choice rather than a register-count choice, so it does not vary by architecture.
//
// Measured (docs/benchmarks.md): CumSum reaches 41.0 GB/s against 12.7 GB/s for
// the plain loop at n=4M, a 3.22x gain. That is the whole reason for the structure
// below.
//
// # Reassociation
//
// The blocked form reassociates, like every other tuned loop here: it computes
// (carry + (x0 + x1 + x2 + x3)) rather than ((((carry + x0) + x1) + x2) + x3).
// Results may differ in the last bits from a naive prefix sum, and the package's
// usual caveats apply.
//
// # Aliasing
//
// Every *To form here permits dst to alias xs. Each block reads its inputs into
// locals before writing any output, and the tail loop reads xs[i] before writing
// dst[i].
// ---------------------------------------------------------------------------

// CumSum returns the inclusive prefix sum of xs, so out[i] is the sum of
// xs[0..i].
//
// It returns nil for an empty input.
//
// CumSum allocates; for hot loops that can supply a buffer, use [CumSumTo].
func CumSum(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return CumSumTo(make([]float64, len(xs)), xs)
}

// CumSumTo stores the inclusive prefix sum of xs into dst and returns dst.
//
// dst and xs must be the same length. CumSumTo panics if they are not. dst may
// alias xs.
//
// # How it is blocked
//
// The input is consumed four elements at a time. Inside a block the local prefix
// a0, a1, a2, a3 is computed with three adds that do not depend on the carried
// total, so they can overlap with the previous block's completion. The carry is
// then added to each of the four locals and advanced once. The critical path is
// therefore one add per four elements, not one add per element.
func CumSumTo(dst, xs []float64) []float64 {
	checkSameLen1("CumSumTo", dst, xs)
	return cumSumLoop(dst, xs)
}

func cumSumLoop(dst, xs []float64) []float64 {
	n := len(dst)
	var carry float64
	i := 0
	limit := n - 3
	for i < limit {
		a0 := xs[i]
		a1 := a0 + xs[i+1]
		a2 := a1 + xs[i+2]
		a3 := a2 + xs[i+3]

		dst[i] = carry + a0
		dst[i+1] = carry + a1
		dst[i+2] = carry + a2
		dst[i+3] = carry + a3
		carry = dst[i+3]
		i += 4
	}
	for ; i < n; i++ {
		carry += xs[i]
		dst[i] = carry
	}
	return dst
}

// CumProd returns the inclusive prefix product of xs, so out[i] is the product
// of xs[0..i].
//
// It returns nil for an empty input.
//
// CumProd allocates; for hot loops that can supply a buffer, use [CumProdTo].
func CumProd(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return CumProdTo(make([]float64, len(xs)), xs)
}

// CumProdTo stores the inclusive prefix product of xs into dst and returns dst.
//
// dst and xs must be the same length. CumProdTo panics if they are not. dst may
// alias xs.
//
// The blocking is the same as [CumSumTo]'s. Multiplication is not faster than
// addition, so the gain comes from the same place: the carried value advances
// once per block rather than once per element.
//
// A prefix product overflows to ±Inf or underflows to zero quickly for most real
// inputs; it is offered because a caller building a geometric series or a
// compounding factor needs it, not because it is commonly the right tool.
func CumProdTo(dst, xs []float64) []float64 {
	checkSameLen1("CumProdTo", dst, xs)
	n := len(dst)
	carry := 1.0
	i := 0
	limit := n - 3
	for i < limit {
		a0 := xs[i]
		a1 := a0 * xs[i+1]
		a2 := a1 * xs[i+2]
		a3 := a2 * xs[i+3]

		dst[i] = carry * a0
		dst[i+1] = carry * a1
		dst[i+2] = carry * a2
		dst[i+3] = carry * a3
		carry = dst[i+3]
		i += 4
	}
	for ; i < n; i++ {
		carry *= xs[i]
		dst[i] = carry
	}
	return dst
}

// CumMax returns the running maximum of xs, so out[i] is the largest of xs[0..i].
//
// NaN wins: once any element is NaN, every output from that index onward is NaN.
// The mechanism is that the running maximum is *replaced* by NaN rather than
// compared against it, so the poisoning is self-sustaining and no separate flag or
// second pass is needed. See [NaN policy] for why NaN wins rather than is skipped.
//
// It returns nil for an empty input.
//
// CumMax allocates; for hot loops that can supply a buffer, use [CumMaxTo].
//
// [NaN policy]: #nan-policy
func CumMax(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return CumMaxTo(make([]float64, len(xs)), xs)
}

// CumMaxTo stores the running maximum of xs into dst and returns dst.
//
// dst and xs must be the same length. CumMaxTo panics if they are not. dst may
// alias xs.
//
// Unlike [CumSumTo] this is not blocked. Max has no associative "block prefix plus
// carry" form that shortens the chain more than the compare already does: the
// output at each position needs the true running maximum, and the local prefix
// maximum within a block still requires a compare per element to produce. The
// carry would advance once per block, but the per-element compares remain, so the
// blocking removes nothing. This is the same latency-versus-throughput wall the
// Min/Max scans hit.
//
// That reasoning is confirmed by measurement rather than left as an argument: at
// n=4M this reaches 9.65 GB/s against 9.66 GB/s for the plain loop, exactly break
// even. See docs/benchmarks.md.
func CumMaxTo(dst, xs []float64) []float64 {
	checkSameLen1("CumMaxTo", dst, xs)
	m := math.Inf(-1)
	for i, v := range xs {
		if v != v {
			m = math.NaN()
		} else if v > m {
			m = v
		}
		dst[i] = m
	}
	return dst
}

// CumMin returns the running minimum of xs, so out[i] is the smallest of xs[0..i].
//
// NaN wins, by the same mechanism as [CumMax]: the running value is replaced by
// NaN, and every later comparison against NaN is false, so the NaN persists.
//
// It returns nil for an empty input.
//
// CumMin allocates; for hot loops that can supply a buffer, use [CumMinTo].
func CumMin(xs []float64) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return CumMinTo(make([]float64, len(xs)), xs)
}

// CumMinTo stores the running minimum of xs into dst and returns dst.
//
// dst and xs must be the same length. CumMinTo panics if they are not. dst may
// alias xs. See [CumMaxTo] for why this is not blocked.
func CumMinTo(dst, xs []float64) []float64 {
	checkSameLen1("CumMinTo", dst, xs)
	m := math.Inf(1)
	for i, v := range xs {
		if v != v {
			m = math.NaN()
		} else if v < m {
			m = v
		}
		dst[i] = m
	}
	return dst
}

// CumCountTrue returns the running count of non-zero entries in mask, so out[i]
// is the number of non-zero entries in mask[0..i].
//
// It returns nil for an empty input. The count is an int, not a float, because a
// running count is a position and positions should not be rounded.
func CumCountTrue(mask []uint8) []int {
	if len(mask) == 0 {
		return nil
	}
	return CumCountTrueTo(make([]int, len(mask)), mask)
}

// CumCountTrueTo stores the running count of non-zero entries into dst and
// returns dst.
//
// dst and mask must be the same length. CumCountTrueTo panics if they are not.
func CumCountTrueTo(dst []int, mask []uint8) []int {
	if len(dst) != len(mask) {
		panic("vec: CumCountTrueTo length mismatch")
	}
	c := 0
	for i, m := range mask {
		c += int(boolToU8(m != 0))
		dst[i] = c
	}
	return dst
}
