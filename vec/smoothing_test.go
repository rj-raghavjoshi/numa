package vec

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the rolling exponential smoothers.
//
// The seed decides every later value, so the tests pin it directly rather than only comparing against
// a reference: a reference written from the same product description would share any misreading of
// "seeded with the SMA of the first n values", and an implementation that seeded with the first
// observation would agree with a careless reference while disagreeing with the streaming roller.
//
// The equivalence with `series.EMA`/`RMA` is checked in the root package, which is the only place that
// can import both.
// ---------------------------------------------------------------------------

// refSmoother is an independent implementation of the recurrence, taking the seed as a parameter so
// the seed convention itself can be varied by a test.
func refSmoother(xs []float64, n int, alpha float64, seed func([]float64, int) float64) []float64 {
	out := make([]float64, len(xs))
	fillNaN(out, 0, min(n-1, len(xs)))
	if len(xs) < n {
		return out
	}
	prev := seed(xs, n)
	out[n-1] = prev
	for i := n; i < len(xs); i++ {
		prev = alpha*xs[i] + (1-alpha)*prev
		out[i] = prev
	}
	return out
}

func smaSeed(xs []float64, n int) float64 {
	var s float64
	for i := 0; i < n; i++ {
		s += xs[i]
	}
	return s / float64(n)
}

func TestRollingEMAAndRMAMatchReference(t *testing.T) {
	for _, size := range rollingLengths() {
		if size == 0 {
			continue
		}
		xs := randomSlice(size, int64(size)+31)
		for _, n := range rollingWindows() {
			wantEMA := refSmoother(xs, n, 2.0/float64(n+1), smaSeed)
			assertRolling(t, "RollingEMA", RollingEMA(xs, n), wantEMA, false)

			wantRMA := refSmoother(xs, n, 1.0/float64(n), smaSeed)
			assertRolling(t, "RollingRMA", RollingRMA(xs, n), wantRMA, false)
		}
	}
}

// TestRollingSmootherSeedIsTheSMA is the decisive check on the convention.
func TestRollingSmootherSeedIsTheSMA(t *testing.T) {
	xs := []float64{10, 20, 30, 40, 50}
	const n = 3

	ema := RollingEMA(xs, n)
	if !closeEnough(ema[n-1], 20) {
		t.Fatalf("EMA[%d] = %v, want the SMA of the first three values, 20", n-1, ema[n-1])
	}
	if closeEnough(ema[n-1], xs[0]) {
		t.Error("the seed is the first observation rather than the SMA")
	}
	if !math.IsNaN(ema[0]) || !math.IsNaN(ema[1]) {
		t.Errorf("the warm-up must be NaN, got %v %v", ema[0], ema[1])
	}

	rma := RollingRMA(xs, n)
	if !closeEnough(rma[n-1], 20) {
		t.Fatalf("RMA[%d] = %v, want 20", n-1, rma[n-1])
	}
}

// TestRollingSmootherWindowOfOneIsTheInput pins the degenerate window: alpha is 1, so the recurrence
// reduces to the identity.
func TestRollingSmootherWindowOfOneIsTheInput(t *testing.T) {
	xs := randomSlice(50, 32)
	assertRolling(t, "RollingEMA(1)", RollingEMA(xs, 1), xs, true)
	assertRolling(t, "RollingRMA(1)", RollingRMA(xs, 1), xs, true)
}

// TestRollingSmootherNaNInSeedIsPermanent pins the documented asymmetry: a NaN that arrives before
// the window is full poisons the seed, and a seeded recurrence cannot forget it.
func TestRollingSmootherNaNInSeedIsPermanent(t *testing.T) {
	const n = 5
	xs := randomSlice(40, 33)
	xs[2] = math.NaN()

	for name, got := range map[string][]float64{
		"RollingEMA": RollingEMA(xs, n),
		"RollingRMA": RollingRMA(xs, n),
	} {
		for i := n - 1; i < len(xs); i++ {
			if !math.IsNaN(got[i]) {
				t.Fatalf("%s[%d] = %v: a NaN inside the seed must poison every later output", name, i, got[i])
			}
		}
	}
}

