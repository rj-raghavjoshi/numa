package vec

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Min / Max / MinMax.
//
// The headline test here is TestScansConsumeEveryElement. A scan bug usually
// shows up as "the answer is right, but only by luck" -- the true extreme
// happened to sit in a block that was scanned. That test forces every position to
// matter by planting a new extreme at each index in turn.
// ---------------------------------------------------------------------------

func TestScansAgreeWithReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := randomSlice(n, int64(n)+41)

		if got, want := Min(xs), refMin(xs); got != want {
			t.Errorf("n=%d: Min = %v, reference = %v", n, got, want)
		}
		if got, want := Max(xs), refMax(xs); got != want {
			t.Errorf("n=%d: Max = %v, reference = %v", n, got, want)
		}

		lo, hi := MinMax(xs)
		if want := refMin(xs); lo != want {
			t.Errorf("n=%d: MinMax lo = %v, reference = %v", n, lo, want)
		}
		if want := refMax(xs); hi != want {
			t.Errorf("n=%d: MinMax hi = %v, reference = %v", n, hi, want)
		}
	}
}

func TestScansConsumeEveryElement(t *testing.T) {
	const n = 64

	for i := range n {
		// A slice of zeros with a single negative spike at i.
		lo := make([]float64, n)
		lo[i] = -1

		if got := Min(lo); got != -1 {
			t.Errorf("Min: spike at %d not found, got %v", i, got)
		}
		if got, _ := MinMax(lo); got != -1 {
			t.Errorf("MinMax lo: spike at %d not found, got %v", i, got)
		}

		// A slice of zeros with a single positive spike at i.
		hi := make([]float64, n)
		hi[i] = 1

		if got := Max(hi); got != 1 {
			t.Errorf("Max: spike at %d not found, got %v", i, got)
		}
		if _, got := MinMax(hi); got != 1 {
			t.Errorf("MinMax hi: spike at %d not found, got %v", i, got)
		}
	}
}

func TestScansEmpty(t *testing.T) {
	if got := Min(nil); !math.IsInf(got, 1) {
		t.Errorf("Min(nil) = %v, want +Inf", got)
	}
	if got := Max(nil); !math.IsInf(got, -1) {
		t.Errorf("Max(nil) = %v, want -Inf", got)
	}

	lo, hi := MinMax(nil)
	if !math.IsInf(lo, 1) || !math.IsInf(hi, -1) {
		t.Errorf("MinMax(nil) = (%v, %v), want (+Inf, -Inf)", lo, hi)
	}
}

func TestScansSingleElement(t *testing.T) {
	xs := []float64{42}
	if got := Min(xs); got != 42 {
		t.Errorf("Min = %v, want 42", got)
	}
	if got := Max(xs); got != 42 {
		t.Errorf("Max = %v, want 42", got)
	}
	lo, hi := MinMax(xs)
	if lo != 42 || hi != 42 {
		t.Errorf("MinMax = (%v, %v), want (42, 42)", lo, hi)
	}
}

func TestScansNegativeAndLarge(t *testing.T) {
	xs := make([]float64, 100)
	for i := range xs {
		xs[i] = float64(i) - 50
	}
	if got := Min(xs); got != -50 {
		t.Errorf("Min = %v, want -50", got)
	}
	if got := Max(xs); got != 49 {
		t.Errorf("Max = %v, want 49", got)
	}
}

// ---------------------------------------------------------------------------
// NaN policy.
//
// The documented contract is "NaN wins": any NaN in the input poisons the
// result. These tests pin that decision, because the natural implementation of a
// scan (using `<` or `>`) quietly does the opposite -- comparisons against NaN
// are always false, so a NaN is simply skipped.
//
// The position-dependent cases are the important ones: a NaN at index 0 and a
// NaN at the last index must behave identically. Without an explicit check they
// do not.
// ---------------------------------------------------------------------------

