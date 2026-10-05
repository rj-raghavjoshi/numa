package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the catalogue-completion indicators.
//
// Most of these are thin: a ratio, a difference, a standard deviation. The ones with real logic are
// the Relative Vigor Index (a ratio of sums, not a sum of ratios), the Chop Zone (a logarithm whose
// argument can be zero), and the cross indicators (which report an event rather than a level, so the
// test has to be about *where* the signal appears).
// ---------------------------------------------------------------------------

func TestStandardDeviation(t *testing.T) {
	xs := synthClose(150, 5151)
	for _, n := range []int{1, 2, 5, 20} {
		got := StandardDeviation(xs, n)
		if first := FirstValid(got); first != n-1 {
			t.Errorf("n=%d: first valid = %d, want %d", n, first, n-1)
		}
		// Independent reference: the population deviation about the window mean.
		for i := n - 1; i < len(xs); i++ {
			var mean float64
			for j := i - n + 1; j <= i; j++ {
				mean += xs[j]
			}
			mean /= float64(n)
			var ss float64
			for j := i - n + 1; j <= i; j++ {
				d := xs[j] - mean
				ss += d * d
			}
			want := math.Sqrt(ss / float64(n))
			if math.Abs(got[i]-want) > 1e-9 {
				t.Fatalf("n=%d: StandardDeviation[%d] = %v, want %v", n, i, got[i], want)
			}
		}
	}
}

// TestStandardDeviationConstantSeriesIsZero pins the degenerate case.
func TestStandardDeviationConstantSeriesIsZero(t *testing.T) {
	const size = 40
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = 13
	}
	got := StandardDeviation(xs, 10)
	for i := 9; i < size; i++ {
		if math.Abs(got[i]) > 1e-12 {
			t.Fatalf("standard deviation of a constant[%d] = %v, want 0", i, got[i])
		}
	}
}

func TestRatioAndSpread(t *testing.T) {
	a := []float64{10, 20, 30, 40}
	b := []float64{2, 4, 0, 5}

	r := Ratio(a, b)
	want := []float64{5, 5, 0, 8} // the zero denominator yields 0 rather than an infinity
	for i := range want {
		if r[i] != want[i] {
			t.Fatalf("Ratio[%d] = %v, want %v", i, r[i], want[i])
		}
	}

	s := Spread(a, b)
	wantSpread := []float64{8, 16, 30, 35}
	for i := range wantSpread {
		if s[i] != wantSpread[i] {
			t.Fatalf("Spread[%d] = %v, want %v", i, s[i], wantSpread[i])
		}
	}

	if Ratio(nil, nil) != nil || Spread(nil, nil) != nil {
		t.Error("empty inputs should return nil")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("Ratio with mismatched lengths did not panic")
			}
		}()
		Ratio([]float64{1, 2}, []float64{1})
	}()
}

func TestRelativeVigorIndex(t *testing.T) {
	open, high, low, close, _ := synthOHLCV(200, 6161)
	const n, signalPeriod = 10, 4

	got, signal := RelativeVigorIndex(open, high, low, close, n, signalPeriod)

	// Reference: the ratio of the summed vigour to the summed range.
	for i := n - 1; i < len(close); i++ {
		var num, den float64
		for j := i - n + 1; j <= i; j++ {
			r := high[j] - low[j]
			den += r
			if r != 0 {
				num += (close[j] - open[j]) / r
			}
		}
		want := 0.0
		if den != 0 {
			want = num / den
		}
		if math.Abs(got[i]-want) > 1e-9 {
			t.Fatalf("RelativeVigorIndex[%d] = %v, want %v", i, got[i], want)
		}
	}
	if first := FirstValid(signal); first != n-1+signalPeriod-1 {
		t.Errorf("signal first valid = %d, want %d", first, n-1+signalPeriod-1)
	}

	// The vigour is a signed fraction of the range, so the index is bounded by [-1,1].
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v < -1-1e-9 || v > 1+1e-9 {
			t.Fatalf("RelativeVigorIndex[%d] = %v, outside [-1,1]", i, v)
		}
	}
}

// TestRelativeVigorIndexDirection pins the sign convention.
func TestRelativeVigorIndexDirection(t *testing.T) {
	// Same range every bar; bar 1 closes above its open, bar 2 below.
	open := []float64{10, 10, 10}
	high := []float64{11, 11, 11}
	low := []float64{9, 9, 9}
	close := []float64{10, 11, 9}

	rvi, _ := RelativeVigorIndex(open, high, low, close, 1, 1)
	if !(rvi[1] > 0) {
		t.Errorf("a bar closing at its high gave %v, want positive", rvi[1])
	}
	if !(rvi[2] < 0) {
		t.Errorf("a bar closing at its low gave %v, want negative", rvi[2])
	}
}