// TestRollingSmootherNaNAfterTheSeedPropagates is the other half: a NaN after seeding enters the
// recurrence and carries forward.
func TestRollingSmootherNaNAfterTheSeedPropagates(t *testing.T) {
	const n = 3
	xs := randomSlice(30, 34)
	xs[10] = math.NaN()

	for name, got := range map[string][]float64{
		"RollingEMA": RollingEMA(xs, n),
		"RollingRMA": RollingRMA(xs, n),
	} {
		if math.IsNaN(got[9]) {
			t.Errorf("%s[9] = NaN, want a value before the NaN arrives at index 10", name)
		}
		for i := 10; i < len(xs); i++ {
			if !math.IsNaN(got[i]) {
				t.Fatalf("%s[%d] = %v, want NaN after the recurrence absorbed one", name, i, got[i])
			}
		}
	}
}

func TestRollingSmootherToVariants(t *testing.T) {
	xs := randomSlice(80, 35)
	// Separate destinations: the two calls must not share a buffer, or the second overwrites the
	// first and the comparison below silently checks one against the other.
	dstEMA := make([]float64, len(xs))
	dstRMA := make([]float64, len(xs))

	if got := RollingEMATo(dstEMA, xs, 10); &got[0] != &dstEMA[0] {
		t.Error("RollingEMATo did not return dst")
	}
	if got := RollingRMATo(dstRMA, xs, 10); &got[0] != &dstRMA[0] {
		t.Error("RollingRMATo did not return dst")
	}
	assertRolling(t, "RollingEMATo", dstEMA, RollingEMA(xs, 10), true)
	assertRolling(t, "RollingRMATo", dstRMA, RollingRMA(xs, 10), true)
}

func TestRollingSmootherEmptyAndPanics(t *testing.T) {
	if RollingEMA(nil, 5) != nil || RollingRMA(nil, 5) != nil {
		t.Error("empty inputs should return nil")
	}

	xs := randomSlice(20, 36)
	short := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"EMA period", func() { RollingEMA(xs, 0) }},
		{"EMA negative period", func() { RollingEMA(xs, -1) }},
		{"RMA period", func() { RollingRMA(xs, 0) }},
		{"EMATo length", func() { RollingEMATo(short, xs, 5) }},
		{"RMATo length", func() { RollingRMATo(short, xs, 5) }},
	} {
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

func TestRollingSmootherWindowLargerThanInput(t *testing.T) {
	xs := []float64{1, 2, 3}
	for name, got := range map[string][]float64{
		"RollingEMA": RollingEMA(xs, 10),
		"RollingRMA": RollingRMA(xs, 10),
	} {
		for i, v := range got {
			if !math.IsNaN(v) {
				t.Fatalf("%s[%d] = %v, want NaN with a window larger than the input", name, i, v)
			}
		}
	}
}

// TestRollingSmootherSeedIsCompensated checks the property the compensation exists for: a seed over a
// series with a large offset must not lose its low-order bits, because the seed error would then decay
// only exponentially rather than disappearing.
func TestRollingSmootherSeedIsCompensated(t *testing.T) {
	const n = 4
	// 1e16 followed by three ones: the plain sum loses all three, the compensated one does not.
	xs := []float64{1e16, 1, 1, 1, 1e16, 1, 1, 1}
	got := RollingEMA(xs, n)
	// The seed over the first four is 1e16+3, which as a float64 is 1e16 -- so the seed *value* is the
	// same either way. What differs is that the compensation keeps the 3 available for later, which a
	// test cannot see through a float64 output. Assert instead that the result is finite and equals the
	// value the compensated arithmetic produces, computed here directly.
	var total, compensation float64
	for i := 0; i < n; i++ {
		d := total + xs[i]
		if math.Abs(total) >= math.Abs(xs[i]) {
			compensation += (total - d) + xs[i]
		} else {
			compensation += (xs[i] - d) + total
		}
		total = d
	}
	seed := (total + compensation) / float64(n)
	if got[n-1] != seed {
		t.Fatalf("seed = %v, want the compensated value %v", got[n-1], seed)
	}
}
