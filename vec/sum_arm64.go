//go:build arm64

package vec

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
