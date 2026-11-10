package numa

import "testing"

func TestCountFacadeForwards(t *testing.T) {
	xs := []float64{0, 1, 2, 3}
	if got := CountGreaterScalar(xs, 1); got != 2 {
		t.Fatalf("CountGreaterScalar = %d", got)
	}
	if got := CountGreaterEqualScalar(xs, 1); got != 3 {
		t.Fatalf("CountGreaterEqualScalar = %d", got)
	}
}
