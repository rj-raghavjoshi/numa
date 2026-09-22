//go:build !arm64 && !amd64

package vec

func sumArch(xs []float64) float64 {
	var total float64
	for _, v := range xs {
		total += v
	}
	return total
}
