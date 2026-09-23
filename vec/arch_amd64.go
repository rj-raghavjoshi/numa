//go:build amd64

package vec

import "math"

// ---------------------------------------------------------------------------
// x86-64 tuning parameters.
//
// x86-64 has 16 architectural XMM registers (more with AVX-512). That is enough
// headroom to run four independent accumulator chains, so every reduction here
// uses four accumulators fed two elements each per iteration, i.e. a stride of
// eight.
//
// The extra accumulators are not free -- each needs a register, and running out
// causes spilling to memory, which is far slower than the stall the extra
// accumulator was meant to hide. Four is the measured sweet spot for these
// reductions on this architecture. See ../docs/benchmarks.md.
// ---------------------------------------------------------------------------

// sumArch is the x86-64 tuned summation.
//
// Four independent accumulator chains, two elements per chain per iteration.
// The final combination is written with explicit parentheses, (s0+s1)+(s2+s3),
// so the shape is a balanced tree rather than a left-to-right fold. The tree
// matters: it is one level deep instead of three, and it pairs
// similar-magnitude partial sums together, which reduces rounding error.
func sumArch(xs []float64) float64 {
	n := len(xs)
	var s0, s1, s2, s3 float64
	i := 0
	limit := n - 7

	for i < limit {
		s0 += xs[i] + xs[i+1]
		s1 += xs[i+2] + xs[i+3]
		s2 += xs[i+4] + xs[i+5]
		s3 += xs[i+6] + xs[i+7]
		i += 8
	}

	for ; i < n; i++ {
		s0 += xs[i]
	}

	return (s0 + s1) + (s2 + s3)
}

// dotArch is the x86-64 tuned inner product.
//
// Four independent multiply-accumulate chains. Dot is the reduction that most
// benefits from this: a floating-point multiply has higher latency than an add,
// so without independent chains the loop stalls on every single element.
func dotArch(xs, ys []float64) float64 {
	n := len(xs)
	var s0, s1, s2, s3 float64
	i := 0
	limit := n - 7

	for i < limit {
		s0 += xs[i]*ys[i] + xs[i+1]*ys[i+1]
		s1 += xs[i+2]*ys[i+2] + xs[i+3]*ys[i+3]
		s2 += xs[i+4]*ys[i+4] + xs[i+5]*ys[i+5]
		s3 += xs[i+6]*ys[i+6] + xs[i+7]*ys[i+7]
		i += 8
	}

	for ; i < n; i++ {
		s0 += xs[i] * ys[i]
	}

	return (s0 + s1) + (s2 + s3)
}

// sumSqArch is the x86-64 tuned sum of squares.
func sumSqArch(xs []float64) float64 {
	n := len(xs)
	var s0, s1, s2, s3 float64
	i := 0
	limit := n - 7

	for i < limit {
		s0 += xs[i]*xs[i] + xs[i+1]*xs[i+1]
		s1 += xs[i+2]*xs[i+2] + xs[i+3]*xs[i+3]
		s2 += xs[i+4]*xs[i+4] + xs[i+5]*xs[i+5]
		s3 += xs[i+6]*xs[i+6] + xs[i+7]*xs[i+7]
		i += 8
	}

	for ; i < n; i++ {
		s0 += xs[i] * xs[i]
	}

	return (s0 + s1) + (s2 + s3)
}

// minArch is the x86-64 tuned minimum scan, including NaN detection.
//
// Four accumulators. Unlike summation there is no pairwise trick available,
// because min has no associative "combine two at once" form that shortens the
// chain -- a compare always needs the running value. Splitting the input into
// four independent partial scans is therefore the entire optimization.
//
// The NaN check is fused into the loop rather than a separate pass. Measured on
// arm64, fusion is nearly free (9.43 GB/s vs 9.51 GB/s without any check) while a
// second pass halves throughput (4.68 GB/s). The NaN test is an integer-flag
// accumulation, so it uses different ports than the float compare and does not
// lengthen the latency-bound chain. See arch_arm64.go's minArch for why the
// arithmetic alternative does not work.
func minArch(xs []float64) float64 {
	m0, m1, m2, m3, nan := scanMin4(xs)
	if nan {
		return math.NaN()
	}
	lo01, lo23 := m0, m2
	if m1 < lo01 {
		lo01 = m1
	}
	if m3 < lo23 {
		lo23 = m3
	}
	if lo23 < lo01 {
		return lo23
	}
	return lo01
}