func TestChopZone(t *testing.T) {
	high, low, close := synthHLC(200, 7171)
	const n = 14

	got := ChopZone(high, low, close, n)
	if first := FirstValid(got); first != n {
		t.Errorf("ChopZone first valid = %d, want %d", first, n)
	}

	// Reference, computed without either rolling sum.
	for i := n; i < len(close); i++ {
		var up, down float64
		for j := i - n + 1; j <= i; j++ {
			up += math.Max(high[j]-close[j-1], 0)
			down += math.Max(close[j-1]-low[j], 0)
		}
		if up == 0 || down == 0 {
			if !math.IsNaN(got[i]) {
				t.Fatalf("index %d should be NaN where a sum is zero", i)
			}
			continue
		}
		want := 100 * math.Log10(up/down) / math.Log10(float64(n))
		if math.Abs(got[i]-want) > 1e-9 {
			t.Fatalf("ChopZone[%d] = %v, want %v", i, got[i], want)
		}
	}
}

// TestChopZoneIsAntisymmetric swaps the high and the low and expects the sign to flip, which checks
// the logarithm's placement rather than its magnitude.
func TestChopZoneIsAntisymmetric(t *testing.T) {
	high, low, close := synthHLC(120, 8181)
	const n = 10

	a := ChopZone(high, low, close, n)
	// Mirror each bar about the *previous* close, which is the price up and down are measured from.
	// Mirroring about the current close would not swap them, because the two sums reference
	// close[i-1] and the mirror would move the reference too.
	mirroredHigh := make([]float64, len(high))
	mirroredLow := make([]float64, len(low))
	for i := 1; i < len(high); i++ {
		mirroredHigh[i] = 2*close[i-1] - low[i]
		mirroredLow[i] = 2*close[i-1] - high[i]
	}
	mirroredHigh[0], mirroredLow[0] = high[0], low[0]
	b := ChopZone(mirroredHigh, mirroredLow, close, n)

	for i := n; i < len(close); i++ {
		if math.Abs(a[i]+b[i]) > 1e-9 {
			t.Fatalf("ChopZone[%d] = %v, mirrored = %v, want negatives of each other", i, a[i], b[i])
		}
	}
}

// TestChopZoneZeroSumIsNaN pins the documented convention: an undefined logarithm is NaN, not a zero.
func TestChopZoneZeroSumIsNaN(t *testing.T) {
	// Every bar sits entirely above the previous close, so the downward reach is zero throughout.
	const size = 40
	high := make([]float64, size)
	low := make([]float64, size)
	close := make([]float64, size)
	for i := range high {
		base := 100 + 10*float64(i)
		low[i] = base
		high[i] = base + 2
		close[i] = base + 1
	}
	got := ChopZone(high, low, close, 5)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		t.Fatalf("ChopZone[%d] = %v, want NaN when the downward reach is zero", i, v)
	}
}

func TestHammingMA(t *testing.T) {
	xs := synthClose(150, 9191)

	for _, n := range []int{1, 2, 5, 20} {
		got := HammingMA(xs, n)
		weights := make([]float64, n)
		var wsum float64
		if n == 1 {
			weights[0] = 1
			wsum = 1
		} else {
			for k := 0; k < n; k++ {
				weights[k] = 0.54 - 0.46*math.Cos(2*math.Pi*float64(k)/float64(n-1))
				wsum += weights[k]
			}
		}
		for i := n - 1; i < len(xs); i++ {
			var acc float64
			for k := 0; k < n; k++ {
				acc += weights[k] * xs[i-n+1+k]
			}
			want := acc / wsum
			if math.Abs(got[i]-want) > 1e-9 {
				t.Fatalf("n=%d: HammingMA[%d] = %v, want %v", n, i, got[i], want)
			}
		}
	}
}

// TestHammingMAWeightsArePositiveAndSumToOne checks the profile, which is what distinguishes it from
// a WMA: the weights are not monotone, so the average is not simply lagged.
func TestHammingMAWeightsArePositiveAndSumToOne(t *testing.T) {
	const size = 60
	const c = 21.0
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = c
	}
	got := HammingMA(flat, 9)
	for i := 8; i < size; i++ {
		if math.Abs(got[i]-c) > 1e-9 {
			t.Fatalf("HammingMA of a constant[%d] = %v, want %v (weights must sum to 1)", i, got[i], c)
		}
	}
	// n = 1 is the input itself.
	i1 := HammingMA(synthClose(20, 1), 1)
	if i1[5] != synthClose(20, 1)[5] {
		t.Error("HammingMA with a window of one should be the input")
	}
}

func TestSMIErgodic(t *testing.T) {
	close := synthClose(300, 1010)
	const long, short, signalPeriod = 25, 13, 7

	ergodic, signal, osc := SMIErgodic(close, long, short, signalPeriod)

	assertSeries(t, "ergodic", ergodic, TrueStrengthIndex(close, long, short), false)
	wantSignal := applyEMA(ergodic, FirstValid(ergodic), signalPeriod)
	assertSeries(t, "signal", signal, wantSignal, false)

	for i := range close {
		if math.IsNaN(ergodic[i]) || math.IsNaN(signal[i]) {
			if !math.IsNaN(osc[i]) {
				t.Fatalf("oscillator[%d] = %v is defined where its inputs are not", i, osc[i])
			}
			continue
		}
		if !closeOrNaN(osc[i], ergodic[i]-signal[i]) {
			t.Fatalf("oscillator[%d] = %v, want ergodic-signal %v", i, osc[i], ergodic[i]-signal[i])
		}
	}
}

