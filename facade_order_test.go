package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the arg/order facade.
//
// The contracts a forwarder can silently break here:
//
//   - the -1 sentinel, including its two meanings (empty input, NaN present)
//   - the ascending direction of Rank, and the average-of-ties rule
//   - the interpolation convention, which a forwarder could inadvertently change
//     by forwarding to Percentile instead of Quantile
// ---------------------------------------------------------------------------

func TestArgForwardersMatchVec(t *testing.T) {
	xs := []float64{3, 1, 4, 1, 5}

	if got, want := ArgMin(xs), vec.ArgMin(xs); got != want {
		t.Errorf("ArgMin = %d, want %d", got, want)
	}
	if got, want := ArgMax(xs), vec.ArgMax(xs); got != want {
		t.Errorf("ArgMax = %d, want %d", got, want)
	}
	lo, hi := ArgMinMax(xs)
	wlo, whi := vec.ArgMinMax(xs)
	if lo != wlo || hi != whi {
		t.Errorf("ArgMinMax = (%d,%d), want (%d,%d)", lo, hi, wlo, whi)
	}
	if got, want := HasNaN(xs), vec.HasNaN(xs); got != want {
		t.Errorf("HasNaN = %v, want %v", got, want)
	}
}

// TestArgForwarderSentinel pins -1 for both of its meanings through the facade.
func TestArgForwarderSentinel(t *testing.T) {
	if got := ArgMin(nil); got != -1 {
		t.Errorf("ArgMin(nil) = %d, want -1", got)
	}
	if got := ArgMax([]float64{math.NaN()}); got != -1 {
		t.Errorf("ArgMax([NaN]) = %d, want -1", got)
	}
	lo, hi := ArgMinMax([]float64{1, math.NaN()})
	if lo != -1 || hi != -1 {
		t.Errorf("ArgMinMax with NaN = (%d,%d), want (-1,-1)", lo, hi)
	}
	if !HasNaN([]float64{1, math.NaN()}) {
		t.Error("HasNaN missed a NaN through the facade")
	}
}

func TestOrderForwardersMatchVec(t *testing.T) {
	xs := []float64{4, 1, 3, 2}

	if got, want := Median(xs), vec.Median(xs); got != want {
		t.Errorf("Median = %v, want %v", got, want)
	}
	if got, want := Quantile(xs, 0.25), vec.Quantile(xs, 0.25); got != want {
		t.Errorf("Quantile = %v, want %v", got, want)
	}
	if got, want := Percentile(xs, 75), vec.Percentile(xs, 75); got != want {
		t.Errorf("Percentile = %v, want %v", got, want)
	}
	if got, want := MAD(xs), vec.MAD(xs); got != want {
		t.Errorf("MAD = %v, want %v", got, want)
	}

	// Quantile(0.5) and Median must agree; a forwarder that swapped the q and p
	// conventions would diverge here.
	if Median(xs) != Quantile(xs, 0.5) {
		t.Error("Median and Quantile(0.5) disagree through the facade")
	}
	// Percentile(p) must equal Quantile(p/100), not Quantile(p).
	if Percentile(xs, 25) != Quantile(xs, 0.25) {
		t.Error("Percentile and Quantile disagree on the scale convention")
	}
}

// TestRankForwarderDirectionAndTies checks the ascending direction and the
// average-of-ties rule, the two properties a forwarding mistake would disturb.
func TestRankForwarderDirectionAndTies(t *testing.T) {
	xs := []float64{10, 20, 20, 30}
	got := Rank(xs)
	want := []float64{1, 2.5, 2.5, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Rank[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestSortForwarders(t *testing.T) {
	xs := []float64{3, 1, 2}
	cp := SortCopy(xs)
	if cp[0] != 1 || cp[1] != 2 || cp[2] != 3 {
		t.Errorf("SortCopy = %v, want [1 2 3]", cp)
	}
	if xs[0] != 3 {
		t.Error("SortCopy modified its input through the facade")
	}

	SortInPlace(xs)
	if xs[0] != 1 || xs[2] != 3 {
		t.Errorf("SortInPlace = %v, want ascending", xs)
	}
}

func TestOrderForwardersPanicLikeVec(t *testing.T) {
	xs := make([]float64, 4)
	small := make([]float64, 2)

	cases := []struct {
		name string
		fn   func()
	}{
		{"Quantile q>1", func() { Quantile(xs, 2) }},
		{"Percentile p>100", func() { Percentile(xs, 200) }},
		{"MedianInto small buf", func() { MedianInto(xs, small) }},
		{"QuantileInto small buf", func() { QuantileInto(xs, 0.5, small) }},
		{"RankTo length", func() { RankTo(small, xs) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}
