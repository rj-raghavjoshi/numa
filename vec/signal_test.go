package vec

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for lag, difference, and signal primitives.
//
// The failure modes worth pinning:
//
//   - the NaN boundary being off by one for a positive lag (a prefix) versus a
//     negative lag (a suffix), which are easy to get inconsistently wrong
//   - a lag larger than the input, where every output must be NaN rather than a
//     partial or wrapped result
//   - a cross detector that fires on the same bar as a touch, or that treats NaN
//     as an ordering
//   - BarsSince and the *Since family mishandling the "no occurrence yet" case,
//     which is the easy one to leave as a zero
// ---------------------------------------------------------------------------

func refShift(xs []float64, lag int) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		j := i - lag
		if j < 0 || j >= len(xs) {
			out[i] = math.NaN()
		} else {
			out[i] = xs[j]
		}
	}
	return out
}

func refDiff(xs []float64, lag int) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		j := i - lag
		if j < 0 || j >= len(xs) {
			out[i] = math.NaN()
		} else {
			out[i] = xs[i] - xs[j]
		}
	}
	return out
}

func refRate(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		j := i - n
		if j < 0 || j >= len(xs) {
			out[i] = math.NaN()
		} else if xs[j] == 0 {
			out[i] = 0
		} else {
			out[i] = (xs[i] - xs[j]) / xs[j]
		}
	}
	return out
}

// TestLaggedOpsMatchReference checks Shift, Diff and Rate against scalar
// references across every boundary length and a spread of lags, including lags
// that exceed the input and negative lags.
func TestLaggedOpsMatchReference(t *testing.T) {
	lags := []int{-5, -2, -1, 0, 1, 2, 3, 4, 5, 8, 100}

	for _, n := range boundaryLengths() {
		for _, lag := range lags {
			if n == 0 {
				continue
			}
			// Positive values, so that Rate's denominator is nonzero and the
			// reference and implementation agree on the general case.
			xs := benchPos(n)

			assertFloats(t, "Shift", Shift(xs, lag), refShift(xs, lag), n, lag)
			assertFloats(t, "Diff", Diff(xs, lag), refDiff(xs, lag), n, lag)
		}
	}

	// Rate takes a strictly positive lag by contract.
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		xs := benchPos(n)
		for _, lag := range []int{1, 2, 3, 4, 5, 8, 100} {
			assertFloats(t, "Rate", Rate(xs, lag), refRate(xs, lag), n, lag)
		}
	}
}

func assertFloats(t *testing.T, name string, got, want []float64, n, lag int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s n=%d lag=%d: length %d, want %d", name, n, lag, len(got), len(want))
	}
	for i := range want {
		if math.IsNaN(want[i]) != math.IsNaN(got[i]) {
			t.Fatalf("%s n=%d lag=%d i=%d: got %v, want %v", name, n, lag, i, got[i], want[i])
		}
		if !math.IsNaN(want[i]) && got[i] != want[i] {
			t.Fatalf("%s n=%d lag=%d i=%d: got %v, want %v", name, n, lag, i, got[i], want[i])
		}
	}
}

// TestLagLargerThanInput makes the degenerate case explicit: when the lag exceeds
// the input there is no overlap at all, so every output is NaN and nothing wraps.
func TestLagLargerThanInput(t *testing.T) {
	xs := []float64{1, 2, 3}
	for _, lag := range []int{3, 4, 1000} {
		for i, v := range Shift(xs, lag) {
			if !math.IsNaN(v) {
				t.Errorf("Shift(lag=%d)[%d] = %v, want NaN", lag, i, v)
			}
		}
		for i, v := range Diff(xs, lag) {
			if !math.IsNaN(v) {
				t.Errorf("Diff(lag=%d)[%d] = %v, want NaN", lag, i, v)
			}
		}
	}
	// A negative lag at least as large as the input is likewise all NaN.
	for i, v := range Shift(xs, -3) {
		if !math.IsNaN(v) {
			t.Errorf("Shift(lag=-3)[%d] = %v, want NaN", i, v)
		}
	}
}

