package vec

import "testing"

func TestCountGreaterScalar(t *testing.T) {
	xs := []float64{0, 1, 2, 3}
	if got := CountGreaterScalar(xs, 1); got != 2 {
		t.Fatalf("CountGreaterScalar = %d, want 2", got)
	}
	if got := CountGreaterEqualScalar(xs, 1); got != 3 {
		t.Fatalf("CountGreaterEqualScalar = %d, want 3", got)
	}
	if got := CountGreaterScalar(nil, 0); got != 0 {
		t.Fatalf("empty = %d", got)
	}
}
