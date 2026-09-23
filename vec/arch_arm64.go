//go:build arm64

package vec

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

// minArch is the ARM64 tuned minimum scan.
//
// Min/max reductions are latency-bound rather than throughput-bound: a compare
// cannot start until the previous compare's result is known, so the chain is
// inherently the length of the input. Two accumulators let the machine work on
// two halves of the input simultaneously, which is the only available
// parallelism. There is no pairwise trick here, because min has no equivalent
// of `a + b`.
func minArch(xs []float64) float64 {
	m0, m1 := scanMin2(xs)
	if m1 < m0 {
		return m1
	}
	return m0
}

// maxArch is the ARM64 tuned maximum scan.
func maxArch(xs []float64) float64 {
	m0, m1 := scanMax2(xs)
	if m1 > m0 {
		return m1
	}
	return m0
}

// minMaxArch is the ARM64 tuned combined min/max scan.
//
// Four accumulators: separate min and max for each of the two halves. Keeping
// the min and max chains distinct lets the compare circuitry stay busy, since
// the two directions are independent.
func minMaxArch(xs []float64) (lo, hi float64) {
	return scanMinMax2(xs)
}

// scanMin2 scans xs with two independent minimum chains and returns both
// partial results. The NaN policy is documented in nan.go.
func scanMin2(xs []float64) (m0, m1 float64) {
	m0, m1 = xs[0], xs[0]
	i := 0
	limit := len(xs) - 1

	for i < limit {
		v0, v1 := xs[i], xs[i+1]
		if v0 < m0 {
			m0 = v0
		}
		if v1 < m1 {
			m1 = v1
		}
		i += 2
	}

	for ; i < len(xs); i++ {
		if v := xs[i]; v < m0 {
			m0 = v
		}
	}

	return m0, m1
}

// scanMax2 is scanMin2 with the comparison reversed.
func scanMax2(xs []float64) (m0, m1 float64) {
	m0, m1 = xs[0], xs[0]
	i := 0
	limit := len(xs) - 1

	for i < limit {
		v0, v1 := xs[i], xs[i+1]
		if v0 > m0 {
			m0 = v0
		}
		if v1 > m1 {
			m1 = v1
		}
		i += 2
	}

	for ; i < len(xs); i++ {
		if v := xs[i]; v > m0 {
			m0 = v
		}
	}

	return m0, m1
}

// scanMinMax2 is the ARM64 combined scan.
func scanMinMax2(xs []float64) (lo, hi float64) {
	lo0, lo1 := xs[0], xs[0]
	hi0, hi1 := xs[0], xs[0]
	i := 0
	limit := len(xs) - 1

	for i < limit {
		a, b := xs[i], xs[i+1]

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
	return lo0, hi0
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
