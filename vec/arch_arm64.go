//go:build arm64

package vec

import "math"

// ---------------------------------------------------------------------------
// ARM64 tuning parameters.
//
// ARM64 has 32 floating-point registers available to a leaf function, but only
// a subset are practical for accumulators once addressing temporaries are
// accounted for. The sweet spot for these reductions on this architecture is
// two independent accumulators fed two elements each per iteration, i.e. a
// stride of four.
//
// Each loop below is therefore written as:
//
//	accumulator 0 <- a pair of elements
//	accumulator 1 <- a different pair of elements
//
// Two independent chains, two elements per chain, four elements per iteration.
// Accumulating a *pair* into each accumulator (rather than one element twice)
// matters: it keeps each accumulator's dependency chain half as long as the
// naive version, which is what lets the loops run without stalling.
// ---------------------------------------------------------------------------

// sumArch is the ARM64 tuned summation.
//
// Two independent accumulators, combined at the end as a small tree (s0 + s1)
// rather than a left-to-right fold.
func sumArch(xs []float64) float64 {
	n := len(xs)
	var s0, s1 float64
	i := 0
	limit := n - 3

	for i < limit {
		s0 += xs[i] + xs[i+1]
		s1 += xs[i+2] + xs[i+3]
		i += 4
	}

	for ; i < n; i++ {
		s0 += xs[i]
	}

	return s0 + s1
}

// dotArch is the ARM64 tuned inner product.
//
// The multiply-accumulate form is the important part: each iteration performs
// two independent mul+add pairs, so the multiply latency of one is hidden
// behind the other.
func dotArch(xs, ys []float64) float64 {
	n := len(xs)
	var s0, s1 float64
	i := 0
	limit := n - 3

	for i < limit {
		s0 += xs[i]*ys[i] + xs[i+1]*ys[i+1]
		s1 += xs[i+2]*ys[i+2] + xs[i+3]*ys[i+3]
		i += 4
	}

	for ; i < n; i++ {
		s0 += xs[i] * ys[i]
	}

	return s0 + s1
}

// sumSqArch is the ARM64 tuned sum of squares.
func sumSqArch(xs []float64) float64 {
	n := len(xs)
	var s0, s1 float64
	i := 0
	limit := n - 3

	for i < limit {
		s0 += xs[i]*xs[i] + xs[i+1]*xs[i+1]
		s1 += xs[i+2]*xs[i+2] + xs[i+3]*xs[i+3]
		i += 4
	}

	for ; i < n; i++ {
		s0 += xs[i] * xs[i]
	}

	return s0 + s1
}

// minArch is the ARM64 tuned minimum scan, including NaN detection.
//
// The NaN check is fused into the loop rather than performed as a second pass.
// Measured, the fusion costs essentially nothing: the scan alone runs at 9.51
// GB/s and the scan with the check at 9.43 GB/s, while a separate pass halves it
// to 4.68 GB/s. The reason is that the NaN test is a boolean flag accumulated with
// integer operations, so it runs on different execution ports than the
// floating-point compare and does not lengthen the latency-bound chain.
//
// (An arithmetic trick was tried first: accumulating `v - v` as a poison value,
// since that is 0 for finite v and NaN for NaN. It does not work, because
// `Inf - Inf` is also NaN, so any array containing an infinity is falsely
// poisoned. So is `Inf * 0`. There is no branch-free float expression that
// isolates NaN from infinity; the compare is required.)
//
// The result is NaN if any element is NaN. See nan.go for the policy.
func minArch(xs []float64) float64 {
	m0, m1, nan := scanMin2(xs)
	if nan {
		return math.NaN()
	}
	if m1 < m0 {
		return m1
	}
	return m0
}

// maxArch is the ARM64 tuned maximum scan, including NaN detection.
// See minArch for why the check is fused rather than a separate pass.
func maxArch(xs []float64) float64 {
	m0, m1, nan := scanMax2(xs)
	if nan {
		return math.NaN()
	}
	if m1 > m0 {
		return m1
	}
	return m0
}

// minMaxArch is the ARM64 tuned combined min/max scan, including NaN detection.
//
// Four accumulators: separate min and max for each of the two halves. Keeping
// the min and max chains distinct lets the compare circuitry stay busy, since
// the two directions are independent.
func minMaxArch(xs []float64) (lo, hi float64) {
	lo, hi, nan := scanMinMax2(xs)
	if nan {
		return math.NaN(), math.NaN()
	}
	return lo, hi
}

