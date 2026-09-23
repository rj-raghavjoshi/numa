//go:build !arm64 && !amd64

package vec

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

func minArch(xs []float64) float64 {
	m := xs[0]
	for _, v := range xs[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxArch(xs []float64) float64 {
	m := xs[0]
	for _, v := range xs[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func minMaxArch(xs []float64) (lo, hi float64) {
	lo, hi = xs[0], xs[0]
	for _, v := range xs[1:] {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
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
