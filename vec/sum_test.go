package vec

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Sum and Mean.
// ---------------------------------------------------------------------------

func TestSumMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		xs := randomSlice(n, int64(n)+1)

		if got, want := Sum(xs), refSum(xs); !closeEnough(got, want) {
			t.Errorf("n=%d: Sum = %v, reference = %v", n, got, want)
		}
	}
}

func TestSumArchMatchesReference(t *testing.T) {
	// Test sumArch directly so the tuned loop is exercised even at n=0, which the
	// public Sum short-circuits.
	for _, n := range boundaryLengths() {
		xs := randomSlice(n, int64(n)+7)

		if got, want := sumArch(xs), refSum(xs); !closeEnough(got, want) {
			t.Errorf("n=%d: sumArch = %v, reference = %v", n, got, want)
		}
	}
}

func TestSumEmptyAndNil(t *testing.T) {
	if got := Sum(nil); got != 0 {
		t.Errorf("Sum(nil) = %v, want 0", got)
	}
	if got := Sum([]float64{}); got != 0 {
		t.Errorf("Sum([]) = %v, want 0", got)
	}
}

// TestSumConsumesEveryElement catches skipped and double-counted elements.
//
// Comparing against the reference is not always enough to make a skipped block
// obvious, because a wrong answer can still look plausible. This test makes it
// explicit: perturb one element at a time and require the result to move.
//
// This is the structural defence against the class of bug that reading the source
// cannot catch. See ../docs/learn/03-the-truth.md section 3.
func TestSumConsumesEveryElement(t *testing.T) {
	const n = 64

	base := randomSlice(n, 99)
	baseSum := Sum(base)

	for i := range n {
		perturbed := make([]float64, n)
		copy(perturbed, base)
		perturbed[i] += 1.0

		if got := Sum(perturbed); got == baseSum {
			t.Errorf("element %d appears to be ignored: Sum unchanged after perturbation", i)
		}
	}
}

// TestSumKnownValues pins results against values worked out by hand, so a
// catastrophic error cannot hide behind the relative tolerance used elsewhere.
func TestSumKnownValues(t *testing.T) {
	// xs[i] = i, so the sum of 0..n-1 is n(n-1)/2. For these n it is exactly
	// representable, so an exact comparison is legitimate.
	for _, n := range []int{1, 2, 3, 4, 5, 8, 9, 16, 17, 100} {
		xs := make([]float64, n)
		for i := range xs {
			xs[i] = float64(i)
		}
		want := float64(n * (n - 1) / 2)
		if got := Sum(xs); got != want {
			t.Errorf("n=%d: Sum = %v, want %v", n, got, want)
		}
	}
}

// TestSumIsNotLessAccurateThanNaive tests the stability claim.
//
// Agreement with the naive loop is a different property from accuracy: a
// reassociated sum can differ slightly from the naive loop while being *closer* to
// the true answer. This test checks the second property.
func TestSumIsNotLessAccurateThanNaive(t *testing.T) {
	cases := []struct {
		name string
		xs   []float64
	}{
		{"all ones", ones(1000)},
		{"small plus large", append(ones(1000), 1e16)},
		{"wide dynamic range", wideRange(1000)},
		{"near cancellation", cancelling(1000)},
		{"random", randomSlice(1000, 12345)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Sum(tc.xs)
			exact, _ := exactSum(tc.xs).Float64()
			naive := refSum(tc.xs)

			gotErr := math.Abs(got - exact)
			naiveErr := math.Abs(naive - exact)
			scale := math.Max(1, math.Abs(exact))

			// Generous factor: this detects gross instability, not last-bit
			// differences, which are legitimate.
			if gotErr > 8*naiveErr+1e-9*scale {
				t.Errorf("Sum = %v (err %g) vs naive = %v (err %g), exact = %v: tuned path less accurate",
					got, gotErr, naive, naiveErr, exact)
			}
		})
	}
}

func TestMeanMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue // covered separately
		}
		xs := randomSlice(n, int64(n)+11)

		if got, want := Mean(xs), refSum(xs)/float64(n); !closeEnough(got, want) {
			t.Errorf("n=%d: Mean = %v, reference = %v", n, got, want)
		}
	}
}

func TestMeanEmpty(t *testing.T) {
	if got := Mean(nil); got != 0 {
		t.Errorf("Mean(nil) = %v, want 0", got)
	}
}

// TestMeanIsStable checks that summing then dividing once does not accumulate the
// per-element error that `total += v/n` would.
func TestMeanIsStable(t *testing.T) {
	xs := make([]float64, 100000)
	for i := range xs {
		xs[i] = 1e15
	}
	if got := Mean(xs); !closeEnough(got, 1e15) {
		t.Errorf("Mean of 1e5 copies of 1e15 = %v, want 1e15", got)
	}
}

// TestMeanIsSumOverLen checks the documented relationship between the two.
func TestMeanIsSumOverLen(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, int64(n)+111)

		if !closeEnough(Mean(xs), Sum(xs)/float64(n)) {
			t.Errorf("n=%d: Mean = %v, Sum/n = %v", n, Mean(xs), Sum(xs)/float64(n))
		}
	}
}