// scanMin2 scans xs with two independent minimum chains, returning both partial
// results and whether any element was NaN.
//
// The per-element NaN test is `v != v`, the standard idiom: it is the only value
// for which the comparison is true. The flag is accumulated with `||` into a
// separate variable per accumulator rather than into one shared flag, so the two
// chains stay independent and the compiler does not introduce a loop-carried
// dependency on the flag itself.
func scanMin2(xs []float64) (m0, m1 float64, nan bool) {
	m0, m1 = xs[0], xs[0]
	var nan0, nan1 bool
	i := 0
	limit := len(xs) - 1

	for i < limit {
		v0, v1 := xs[i], xs[i+1]

		nan0 = nan0 || v0 != v0
		nan1 = nan1 || v1 != v1

		if v0 < m0 {
			m0 = v0
		}
		if v1 < m1 {
			m1 = v1
		}
		i += 2
	}

	for ; i < len(xs); i++ {
		v := xs[i]
		nan0 = nan0 || v != v
		if v < m0 {
			m0 = v
		}
	}

	return m0, m1, nan0 || nan1
}

// scanMax2 is scanMin2 with the comparison reversed.
func scanMax2(xs []float64) (m0, m1 float64, nan bool) {
	m0, m1 = xs[0], xs[0]
	var nan0, nan1 bool
	i := 0
	limit := len(xs) - 1

	for i < limit {
		v0, v1 := xs[i], xs[i+1]

		nan0 = nan0 || v0 != v0
		nan1 = nan1 || v1 != v1

		if v0 > m0 {
			m0 = v0
		}
		if v1 > m1 {
			m1 = v1
		}
		i += 2
	}

	for ; i < len(xs); i++ {
		v := xs[i]
		nan0 = nan0 || v != v
		if v > m0 {
			m0 = v
		}
	}

	return m0, m1, nan0 || nan1
}

// scanMinMax2 is the ARM64 combined min/max scan with fused NaN detection.
func scanMinMax2(xs []float64) (lo, hi float64, nan bool) {
	lo0, lo1 := xs[0], xs[0]
	hi0, hi1 := xs[0], xs[0]
	var nan0, nan1 bool
	i := 0
	limit := len(xs) - 1

	for i < limit {
		a, b := xs[i], xs[i+1]

		nan0 = nan0 || a != a
		nan1 = nan1 || b != b

		if a < lo0 {
			lo0 = a
		}
		if a > hi0 {
			hi0 = a
		}
		if b < lo1 {
			lo1 = b
		}
		if b > hi1 {
			hi1 = b
		}
		i += 2
	}

	for ; i < len(xs); i++ {
		v := xs[i]
		nan0 = nan0 || v != v
		if v < lo0 {
			lo0 = v
		}
		if v > hi0 {
			hi0 = v
		}
	}

	if lo1 < lo0 {
		lo0 = lo1
	}
	if hi1 > hi0 {
		hi0 = hi1
	}
	return lo0, hi0, nan0 || nan1
}

// addArch is the ARM64 tuned elementwise addition.
//
// Elementwise maps have no reduction dependency chain at all -- dst[i] depends
// only on xs[i] and ys[i] -- so they are bound by memory throughput instead of
// by compute. The tuning here is simply to issue four independent loads and one
// store per iteration so the memory subsystem stays saturated, plus a tail loop
// for the remainder.
func addArch(dst, xs, ys []float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 3

	for i < limit {
		dst[i] = xs[i] + ys[i]
		dst[i+1] = xs[i+1] + ys[i+1]
		dst[i+2] = xs[i+2] + ys[i+2]
		dst[i+3] = xs[i+3] + ys[i+3]
		i += 4
	}

	for ; i < n; i++ {
		dst[i] = xs[i] + ys[i]
	}
	return dst
}

// subArch is the ARM64 tuned elementwise subtraction.
func subArch(dst, xs, ys []float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 3

	for i < limit {
		dst[i] = xs[i] - ys[i]
		dst[i+1] = xs[i+1] - ys[i+1]
		dst[i+2] = xs[i+2] - ys[i+2]
		dst[i+3] = xs[i+3] - ys[i+3]
		i += 4
	}

	for ; i < n; i++ {
		dst[i] = xs[i] - ys[i]
	}
	return dst
}