func TestAdvanceDecline(t *testing.T) {
	adv := []float64{10, 20, 5}
	dec := []float64{5, 25, 5}
	got := AdvanceDecline(adv, dec)
	want := []float64{5, 0, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("AdvanceDecline[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if AdvanceDecline(nil, nil) != nil {
		t.Error("empty AdvanceDecline should be nil")
	}
}

func TestMACrossFamily(t *testing.T) {
	close := synthClose(200, 1111)
	const fast, slow = 10, 30

	for name, tc := range map[string]struct {
		fastExp, slowExp bool
	}{
		"MACross":     {false, false},
		"EMACross":    {true, true},
		"EMASMACross": {true, false},
	} {
		gotFast, gotSlow, cross := movingAverageCross(close, fast, slow, tc.fastExp, tc.slowExp)
		// fastExp true means the exponential average, matching the helper's parameter.
		if tc.fastExp {
			assertSeries(t, name+" fast", gotFast, EMA(close, fast), false)
		} else {
			assertSeries(t, name+" fast", gotFast, SMA(close, fast), false)
		}
		if tc.slowExp {
			assertSeries(t, name+" slow", gotSlow, EMA(close, slow), false)
		} else {
			assertSeries(t, name+" slow", gotSlow, SMA(close, slow), false)
		}
		// Every signal must be a real crossing: the sign of the difference has to change.
		for i := 1; i < len(close); i++ {
			if cross[i] == 0 {
				continue
			}
			prev := gotFast[i-1] - gotSlow[i-1]
			now := gotFast[i] - gotSlow[i]
			if math.IsNaN(prev) || math.IsNaN(now) {
				t.Fatalf("%s: a signal at %d where the lines are undefined", name, i)
			}
			if cross[i] == 1 && !(prev <= 0 && now > 0) {
				t.Fatalf("%s: up-cross at %d but the difference went %v -> %v", name, i, prev, now)
			}
			if cross[i] == -1 && !(prev >= 0 && now < 0) {
				t.Fatalf("%s: down-cross at %d but the difference went %v -> %v", name, i, prev, now)
			}
		}
	}
}

// TestCrossIndicatorsReportEventsNotLevels is the property that distinguishes them: the signal must
// be sparse. A caller acting on the sign of the difference would trade every bar; a crossing happens
// a handful of times.
func TestCrossIndicatorsReportEventsNotLevels(t *testing.T) {
	close := synthClose(400, 1212)
	_, _, cross := MACross(close, 10, 30)

	events := 0
	for _, c := range cross {
		if c != 0 {
			events++
		}
	}
	if events == 0 {
		t.Fatal("no crossings over 400 bars; the test data cannot distinguish an event from a level")
	}
	if events > len(close)/4 {
		t.Fatalf("%d crossings over %d bars: the signal is a sign, not an event", events, len(close))
	}
}

func TestCatalogueEmptyAndPanics(t *testing.T) {
	open, high, low, close, _ := synthOHLCV(40, 1313)

	if StandardDeviation(nil, 5) != nil {
		t.Error("empty StandardDeviation should be nil")
	}
	if ChopZone(nil, nil, nil, 5) != nil {
		t.Error("empty ChopZone should be nil")
	}
	if HammingMA(nil, 5) != nil {
		t.Error("empty HammingMA should be nil")
	}
	if e, s, o := SMIErgodic(nil, 25, 13, 7); e != nil || s != nil || o != nil {
		t.Error("empty SMIErgodic should return nils")
	}
	if r, s := RelativeVigorIndex(nil, nil, nil, nil, 10, 4); r != nil || s != nil {
		t.Error("empty RelativeVigorIndex should return nils")
	}
	if f, s, c := MACross(nil, 10, 30); f != nil || s != nil || c != nil {
		t.Error("empty MACross should return nils")
	}

	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"StandardDeviation period", func() { StandardDeviation(close, 0) }},
		{"Ratio length", func() { Ratio(short, long) }},
		{"Spread length", func() { Spread(short, long) }},
		{"RVI period", func() { RelativeVigorIndex(open, high, low, close, 0, 4) }},
		{"RVI length", func() { RelativeVigorIndex(short, high, low, close, 10, 4) }},
		{"ChopZone period", func() { ChopZone(high, low, close, 0) }},
		{"ChopZone length", func() { ChopZone(short, low, close, 10) }},
		{"HammingMA period", func() { HammingMA(close, 0) }},
		{"SMIErgodic signalPeriod", func() { SMIErgodic(close, 25, 13, 0) }},
		{"AdvanceDecline length", func() { AdvanceDecline(short, long) }},
		{"MACross reversed", func() { MACross(close, 30, 10) }},
		{"MACross equal", func() { MACross(close, 10, 10) }},
		{"EMACross period", func() { EMACross(close, 0, 30) }},
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
