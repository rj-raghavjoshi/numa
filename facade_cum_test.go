package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the cumulative facade.
//
// A forwarder for a prefix operation can be wrong by swapping dst and xs, which
// is invisible on a symmetric test and on length alone. The checks below compare
// against vec and use non-palindromic inputs, and the NaN test pins the direction
// of the documented poisoning.
// ---------------------------------------------------------------------------

func TestCumulativeForwardersMatchVec(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5, 6, 7}

	compare := func(name string, got, want []float64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %v, want %v", name, i, got[i], want[i])
			}
		}
	}

	compare("CumSum", CumSum(xs), vec.CumSum(xs))
	compare("CumMax", CumMax(xs), vec.CumMax(xs))
	compare("CumMin", CumMin(xs), vec.CumMin(xs))

	// CumProd is arm64-reassociated, so compare with the implementation rather
	// than with hand values; the implementations are the same code path here.
	compare("CumProd", CumProd(xs), vec.CumProd(xs))
}

// TestCumulativeArgumentOrderForwarders uses a non-palindromic input so a swap of
// dst and xs is visible: CumSum of a reversed slice is a different sequence.
func TestCumulativeArgumentOrderForwarders(t *testing.T) {
	xs := []float64{1, 2, 4, 8}

	got := CumSum(xs)
	want := []float64{1, 3, 7, 15}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("CumSum[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// Writing into dst must not disturb xs, and must equal the allocating form.
	dst := make([]float64, len(xs))
	CumSumTo(dst, xs)
	for i := range want {
		if dst[i] != want[i] {
			t.Errorf("CumSumTo[%d] = %v, want %v", i, dst[i], want[i])
		}
	}
	if xs[0] != 1 || xs[3] != 8 {
		t.Errorf("CumSumTo mutated its input: %v", xs)
	}

	if CumCountTrue([]uint8{1, 0, 1})[2] != vec.CumCountTrue([]uint8{1, 0, 1})[2] {
		t.Error("CumCountTrue disagrees with vec")
	}
}

// TestCumulativeNaNPolicySurvivesFacade pins the forward direction of poisoning:
// nothing before the NaN is affected, and everything from it onward is NaN.
func TestCumulativeNaNPolicySurvivesFacade(t *testing.T) {
	xs := []float64{1, 2, math.NaN(), 4, 5}

	max := CumMax(xs)
	if max[0] != 1 || max[1] != 2 {
		t.Errorf("CumMax before NaN = %v, want [1 2 ...]", max[:2])
	}
	for i := 2; i < len(xs); i++ {
		if !math.IsNaN(max[i]) {
			t.Errorf("CumMax[%d] = %v, want NaN", i, max[i])
		}
	}

	min := CumMin(xs)
	if min[0] != 1 || min[1] != 1 {
		t.Errorf("CumMin before NaN = %v, want [1 1 ...]", min[:2])
	}
	if !math.IsNaN(min[4]) {
		t.Errorf("CumMin[4] = %v, want NaN", min[4])
	}
}

func TestCumulativeForwardersPanicLikeVec(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)
	shortI := make([]int, 2)
	longM := make([]uint8, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"CumSumTo", func() { CumSumTo(short, long) }},
		{"CumProdTo", func() { CumProdTo(short, long) }},
		{"CumMaxTo", func() { CumMaxTo(short, long) }},
		{"CumMinTo", func() { CumMinTo(short, long) }},
		{"CumCountTrueTo", func() { CumCountTrueTo(shortI, longM) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic on length mismatch", tc.name)
				}
			}()
			tc.fn()
		})
	}
}
