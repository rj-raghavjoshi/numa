package numa

import "testing"

func TestScalarFacadeForwards(t *testing.T) {
	xs := []float64{1, 2, 4}
	checks := []struct {
		name string
		got  []float64
		want []float64
	}{
		{"sub", SubScalar(xs, 1), []float64{0, 1, 3}},
		{"rsub", RSubScalar(xs, 1), []float64{0, -1, -3}},
		{"div", DivScalar(xs, 2), []float64{0.5, 1, 2}},
		{"rdiv", RDivScalar(xs, 8), []float64{8, 4, 2}},
	}
	for _, check := range checks {
		for i := range check.want {
			if check.got[i] != check.want[i] {
				t.Fatalf("%s[%d] = %v, want %v", check.name, i, check.got[i], check.want[i])
			}
		}
	}
	dst := make([]float64, len(xs))
	if got := SubScalarTo(dst, xs, 1); len(got) != len(xs) || got[2] != 3 {
		t.Fatalf("SubScalarTo = %v", got)
	}
}