func TestScansPropagateNaN(t *testing.T) {
	const n = 16

	for i := range n {
		xs := ones(n) // all 1.0, so the NaN is unambiguous
		xs[i] = math.NaN()

		if got := Min(xs); !math.IsNaN(got) {
			t.Errorf("Min with NaN at %d = %v, want NaN", i, got)
		}
		if got := Max(xs); !math.IsNaN(got) {
			t.Errorf("Max with NaN at %d = %v, want NaN", i, got)
		}
		lo, hi := MinMax(xs)
		if !math.IsNaN(lo) || !math.IsNaN(hi) {
			t.Errorf("MinMax with NaN at %d = (%v, %v), want (NaN, NaN)", i, lo, hi)
		}
	}
}

func TestScansPropagateNaNInLargeSlice(t *testing.T) {
	// A length that exercises the unrolled main loop and the tail loop, with the
	// NaN somewhere in the middle rather than at either end.
	xs := ones(1000)
	xs[500] = math.NaN()

	if got := Min(xs); !math.IsNaN(got) {
		t.Errorf("Min = %v, want NaN", got)
	}
	if got := Max(xs); !math.IsNaN(got) {
		t.Errorf("Max = %v, want NaN", got)
	}
}

func TestScansPropagateNaNAtBoundaries(t *testing.T) {
	// The first and last positions are the ones most likely to be special-cased
	// by the loop structure, since the scans seed their accumulators from xs[0].
	for _, n := range []int{1, 2, 3, 4, 5, 8, 9, 16, 17, 64, 65} {
		for _, i := range []int{0, n / 2, n - 1} {
			xs := ones(n)
			xs[i] = math.NaN()

			if got := Min(xs); !math.IsNaN(got) {
				t.Errorf("n=%d NaN at %d: Min = %v, want NaN", n, i, got)
			}
			if got := Max(xs); !math.IsNaN(got) {
				t.Errorf("n=%d NaN at %d: Max = %v, want NaN", n, i, got)
			}
		}
	}
}

// TestScansSmallInputsNaNBehaviour exercises the fused NaN check on the small and
// degenerate inputs that the unrolled loops handle via their tail paths.
//
// This replaced a unit test of the old standalone hasNaN helper. The check now
// lives inside each scan loop, so the observable contract is worth testing at the
// scan level rather than in isolation.
func TestScansSmallInputsNaNBehaviour(t *testing.T) {
	cases := []struct {
		name string
		xs   []float64
		nan  bool
	}{
		{"no nan", []float64{1, 2, 3}, false},
		{"infinities are not nan", []float64{math.Inf(1), math.Inf(-1)}, false},
		{"nan first", []float64{math.NaN(), 1, 2}, true},
		{"nan last", []float64{1, 2, math.NaN()}, true},
		{"nan only", []float64{math.NaN()}, true},
		{"many nan", []float64{math.NaN(), math.NaN()}, true},
		{"mixed inf and nan", []float64{math.Inf(1), math.NaN(), math.Inf(-1)}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Min(tc.xs); math.IsNaN(got) != tc.nan {
				t.Errorf("Min = %v, want NaN=%v", got, tc.nan)
			}
			if got := Max(tc.xs); math.IsNaN(got) != tc.nan {
				t.Errorf("Max = %v, want NaN=%v", got, tc.nan)
			}
			lo, hi := MinMax(tc.xs)
			if math.IsNaN(lo) != tc.nan || math.IsNaN(hi) != tc.nan {
				t.Errorf("MinMax = (%v,%v), want NaN=%v", lo, hi, tc.nan)
			}
		})
	}
}

// TestScansInfinitiesAreNotNaN checks that the NaN check does not misfire on
// infinities, which are legitimate values that the scans must handle.
func TestScansInfinitiesAreNotNaN(t *testing.T) {
	xs := []float64{1, math.Inf(1), 2}

	if got := Max(xs); !math.IsInf(got, 1) {
		t.Errorf("Max = %v, want +Inf", got)
	}
	if got := Min(xs); got != 1 {
		t.Errorf("Min = %v, want 1", got)
	}

	lo, hi := MinMax(xs)
	if lo != 1 || !math.IsInf(hi, 1) {
		t.Errorf("MinMax = (%v, %v), want (1, +Inf)", lo, hi)
	}

	negs := []float64{1, math.Inf(-1), 2}
	if got := Min(negs); !math.IsInf(got, -1) {
		t.Errorf("Min = %v, want -Inf", got)
	}
}