// maxArch is the x86-64 tuned maximum scan, including NaN detection.
func maxArch(xs []float64) float64 {
	m0, m1, m2, m3, nan := scanMax4(xs)
	if nan {
		return math.NaN()
	}
	hi01, hi23 := m0, m2
	if m1 > hi01 {
		hi01 = m1
	}
	if m3 > hi23 {
		hi23 = m3
	}
	if hi23 > hi01 {
		return hi23
	}
	return hi01
}

// minMaxArch is the x86-64 tuned combined min/max scan, including NaN detection.
//
// Eight accumulators: four scanning for the minimum and four for the maximum.
// That is a lot of live registers, which is exactly why the min/max pair is
// worth measuring rather than assuming.
func minMaxArch(xs []float64) (lo, hi float64) {
	lo, hi, nan := scanMinMax4(xs)
	if nan {
		return math.NaN(), math.NaN()
	}
	return lo, hi
}

// scanMin4 scans xs with four independent minimum chains, plus a fused NaN check.
//
// Four separate NaN flags rather than one shared flag, so the chains stay
// independent and the flag does not become a loop-carried dependency.
func scanMin4(xs []float64) (m0, m1, m2, m3 float64, nan bool) {
	m0, m1, m2, m3 = xs[0], xs[0], xs[0], xs[0]
	var nan0, nan1, nan2, nan3 bool
	i := 0
	limit := len(xs) - 3

	for i < limit {
		a, b, c, d := xs[i], xs[i+1], xs[i+2], xs[i+3]

		nan0 = nan0 || a != a
		nan1 = nan1 || b != b
		nan2 = nan2 || c != c
		nan3 = nan3 || d != d

		if a < m0 {
			m0 = a
		}
		if b < m1 {
			m1 = b
		}
		if c < m2 {
			m2 = c
		}
		if d < m3 {
			m3 = d
		}
		i += 4
	}

	for ; i < len(xs); i++ {
		v := xs[i]
		nan0 = nan0 || v != v
		if v < m0 {
			m0 = v
		}
	}

	return m0, m1, m2, m3, nan0 || nan1 || nan2 || nan3
}

// scanMax4 is scanMin4 with the comparison reversed.
func scanMax4(xs []float64) (m0, m1, m2, m3 float64, nan bool) {
	m0, m1, m2, m3 = xs[0], xs[0], xs[0], xs[0]
	var nan0, nan1, nan2, nan3 bool
	i := 0
	limit := len(xs) - 3

	for i < limit {
		a, b, c, d := xs[i], xs[i+1], xs[i+2], xs[i+3]

		nan0 = nan0 || a != a
		nan1 = nan1 || b != b
		nan2 = nan2 || c != c
		nan3 = nan3 || d != d

		if a > m0 {
			m0 = a
		}
		if b > m1 {
			m1 = b
		}
		if c > m2 {
			m2 = c
		}
		if d > m3 {
			m3 = d
		}
		i += 4
	}

	for ; i < len(xs); i++ {
		v := xs[i]
		nan0 = nan0 || v != v
		if v > m0 {
			m0 = v
		}
	}

	return m0, m1, m2, m3, nan0 || nan1 || nan2 || nan3
}

