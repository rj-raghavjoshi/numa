//go:build amd64

package vec

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