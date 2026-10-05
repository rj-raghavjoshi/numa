package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the signal facade.
//
// The lagged operations have two contracts a forwarder can break invisibly:
// argument order (dst and xs are both []float64, so a swap compiles and produces
// wrong output only when dst is not a scratch buffer), and the direction of the
// lag. Both are checked with non-palindromic inputs.
// ---------------------------------------------------------------------------

func TestSignalForwardersMatchVec(t *testing.T) {
	xs := []float64{1, 3, 6, 10, 15}

	compare := func(name string, got, want []float64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
		}
		for i := range want {
			if math.IsNaN(want[i]) != math.IsNaN(got[i]) {
				t.Errorf("%s[%d] = %v, want %v", name, i, got[i], want[i])
			} else if !math.IsNaN(want[i]) && got[i] != want[i] {
				t.Errorf("%s[%d] = %v, want %v", name, i, got[i], want[i])
			}
		}
	}

	compare("Shift", Shift(xs, 2), vec.Shift(xs, 2))
	compare("Diff", Diff(xs, 1), vec.Diff(xs, 1))
	compare("Rate", Rate(xs, 2), vec.Rate(xs, 2))
	compare("ValueWhen", ValueWhen([]uint8{0, 1, 0, 1, 0}, xs), vec.ValueWhen([]uint8{0, 1, 0, 1, 0}, xs))
	compare("HighestSince", HighestSince([]uint8{1, 0, 0, 1, 0}, xs), vec.HighestSince([]uint8{1, 0, 0, 1, 0}, xs))
	compare("LowestSince", LowestSince([]uint8{1, 0, 0, 1, 0}, xs), vec.LowestSince([]uint8{1, 0, 0, 1, 0}, xs))
}

// TestShiftLagDirectionSurvivesFacade pins that a positive lag looks backward. A
// forwarder that forwarded the negation, or swapped the argument order, would fail
// here while passing a length-only test.
func TestShiftLagDirectionSurvivesFacade(t *testing.T) {
	xs := []float64{1, 2, 3, 4}

	back := Shift(xs, 1)
	if !math.IsNaN(back[0]) || back[1] != 1 || back[2] != 2 || back[3] != 3 {
		t.Errorf("Shift(xs,1) = %v, want [NaN 1 2 3]", back)
	}

	fwd := Shift(xs, -1)
	if fwd[0] != 2 || fwd[1] != 3 || fwd[2] != 4 || !math.IsNaN(fwd[3]) {
		t.Errorf("Shift(xs,-1) = %v, want [2 3 4 NaN]", fwd)
	}
}

func TestSignalRisingFallingForwarders(t *testing.T) {
	xs := []float64{1, 2, 3, 2, 1}

	up := Rising(xs, 1)
	wantUp := []uint8{0, 1, 1, 0, 0}
	for i := range wantUp {
		if up[i] != wantUp[i] {
			t.Errorf("Rising[%d] = %d, want %d", i, up[i], wantUp[i])
		}
	}

	down := Falling(xs, 1)
	wantDown := []uint8{0, 0, 0, 1, 1}
	for i := range wantDown {
		if down[i] != wantDown[i] {
			t.Errorf("Falling[%d] = %d, want %d", i, down[i], wantDown[i])
		}
	}
}

func TestCrossForwardersMatchVec(t *testing.T) {
	a := []float64{1, 3, 1}
	b := []float64{2, 2, 2}

	cmpMask := func(name string, got, want []uint8) {
		t.Helper()
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %d, want %d", name, i, got[i], want[i])
			}
		}
	}

	cmpMask("CrossOver", CrossOver(a, b), vec.CrossOver(a, b))
	cmpMask("CrossUnder", CrossUnder(a, b), vec.CrossUnder(a, b))

	got := Cross(a, b)
	want := vec.Cross(a, b)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Cross[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestBarsSinceForwarder(t *testing.T) {
	mask := []uint8{0, 1, 0, 0}
	got := BarsSince(mask)
	want := []int{-1, 0, 1, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("BarsSince[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSignalForwardersPanicLikeVec(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"ShiftTo", func() { ShiftTo(short, long, 1) }},
		{"DiffTo", func() { DiffTo(short, long, 1) }},
		{"RateTo", func() { RateTo(short, long, 1) }},
		{"RateTo zero lag", func() { RateTo(short, short, 0) }},
		{"Rising zero lag", func() { Rising(short, 0) }},
		{"ValueWhen length", func() { ValueWhen(make([]uint8, 2), long) }},
		{"HighestSince length", func() { HighestSince(make([]uint8, 2), long) }},
		{"LowestSince length", func() { LowestSince(make([]uint8, 2), long) }},
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