// mulArch is the ARM64 tuned elementwise multiplication.
func mulArch(dst, xs, ys []float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 3

	for i < limit {
		dst[i] = xs[i] * ys[i]
		dst[i+1] = xs[i+1] * ys[i+1]
		dst[i+2] = xs[i+2] * ys[i+2]
		dst[i+3] = xs[i+3] * ys[i+3]
		i += 4
	}

	for ; i < n; i++ {
		dst[i] = xs[i] * ys[i]
	}
	return dst
}

// scaleArch is the ARM64 tuned scalar multiply.
func scaleArch(dst, xs []float64, k float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 3

	for i < limit {
		dst[i] = xs[i] * k
		dst[i+1] = xs[i+1] * k
		dst[i+2] = xs[i+2] * k
		dst[i+3] = xs[i+3] * k
		i += 4
	}

	for ; i < n; i++ {
		dst[i] = xs[i] * k
	}
	return dst
}

// addScalarArch is the ARM64 tuned scalar add.
func addScalarArch(dst, xs []float64, k float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 3

	for i < limit {
		dst[i] = xs[i] + k
		dst[i+1] = xs[i+1] + k
		dst[i+2] = xs[i+2] + k
		dst[i+3] = xs[i+3] + k
		i += 4
	}

	for ; i < n; i++ {
		dst[i] = xs[i] + k
	}
	return dst
}

// absArch is the ARM64 tuned absolute value.
func absArch(dst, xs []float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 3

	for i < limit {
		dst[i] = abs(xs[i])
		dst[i+1] = abs(xs[i+1])
		dst[i+2] = abs(xs[i+2])
		dst[i+3] = abs(xs[i+3])
		i += 4
	}

	for ; i < n; i++ {
		dst[i] = abs(xs[i])
	}
	return dst
}

// ---------------------------------------------------------------------------
// Arg reductions, ARM64 tuning.
//
// Two independent partial scans, two elements per iteration -- the same structure
// as scanMin2, for the same reason: the compare chain is the bottleneck and the
// only parallelism available is splitting the input.
//
// The NaN flag is per-chain so it does not become a shared loop-carried
// dependency, exactly as in the Min/Max scans.
//
// # Tie handling across chains
//
// Values alone are not enough to reconstruct the answer. Each chain finds its own
// extreme, and two chains can hold the *same* extreme value at different indices.
// The combine step therefore breaks an equality by index, returning the smaller.
// A combine that compared only values would return the wrong index for a tie that
// straddles the chain boundary; TestArgReductionsTiesResolveFirst covers it.
//
// Chain 1's index starts at 0 rather than 1, which is safe for the same reason: if
// chain 1's value still equals the seed xs[0], then index 0 is genuinely the first
// occurrence of that value and the tie-break keeps it.
// ---------------------------------------------------------------------------

func argMinArch(xs []float64) int {
	m0, m1 := xs[0], xs[0]
	i0, i1 := 0, 0
	var nan0, nan1 bool
	i := 0
	limit := len(xs) - 1

	for i < limit {
		v0, v1 := xs[i], xs[i+1]
		nan0 = nan0 || v0 != v0
		nan1 = nan1 || v1 != v1
		if v0 < m0 {
			m0, i0 = v0, i
		}
		if v1 < m1 {
			m1, i1 = v1, i+1
		}
		i += 2
	}
	for ; i < len(xs); i++ {
		v := xs[i]
		nan0 = nan0 || v != v
		if v < m0 {
			m0, i0 = v, i
		}
	}

	if nan0 || nan1 {
		return -1
	}
	switch {
	case m1 < m0:
		return i1
	case m0 < m1:
		return i0
	case i1 < i0:
		return i1
	default:
		return i0
	}
}

