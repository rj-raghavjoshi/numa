//go:build !arm64 && !amd64

package vec

import "math"

// ---------------------------------------------------------------------------
// Generic fallback for architectures without a tuned implementation.
//
// These are deliberately the plain, readable versions. They are correct and
// serve as the reference that the tuned loops are checked against, so they
// should stay simple: any cleverness added here makes it harder to tell which
// behaviour is the intended one and which is a tuning side effect.
//
// They also matter for benchmarking. Comparing a tuned loop against this file
// on the same machine is how the tuning is justified; without a baseline the
// tuned version is just a claim.
// ---------------------------------------------------------------------------

func sumArch(xs []float64) float64 {
	var total float64
	for _, v := range xs {
		total += v
	}
	return total
}

func dotArch(xs, ys []float64) float64 {
	var total float64
	for i, v := range xs {
		total += v * ys[i]
	}
	return total
}

func sumSqArch(xs []float64) float64 {
	var total float64
	for _, v := range xs {
		total += v * v
	}
	return total
}

// The scan implementations below fuse the NaN check into the loop, matching the
// contract of the tuned versions: the result is NaN if any element is NaN. The
// reference implementations are kept naive and readable, but they are not allowed
// to differ in behaviour, only in speed.
//
// See nan.go for the policy and arch_arm64.go's minArch for why the check is fused
// rather than a separate pass.

func minArch(xs []float64) float64 {
	m := xs[0]
	nan := false
	for _, v := range xs[1:] {
		nan = nan || v != v
		if v < m {
			m = v
		}
	}
	if nan || xs[0] != xs[0] {
		return math.NaN()
	}
	return m
}

func maxArch(xs []float64) float64 {
	m := xs[0]
	nan := false
	for _, v := range xs[1:] {
		nan = nan || v != v
		if v > m {
			m = v
		}
	}
	if nan || xs[0] != xs[0] {
		return math.NaN()
	}
	return m
}

func minMaxArch(xs []float64) (lo, hi float64) {
	lo, hi = xs[0], xs[0]
	nan := xs[0] != xs[0]
	for _, v := range xs[1:] {
		nan = nan || v != v
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	if nan {
		return math.NaN(), math.NaN()
	}
	return lo, hi
}

func addArch(dst, xs, ys []float64) []float64 {
	for i := range dst {
		dst[i] = xs[i] + ys[i]
	}
	return dst
}

func subArch(dst, xs, ys []float64) []float64 {
	for i := range dst {
		dst[i] = xs[i] - ys[i]
	}
	return dst
}

func mulArch(dst, xs, ys []float64) []float64 {
	for i := range dst {
		dst[i] = xs[i] * ys[i]
	}
	return dst
}

func scaleArch(dst, xs []float64, k float64) []float64 {
	for i := range dst {
		dst[i] = xs[i] * k
	}
	return dst
}

func addScalarArch(dst, xs []float64, k float64) []float64 {
	for i := range dst {
		dst[i] = xs[i] + k
	}
	return dst
}

func absArch(dst, xs []float64) []float64 {
	for i := range dst {
		dst[i] = abs(xs[i])
	}
	return dst
}

// ---------------------------------------------------------------------------
// Arg reductions.
//
// The generic reference is a single running extreme with one NaN flag. It is
// deliberately not parallelised: this file is the baseline the tuned scans are
// measured against, so making it clever would destroy its only purpose.
//
// The `m != m` term in the final test catches a NaN sitting at index 0. The loop
// starts at index 1, so a leading NaN never passes through the in-loop flag, and
// because every ordering comparison against NaN is false it is never replaced.
// ---------------------------------------------------------------------------

func argMinArch(xs []float64) int {
	m := xs[0]
	idx := 0
	nan := false
	for i := 1; i < len(xs); i++ {
		v := xs[i]
		nan = nan || v != v
		if v < m {
			m, idx = v, i
		}
	}
	if nan || m != m {
		return -1
	}
	return idx
}

func argMaxArch(xs []float64) int {
	m := xs[0]
	idx := 0
	nan := false
	for i := 1; i < len(xs); i++ {
		v := xs[i]
		nan = nan || v != v
		if v > m {
			m, idx = v, i
		}
	}
	if nan || m != m {
		return -1
	}
	return idx
}

func argMinMaxArch(xs []float64) (int, int) {
	lo, hi := xs[0], xs[0]
	loi, hii := 0, 0
	nan := xs[0] != xs[0]
	for i := 1; i < len(xs); i++ {
		v := xs[i]
		nan = nan || v != v
		if v < lo {
			lo, loi = v, i
		}
		if v > hi {
			hi, hii = v, i
		}
	}
	if nan {
		return -1, -1
	}
	return loi, hii
}

// ---------------------------------------------------------------------------
// Statistics helpers.
//
// The portable reference for the deviation reductions in stats.go: a single
// accumulator, no independent chains. Kept plain on purpose -- it is the baseline
// the tuned versions are measured against.
// ---------------------------------------------------------------------------

func dotDevArch(xs, ys []float64, cx, cy float64) float64 {
	n := min(len(xs), len(ys))
	var total float64
	for i := 0; i < n; i++ {
		total += (xs[i] - cx) * (ys[i] - cy)
	}
	return total
}

func sumAbsDevArch(xs []float64, center float64) float64 {
	var total float64
	for _, v := range xs {
		total += abs(v - center)
	}
	return total
}