func TestShiftKnownValues(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5}

	got := Shift(xs, 2)
	want := []float64{math.NaN(), math.NaN(), 1, 2, 3}
	for i := range want {
		if math.IsNaN(want[i]) {
			if !math.IsNaN(got[i]) {
				t.Errorf("Shift(2)[%d] = %v, want NaN", i, got[i])
			}
			continue
		}
		if got[i] != want[i] {
			t.Errorf("Shift(2)[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// A negative lag leads rather than trails.
	lead := Shift(xs, -2)
	wantLead := []float64{3, 4, 5, math.NaN(), math.NaN()}
	for i := range wantLead {
		if math.IsNaN(wantLead[i]) {
			if !math.IsNaN(lead[i]) {
				t.Errorf("Shift(-2)[%d] = %v, want NaN", i, lead[i])
			}
			continue
		}
		if lead[i] != wantLead[i] {
			t.Errorf("Shift(-2)[%d] = %v, want %v", i, lead[i], wantLead[i])
		}
	}

	// Shift by zero is an identity copy, not a NaN-filled result.
	same := Shift(xs, 0)
	for i := range xs {
		if same[i] != xs[i] {
			t.Errorf("Shift(0)[%d] = %v, want %v", i, same[i], xs[i])
		}
	}
}

func TestDiffKnownValues(t *testing.T) {
	xs := []float64{1, 3, 6, 10, 15}
	got := Diff(xs, 1)
	want := []float64{math.NaN(), 2, 3, 4, 5}
	for i := 1; i < len(want); i++ {
		if got[i] != want[i] {
			t.Errorf("Diff(1)[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if !math.IsNaN(got[0]) {
		t.Errorf("Diff(1)[0] = %v, want NaN", got[0])
	}

	// A zero lag is exactly zero change everywhere, including index 0.
	for i, v := range Diff(xs, 0) {
		if v != 0 {
			t.Errorf("Diff(0)[%d] = %v, want 0", i, v)
		}
	}
}

// TestRateZeroDenominatorIsZero pins the deliberate choice that a flat base gives 0
// rather than an infinity.
func TestRateZeroDenominatorIsZero(t *testing.T) {
	xs := []float64{0, 5, 0, 0}
	got := Rate(xs, 2)
	// index 2: base is xs[0]=0 -> 0; index 3: base is xs[1]=5 -> (0-5)/5 = -1
	if got[2] != 0 {
		t.Errorf("Rate with zero base = %v, want 0", got[2])
	}
	if math.Abs(got[3]+1) > 1e-15 {
		t.Errorf("Rate[3] = %v, want -1", got[3])
	}
}

func TestRisingFalling(t *testing.T) {
	xs := []float64{1, 2, 3, 2, 1, 5}

	up := Rising(xs, 1)
	wantUp := []uint8{0, 1, 1, 0, 0, 1}
	for i := range wantUp {
		if up[i] != wantUp[i] {
			t.Errorf("Rising[%d] = %d, want %d", i, up[i], wantUp[i])
		}
	}

	down := Falling(xs, 1)
	wantDown := []uint8{0, 0, 0, 1, 1, 0}
	for i := range wantDown {
		if down[i] != wantDown[i] {
			t.Errorf("Falling[%d] = %d, want %d", i, down[i], wantDown[i])
		}
	}

	// The first n positions have no history and must not be marked rising.
	if Rising(xs, 3)[2] != 0 {
		t.Error("Rising reported a signal without enough history")
	}
}

// TestCrossSemantics pins the boundary rule: a cross requires the previous bar to
// be on or beyond the other series and the current bar to be strictly past it. A
// mere touch must not fire, and a NaN must suppress rather than invent a signal.
func TestCrossSemantics(t *testing.T) {
	a := []float64{1, 2, 2, 3, 1}
	b := []float64{2, 2, 2, 2, 2}

	over := CrossOver(a, b)
	wantOver := []uint8{0, 0, 0, 1, 0}
	for i := range wantOver {
		if over[i] != wantOver[i] {
			t.Errorf("CrossOver[%d] = %d, want %d", i, over[i], wantOver[i])
		}
	}

	under := CrossUnder(a, b)
	// index 4: a[3]=3 >= 2 and a[4]=1 < 2
	wantUnder := []uint8{0, 0, 0, 0, 1}
	for i := range wantUnder {
		if under[i] != wantUnder[i] {
			t.Errorf("CrossUnder[%d] = %d, want %d", i, under[i], wantUnder[i])
		}
	}

	signed := Cross(a, b)
	wantSigned := []int8{0, 0, 0, 1, -1}
	for i := range wantSigned {
		if signed[i] != wantSigned[i] {
			t.Errorf("Cross[%d] = %d, want %d", i, signed[i], wantSigned[i])
		}
	}

	// A touch (equality then rise from equality) must not be a cross on the bar it
	// becomes equal, and NaN must not cross.
	nanA := []float64{math.NaN(), 5}
	nanB := []float64{1, 1}
	if CrossOver(nanA, nanB)[1] != 0 {
		t.Error("CrossOver fired across a NaN")
	}
}

func TestBarsSince(t *testing.T) {
	mask := []uint8{0, 0, 1, 0, 0, 1, 0}
	got := BarsSince(mask)
	want := []int{-1, -1, 0, 1, 2, 0, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("BarsSince[%d] = %d, want %d", i, got[i], want[i])
		}
	}
	if BarsSince(nil) != nil {
		t.Error("BarsSince(nil) should be nil")
	}
}

func TestValueWhenForwardFills(t *testing.T) {
	cond := []uint8{0, 1, 0, 0, 1, 0}
	xs := []float64{10, 20, 30, 40, 50, 60}
	got := ValueWhen(cond, xs)

	want := []float64{math.NaN(), 20, 20, 20, 50, 50}
	for i := range want {
		if math.IsNaN(want[i]) {
			if !math.IsNaN(got[i]) {
				t.Errorf("ValueWhen[%d] = %v, want NaN", i, got[i])
			}
			continue
		}
		if got[i] != want[i] {
			t.Errorf("ValueWhen[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestHighestLowestSince(t *testing.T) {
	cond := []uint8{1, 0, 0, 1, 0, 0}
	xs := []float64{5, 9, 3, 7, 2, 8}

	hi := HighestSince(cond, xs)
	wantHi := []float64{5, 9, 9, 7, 7, 8}
	for i := range wantHi {
		if hi[i] != wantHi[i] {
			t.Errorf("HighestSince[%d] = %v, want %v", i, hi[i], wantHi[i])
		}
	}

	lo := LowestSince(cond, xs)
	wantLo := []float64{5, 5, 3, 7, 2, 2}
	for i := range wantLo {
		if lo[i] != wantLo[i] {
			t.Errorf("LowestSince[%d] = %v, want %v", i, lo[i], wantLo[i])
		}
	}
}

// TestExtremeSinceNotYetActive checks that an episode which has not begun yields
// NaN rather than zero or an infinity that could be mistaken for a real extreme.
func TestExtremeSinceNotYetActive(t *testing.T) {
	cond := []uint8{0, 0, 1}
	xs := []float64{1, 2, 3}

	hi := HighestSince(cond, xs)
	if !math.IsNaN(hi[0]) || !math.IsNaN(hi[1]) {
		t.Errorf("HighestSince before activation = %v, want NaNs", hi[:2])
	}
	if hi[2] != 3 {
		t.Errorf("HighestSince at activation = %v, want 3", hi[2])
	}
}

func TestSignalPanics(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"ShiftTo length", func() { ShiftTo(short, long, 1) }},
		{"DiffTo length", func() { DiffTo(short, long, 1) }},
		{"RateTo length", func() { RateTo(short, long, 1) }},
		{"RateTo zero lag", func() { RateTo(short, short, 0) }},
		{"Rising zero lag", func() { Rising(short, 0) }},
		{"Falling zero lag", func() { Falling(short, 0) }},
		{"ValueWhen length", func() { ValueWhen(make([]uint8, 2), long) }},
		{"HighestSince length", func() { HighestSince(make([]uint8, 2), long) }},
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

func TestSignalAllocatingFormsReturnNilWhenEmpty(t *testing.T) {
	if Shift(nil, 1) != nil || Diff(nil, 1) != nil || Rate(nil, 1) != nil {
		t.Error("lagged allocating forms should return nil for empty input")
	}
	if Rising(nil, 1) != nil || Falling(nil, 1) != nil {
		t.Error("Rising/Falling should return nil for empty input")
	}
	if CrossOver(nil, nil) != nil || CrossUnder(nil, nil) != nil || Cross(nil, nil) != nil {
		t.Error("cross forms should return nil for empty input")
	}
	if ValueWhen(nil, nil) != nil || HighestSince(nil, nil) != nil || LowestSince(nil, nil) != nil {
		t.Error("run forms should return nil for empty input")
	}
}