func argMaxArch(xs []float64) int {
	m0, m1 := xs[0], xs[0]
	i0, i1 := 0, 0
	var nan0, nan1 bool
	i := 0
	limit := len(xs) - 1

	for i < limit {
		v0, v1 := xs[i], xs[i+1]
		nan0 = nan0 || v0 != v0
		nan1 = nan1 || v1 != v1
		if v0 > m0 {
			m0, i0 = v0, i
		}
		if v1 > m1 {
			m1, i1 = v1, i+1
		}
		i += 2
	}
	for ; i < len(xs); i++ {
		v := xs[i]
		nan0 = nan0 || v != v
		if v > m0 {
			m0, i0 = v, i
		}
	}

	if nan0 || nan1 {
		return -1
	}
	switch {
	case m1 > m0:
		return i1
	case m0 > m1:
		return i0
	case i1 < i0:
		return i1
	default:
		return i0
	}
}

func argMinMaxArch(xs []float64) (int, int) {
	lo0, lo1 := xs[0], xs[0]
	hi0, hi1 := xs[0], xs[0]
	li0, li1 := 0, 0
	hi_i0, hi_i1 := 0, 0
	var nan0, nan1 bool
	i := 0
	limit := len(xs) - 1

	for i < limit {
		a, b := xs[i], xs[i+1]
		nan0 = nan0 || a != a
		nan1 = nan1 || b != b
		if a < lo0 {
			lo0, li0 = a, i
		}
		if a > hi0 {
			hi0, hi_i0 = a, i
		}
		if b < lo1 {
			lo1, li1 = b, i+1
		}
		if b > hi1 {
			hi1, hi_i1 = b, i+1
		}
		i += 2
	}
	for ; i < len(xs); i++ {
		v := xs[i]
		nan0 = nan0 || v != v
		if v < lo0 {
			lo0, li0 = v, i
		}
		if v > hi0 {
			hi0, hi_i0 = v, i
		}
	}

	if nan0 || nan1 {
		return -1, -1
	}

	loi := li0
	if lo1 < lo0 || (lo1 == lo0 && li1 < loi) {
		loi = li1
	}
	hii := hi_i0
	if hi1 > hi0 || (hi1 == hi0 && hi_i1 < hii) {
		hii = hi_i1
	}
	return loi, hii
}

// ---------------------------------------------------------------------------
// Statistics helpers, ARM64 tuning.
//
// These are the inner sums of the two-pass statistics in stats.go. They are
// reductions over deviations rather than over the raw values, which is what makes
// the statistics numerically stable: computing a sum of squares of the raw values
// and subtracting n*mean^2 loses most of the significant digits when the mean
// dwarfs the spread, which is the normal case for a price series.
//
// # Why these take a center rather than materialising deviations
//
// The obvious alternative is to build a scratch slice of (x - center) and reuse
// [sumSqArch] and [dotArch]. That allocates per call and writes a full pass of
// memory that is read back immediately, so it costs a third memory traversal for no
// accuracy benefit. Subtracting the center inside the accumulator loop keeps the two
// traversals the algorithms already need and nothing more.
//
// Both use the two-chain ARM64 shape, for the usual reason: the multiply and the
// add each carry latency, and two independent chains hide one behind the other.
// ---------------------------------------------------------------------------

// dotDevArch returns the sum of (xs[i]-cx)*(ys[i]-cy) over min(len(xs), len(ys))
// elements. When called with ys == xs and cy == cx it computes the sum of squared
// deviations, which is the second pass of a variance.
func dotDevArch(xs, ys []float64, cx, cy float64) float64 {
	n := min(len(xs), len(ys))
	var s0, s1 float64
	i := 0
	limit := n - 3

	for i < limit {
		s0 += (xs[i]-cx)*(ys[i]-cy) + (xs[i+1]-cx)*(ys[i+1]-cy)
		s1 += (xs[i+2]-cx)*(ys[i+2]-cy) + (xs[i+3]-cx)*(ys[i+3]-cy)
		i += 4
	}

	for ; i < n; i++ {
		s0 += (xs[i] - cx) * (ys[i] - cy)
	}

	return s0 + s1
}

// sumAbsDevArch returns the sum of |xs[i]-center| over xs.
func sumAbsDevArch(xs []float64, center float64) float64 {
	n := len(xs)
	var s0, s1 float64
	i := 0
	limit := n - 3

	for i < limit {
		s0 += abs(xs[i]-center) + abs(xs[i+1]-center)
		s1 += abs(xs[i+2]-center) + abs(xs[i+3]-center)
		i += 4
	}

	for ; i < n; i++ {
		s0 += abs(xs[i] - center)
	}

	return s0 + s1
}