// scanMinMax4 is the x86-64 combined scan: four min chains, four max chains, and
// a fused NaN check.
func scanMinMax4(xs []float64) (lo, hi float64, nan bool) {
	lo0, lo1, lo2, lo3 := xs[0], xs[0], xs[0], xs[0]
	hi0, hi1, hi2, hi3 := xs[0], xs[0], xs[0], xs[0]
	var nan0, nan1, nan2, nan3 bool
	i := 0
	limit := len(xs) - 3

	for i < limit {
		a, b, c, d := xs[i], xs[i+1], xs[i+2], xs[i+3]

		nan0 = nan0 || a != a
		nan1 = nan1 || b != b
		nan2 = nan2 || c != c
		nan3 = nan3 || d != d

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
		if c < lo2 {
			lo2 = c
		}
		if c > hi2 {
			hi2 = c
		}
		if d < lo3 {
			lo3 = d
		}
		if d > hi3 {
			hi3 = d
		}
		i += 4
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

	lo01, lo23 := lo0, lo2
	if lo1 < lo01 {
		lo01 = lo1
	}
	if lo3 < lo23 {
		lo23 = lo3
	}
	if lo23 < lo01 {
		lo01 = lo23
	}

	hi01, hi23 := hi0, hi2
	if hi1 > hi01 {
		hi01 = hi1
	}
	if hi3 > hi23 {
		hi23 = hi3
	}
	if hi23 > hi01 {
		hi01 = hi23
	}

	return lo01, hi01, nan0 || nan1 || nan2 || nan3
}

// addArch is the x86-64 tuned elementwise addition.
//
// Elementwise work has no dependency chain, so this is memory-bound. Four
// independent load/add/store triples per iteration keep the load and store
// units busy simultaneously.
func addArch(dst, xs, ys []float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 7

	for i < limit {
		dst[i] = xs[i] + ys[i]
		dst[i+1] = xs[i+1] + ys[i+1]
		dst[i+2] = xs[i+2] + ys[i+2]
		dst[i+3] = xs[i+3] + ys[i+3]
		dst[i+4] = xs[i+4] + ys[i+4]
		dst[i+5] = xs[i+5] + ys[i+5]
		dst[i+6] = xs[i+6] + ys[i+6]
		dst[i+7] = xs[i+7] + ys[i+7]
		i += 8
	}

	for ; i < n; i++ {
		dst[i] = xs[i] + ys[i]
	}
	return dst
}

// subArch is the x86-64 tuned elementwise subtraction.
func subArch(dst, xs, ys []float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 7

	for i < limit {
		dst[i] = xs[i] - ys[i]
		dst[i+1] = xs[i+1] - ys[i+1]
		dst[i+2] = xs[i+2] - ys[i+2]
		dst[i+3] = xs[i+3] - ys[i+3]
		dst[i+4] = xs[i+4] - ys[i+4]
		dst[i+5] = xs[i+5] - ys[i+5]
		dst[i+6] = xs[i+6] - ys[i+6]
		dst[i+7] = xs[i+7] - ys[i+7]
		i += 8
	}

	for ; i < n; i++ {
		dst[i] = xs[i] - ys[i]
	}
	return dst
}

// mulArch is the x86-64 tuned elementwise multiplication.
func mulArch(dst, xs, ys []float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 7

	for i < limit {
		dst[i] = xs[i] * ys[i]
		dst[i+1] = xs[i+1] * ys[i+1]
		dst[i+2] = xs[i+2] * ys[i+2]
		dst[i+3] = xs[i+3] * ys[i+3]
		dst[i+4] = xs[i+4] * ys[i+4]
		dst[i+5] = xs[i+5] * ys[i+5]
		dst[i+6] = xs[i+6] * ys[i+6]
		dst[i+7] = xs[i+7] * ys[i+7]
		i += 8
	}

	for ; i < n; i++ {
		dst[i] = xs[i] * ys[i]
	}
	return dst
}

// scaleArch is the x86-64 tuned scalar multiply.
func scaleArch(dst, xs []float64, k float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 7

	for i < limit {
		dst[i] = xs[i] * k
		dst[i+1] = xs[i+1] * k
		dst[i+2] = xs[i+2] * k
		dst[i+3] = xs[i+3] * k
		dst[i+4] = xs[i+4] * k
		dst[i+5] = xs[i+5] * k
		dst[i+6] = xs[i+6] * k
		dst[i+7] = xs[i+7] * k
		i += 8
	}

	for ; i < n; i++ {
		dst[i] = xs[i] * k
	}
	return dst
}

// addScalarArch is the x86-64 tuned scalar add.
func addScalarArch(dst, xs []float64, k float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 7

	for i < limit {
		dst[i] = xs[i] + k
		dst[i+1] = xs[i+1] + k
		dst[i+2] = xs[i+2] + k
		dst[i+3] = xs[i+3] + k
		dst[i+4] = xs[i+4] + k
		dst[i+5] = xs[i+5] + k
		dst[i+6] = xs[i+6] + k
		dst[i+7] = xs[i+7] + k
		i += 8
	}

	for ; i < n; i++ {
		dst[i] = xs[i] + k
	}
	return dst
}

// absArch is the x86-64 tuned absolute value.
func absArch(dst, xs []float64) []float64 {
	n := len(dst)
	i := 0
	limit := n - 7

	for i < limit {
		dst[i] = abs(xs[i])
		dst[i+1] = abs(xs[i+1])
		dst[i+2] = abs(xs[i+2])
		dst[i+3] = abs(xs[i+3])
		dst[i+4] = abs(xs[i+4])
		dst[i+5] = abs(xs[i+5])
		dst[i+6] = abs(xs[i+6])
		dst[i+7] = abs(xs[i+7])
		i += 8
	}

	for ; i < n; i++ {
		dst[i] = abs(xs[i])
	}
	return dst
}
