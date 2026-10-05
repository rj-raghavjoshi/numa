package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the momentum indicators.
//
// Two kinds of check are used together, deliberately:
//
//   - agreement with an independently written reference over many periods and
//     lengths, which catches a structural mistake
//   - hand-computed values at specific bars, which catch a reference that shares the
//     implementation's misunderstanding
//
// The division-by-zero cases get their own tests, because a "neutral" value that
// disagrees with the reference implementation is a systematic error, not a rounding
// one.
// ---------------------------------------------------------------------------

func refRSI(close []float64, n int) []float64 {
	out := allNaN(len(close))
	if len(close) <= n {
		return out
	}
	alpha := 1 / float64(n)
	var gs, ls float64
	for i := 1; i <= n; i++ {
		c := close[i] - close[i-1]
		if c > 0 {
			gs += c
		} else {
			ls -= c
		}
	}
	ag, al := gs/float64(n), ls/float64(n)
	out[n] = refRSIValue(ag, al)
	for i := n + 1; i < len(close); i++ {
		c := close[i] - close[i-1]
		var g, l float64
		if c > 0 {
			g = c
		} else {
			l = -c
		}
		ag = alpha*g + (1-alpha)*ag
		al = alpha*l + (1-alpha)*al
		out[i] = refRSIValue(ag, al)
	}
	return out
}

func refRSIValue(g, l float64) float64 {
	if g == 0 && l == 0 {
		return 50
	}
	if l == 0 {
		return 100
	}
	if g == 0 {
		return 0
	}
	return 100 - 100/(1+g/l)
}

func refMACD(close []float64, fast, slow, signal int) (macd, sig, hist []float64) {
	fe := refSmoother(close, fast, 2.0/float64(fast+1))
	se := refSmoother(close, slow, 2.0/float64(slow+1))
	macd = make([]float64, len(close))
	for i := range close {
		macd[i] = fe[i] - se[i]
	}
	// Signal: SMA-seeded EMA over the MACD line, started at its first valid index.
	sig = allNaN(len(close))
	start := -1
	for i, v := range macd {
		if v == v {
			start = i
			break
		}
	}
	if start >= 0 {
		var seed float64
		count := 0
		var prev float64
		seeded := false
		alpha := 2.0 / float64(signal+1)
		for i := start; i < len(close); i++ {
			if !seeded {
				seed += macd[i]
				count++
				if count < signal {
					continue
				}
				prev = seed / float64(signal)
				seeded = true
				sig[i] = prev
				continue
			}
			prev = alpha*macd[i] + (1-alpha)*prev
			sig[i] = prev
		}
	}
	hist = make([]float64, len(close))
	for i := range close {
		hist[i] = macd[i] - sig[i]
	}
	return macd, sig, hist
}

func refRawStoch(high, low, close []float64, n int) []float64 {
	hi := refExtreme(high, n, true)
	lo := refExtreme(low, n, false)
	out := make([]float64, len(close))
	for i := range close {
		span := hi[i] - lo[i]
		switch {
		case hi[i] != hi[i] || lo[i] != lo[i]:
			out[i] = math.NaN()
		case span == 0:
			out[i] = 50
		default:
			out[i] = 100 * (close[i] - lo[i]) / span
		}
	}
	return out
}

func refCCI(high, low, close []float64, n int) []float64 {
	tp := make([]float64, len(close))
	for i := range close {
		tp[i] = (high[i] + low[i] + close[i]) / 3
	}
	out := allNaN(len(close))
	for i := n - 1; i < len(close); i++ {
		var mean float64
		for j := i - n + 1; j <= i; j++ {
			mean += tp[j]
		}
		mean /= float64(n)
		var dev float64
		for j := i - n + 1; j <= i; j++ {
			dev += math.Abs(tp[j] - mean)
		}
		dev /= float64(n)
		if dev == 0 {
			out[i] = 0
			continue
		}
		out[i] = (tp[i] - mean) / (0.015 * dev)
	}
	return out
}

func refCMO(close []float64, n int) []float64 {
	out := allNaN(len(close))
	for i := n; i < len(close); i++ {
		var gs, ls float64
		for j := i - n + 1; j <= i; j++ {
			c := close[j] - close[j-1]
			if c > 0 {
				gs += c
			} else {
				ls -= c
			}
		}
		if gs == 0 && ls == 0 {
			out[i] = 0
			continue
		}
		out[i] = 100 * (gs - ls) / (gs + ls)
	}
	return out
}

// ---------------------------------------------------------------------------

func TestRSIMatchesReference(t *testing.T) {
	for _, n := range boundaryLengths() {
		if n == 0 {
			continue
		}
		close := synthClose(n, int64(n)+101)
		for _, p := range []int{2, 3, 5, 7, 14, 21} {
			assertSeries(t, "RSI", RSI(close, p), refRSI(close, p), false)
		}
	}
}

// TestRSIKnownValues walks a small alternating series by hand.
//
// close = {1,2,1,2,1}, n=2. Changes are +1,-1,+1,-1.
//
//	seed at index 2: avgGain=0.5, avgLoss=0.5 -> RS=1 -> 50
//	index 3: +1 -> avgGain=0.75, avgLoss=0.25 -> RS=3 -> 75
//	index 4: -1 -> avgGain=0.375, avgLoss=0.625 -> RS=0.6 -> 37.5
func TestRSIKnownValues(t *testing.T) {
	got := RSI([]float64{1, 2, 1, 2, 1}, 2)
	want := []float64{math.NaN(), math.NaN(), 50, 75, 37.5}
	assertSeries(t, "RSI", got, want, false)
}

// TestRSIDegenerateSeries pins the three neutral cases, which is where a naive
// division would produce NaN or an infinity.
func TestRSIDegenerateSeries(t *testing.T) {
	up := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	down := []float64{8, 7, 6, 5, 4, 3, 2, 1}
	flat := []float64{5, 5, 5, 5, 5, 5, 5, 5}

	if got := RSI(up, 3); got[len(up)-1] != 100 {
		t.Errorf("RSI of a rising series = %v, want 100", got[len(up)-1])
	}
	if got := RSI(down, 3); got[len(down)-1] != 0 {
		t.Errorf("RSI of a falling series = %v, want 0", got[len(down)-1])
	}
	if got := RSI(flat, 3); got[len(flat)-1] != 50 {
		t.Errorf("RSI of a flat series = %v, want 50", got[len(flat)-1])
	}
}

// TestRSIWarmupIsNNotNMinusOne pins the boundary: RSI consumes n changes, and the
// first change is at index 1, so the first output is at index n.
func TestRSIWarmupIsNNotNMinusOne(t *testing.T) {
	close := synthClose(40, 7)
	for _, n := range []int{2, 5, 14} {
		if got := FirstValid(RSI(close, n)); got != n {
			t.Errorf("RSI(%d) first valid = %d, want %d", n, got, n)
		}
	}
}

func TestMACDMatchesReference(t *testing.T) {
	for _, n := range []int{20, 40, 80, 200} {
		close := synthClose(n, int64(n)+202)
		for _, tc := range [][3]int{{12, 26, 9}, {5, 35, 5}, {3, 10, 16}} {
			gm, gs, gh := MACD(close, tc[0], tc[1], tc[2])
			wm, ws, wh := refMACD(close, tc[0], tc[1], tc[2])
			assertSeries(t, "MACD line", gm, wm, false)
			assertSeries(t, "MACD signal", gs, ws, false)
			assertSeries(t, "MACD hist", gh, wh, false)
		}
	}
}

func TestMACDHistogramIsLineMinusSignal(t *testing.T) {
	close := synthClose(120, 9)
	macd, sig, hist := MACD(close, 12, 26, 9)
	for i := range close {
		if !closeOrNaN(hist[i], macd[i]-sig[i]) {
			t.Fatalf("hist[%d] = %v, want line-signal = %v", i, hist[i], macd[i]-sig[i])
		}
	}
	// The signal line must be NaN strictly before the MACD line is defined plus the
	// signal warm-up, and defined after.
	if !math.IsNaN(sig[FirstValid(macd)]) {
		t.Error("the signal line is defined at the MACD's first valid index, which is too early")
	}
	for i := FirstValid(sig); i < len(close); i++ {
		if math.IsNaN(sig[i]) {
			t.Fatalf("signal[%d] = NaN after its first valid index", i)
		}
	}
}

func TestStochasticMatchesReference(t *testing.T) {
	for _, n := range []int{30, 60, 120} {
		high, low, close := synthHLC(n, int64(n)+303)
		for _, tc := range [][3]int{{14, 3, 3}, {5, 1, 1}, {20, 5, 4}} {
			gk, gd := Stochastic(high, low, close, tc[0], tc[1], tc[2])
			raw := refRawStoch(high, low, close, tc[0])
			wantK := refSMA(raw, tc[1])
			wantD := refSMA(wantK, tc[2])
			assertSeries(t, "Stochastic K", gk, wantK, false)
			assertSeries(t, "Stochastic D", gd, wantD, false)
		}
	}
}

// TestStochasticFlatWindowIsFifty pins the neutral value for a window with no range.
func TestStochasticFlatWindowIsFifty(t *testing.T) {
	flat := make([]float64, 10)
	for i := range flat {
		flat[i] = 5
	}
	k, d := Stochastic(flat, flat, flat, 3, 1, 1)
	for i := 2; i < len(flat); i++ {
		if k[i] != 50 {
			t.Fatalf("Stochastic K on a flat window = %v, want 50", k[i])
		}
		if d[i] != 50 {
			t.Fatalf("Stochastic D on a flat window = %v, want 50", d[i])
		}
	}
}

// TestWilliamsPercentRIsStochMinusHundred pins the exact algebraic relationship
// between the two indicators, which any sign or offset error would break.
func TestWilliamsPercentRIsStochMinusHundred(t *testing.T) {
	high, low, close := synthHLC(80, 55)
	const n = 14

	raw := refRawStoch(high, low, close, n)
	got := WilliamsPercentR(high, low, close, n)
	for i := range close {
		if !closeOrNaN(got[i], raw[i]-100) {
			t.Fatalf("%%R[%d] = %v, want rawK-100 = %v", i, got[i], raw[i]-100)
		}
	}
}

// TestWilliamsPercentRFlatWindowIsZero pins the documented neutral value for a window
// whose range is zero.
func TestWilliamsPercentRFlatWindowIsZero(t *testing.T) {
	flat := make([]float64, 6)
	for i := range flat {
		flat[i] = 3
	}
	got := WilliamsPercentR(flat, flat, flat, 3)
	if got[5] != 0 {
		t.Errorf("%%R on a flat window = %v, want 0", got[5])
	}
}

func TestStochRSIMatchesComposition(t *testing.T) {
	close := synthClose(200, 77)
	const rsiP, stochP, kS, dS = 14, 14, 3, 3

	gk, gd := StochRSI(close, rsiP, stochP, kS, dS)

	// Rebuild from the exported pieces.
	r := RSI(close, rsiP)
	raw := refRawStoch(r, r, r, stochP)
	wantK := refSMA(raw, kS)
	wantD := refSMA(wantK, dS)

	assertSeries(t, "StochRSI K", gk, wantK, false)
	assertSeries(t, "StochRSI D", gd, wantD, false)
}

func TestCCIMatchesReference(t *testing.T) {
	for _, n := range []int{40, 80, 160} {
		high, low, close := synthHLC(n, int64(n)+404)
		for _, p := range []int{5, 14, 20} {
			assertSeries(t, "CCI", CCI(high, low, close, p), refCCI(high, low, close, p), false)
		}
	}
}

// TestCCIFlatWindowIsZero pins the zero-mean-deviation case.
func TestCCIFlatWindowIsZero(t *testing.T) {
	flat := make([]float64, 8)
	for i := range flat {
		flat[i] = 2
	}
	got := CCI(flat, flat, flat, 5)
	if got[7] != 0 {
		t.Errorf("CCI on a flat window = %v, want 0", got[7])
	}
}

func TestCMOMatchesReference(t *testing.T) {
	for _, n := range []int{30, 90} {
		close := synthClose(n, int64(n)+505)
		for _, p := range []int{2, 5, 9, 14} {
			assertSeries(t, "CMO", CMO(close, p), refCMO(close, p), false)
		}
	}
}

// TestCMOKnownValues: for {1,2,1,2,1} with n=2 the windows of changes are
// {+1,-1} -> 0, {+1,-1} -> 0, {-1,+1} -> 0, so a symmetrical alternating series has a
// CMO of exactly 0 at every valid index, unlike RSI which is smoothed.
func TestCMOKnownValues(t *testing.T) {
	got := CMO([]float64{1, 2, 1, 2, 1}, 2)
	for i := 2; i < len(got); i++ {
		if got[i] != 0 {
			t.Errorf("CMO[%d] = %v, want 0 (symmetric gains and losses)", i, got[i])
		}
	}

	// A strictly rising series has no losses at all.
	up := CMO([]float64{1, 2, 3, 4, 5}, 2)
	for i := 2; i < len(up); i++ {
		if up[i] != 100 {
			t.Errorf("CMO of a rising series[%d] = %v, want 100", i, up[i])
		}
	}
}

func TestCMOFlatSeriesIsZero(t *testing.T) {
	flat := []float64{4, 4, 4, 4, 4, 4}
	got := CMO(flat, 2)
	for i := 2; i < len(got); i++ {
		if got[i] != 0 {
			t.Errorf("CMO on a flat series[%d] = %v, want 0", i, got[i])
		}
	}
}

func TestROCMatchesDefinition(t *testing.T) {
	close := synthClose(50, 606)
	for _, n := range []int{1, 3, 9} {
		got := ROC(close, n)
		for i := 0; i < n; i++ {
			if !math.IsNaN(got[i]) {
				t.Errorf("ROC(%d)[%d] = %v, want NaN", n, i, got[i])
			}
		}
		for i := n; i < len(close); i++ {
			want := 100 * (close[i] - close[i-n]) / close[i-n]
			if !closeOrNaN(got[i], want) {
				t.Fatalf("ROC(%d)[%d] = %v, want %v", n, i, got[i], want)
			}
		}
	}

	// Known values.
	got := ROC([]float64{10, 11, 12, 13}, 1)
	if math.Abs(got[1]-10) > 1e-12 || math.Abs(got[2]-100.0/11) > 1e-12 {
		t.Errorf("ROC = %v, want [NaN 10 9.09... ...]", got)
	}
}

// TestROCZeroBaseIsZero pins the SafeDiv convention.
func TestROCZeroBaseIsZero(t *testing.T) {
	got := ROC([]float64{0, 5, 5}, 1)
	if got[1] != 0 {
		t.Errorf("ROC with a zero base = %v, want 0", got[1])
	}
}

func TestMomentumIsDiff(t *testing.T) {
	close := synthClose(30, 707)
	for _, n := range []int{1, 5, 9} {
		assertSeries(t, "Momentum", Momentum(close, n), Change(close, n), true)
	}
}

func TestMomentumIndicatorsEmptyAndPanics(t *testing.T) {
	if RSI(nil, 14) != nil || CMO(nil, 14) != nil || ROC(nil, 14) != nil || Momentum(nil, 14) != nil {
		t.Error("empty series should return nil")
	}
	if k, d := Stochastic(nil, nil, nil, 14, 3, 3); k != nil || d != nil {
		t.Error("empty Stochastic should return nils")
	}
	if k, d := StochRSI(nil, 14, 14, 3, 3); k != nil || d != nil {
		t.Error("empty StochRSI should return nils")
	}
	if m, s, h := MACD(nil, 12, 26, 9); m != nil || s != nil || h != nil {
		t.Error("empty MACD should return nils")
	}
	if CCI(nil, nil, nil, 14) != nil || WilliamsPercentR(nil, nil, nil, 14) != nil {
		t.Error("empty CCI/%%R should return nil")
	}

	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"RSI period", func() { RSI(ramp(10), 0) }},
		{"MACD fast", func() { MACD(ramp(10), 0, 26, 9) }},
		{"Stochastic kPeriod", func() { Stochastic(ramp(10), ramp(10), ramp(10), 0, 3, 3) }},
		{"Stochastic length", func() { Stochastic(ramp(10), ramp(10), ramp(9), 14, 3, 3) }},
		{"StochRSI period", func() { StochRSI(ramp(10), 0, 14, 3, 3) }},
		{"WilliamsPercentR length", func() { WilliamsPercentR(ramp(10), ramp(9), ramp(10), 5) }},
		{"CCI period", func() { CCI(ramp(10), ramp(10), ramp(10), -1) }},
		{"CMO period", func() { CMO(ramp(10), 0) }},
		{"ROC period", func() { ROC(ramp(10), 0) }},
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

// TestMomentumIndicatorsShorterThanPeriod checks the degenerate case where the series
// is shorter than the period: every result must be NaN rather than a partial window.
func TestMomentumIndicatorsShorterThanPeriod(t *testing.T) {
	close := []float64{1, 2, 3}

	for _, v := range RSI(close, 10) {
		if !math.IsNaN(v) {
			t.Error("RSI shorter than period should be all NaN")
		}
	}
	for _, v := range CMO(close, 10) {
		if !math.IsNaN(v) {
			t.Error("CMO shorter than period should be all NaN")
		}
	}
}

// TestRSIComposesFromALeadingNaN is the regression test for a permanent failure rather than a
// warm-up error.
//
// RSI used to seed from the changes at indices 1..n unconditionally. Given an input whose first
// element is NaN -- which is what any composed series looks like -- the seed consumed that NaN and
// every output was NaN, forever, because a seeded recurrence cannot forget. The streak counter
// behind ConnorsRSI is exactly such an input, so this bug made ConnorsRSI all-NaN without anything
// in RSI's own tests noticing: they all fed it a price series, whose first element is valid.
func TestRSIComposesFromALeadingNaN(t *testing.T) {
	const n = 5
	xs := synthClose(120, 4242)

	// A leading NaN with otherwise valid data.
	leading := append([]float64{math.NaN()}, xs...)
	got := RSI(leading, n)

	// The first output must exist, not be NaN for the whole series.
	first := FirstValid(got)
	if first < 0 {
		t.Fatal("RSI of a series with one leading NaN is entirely NaN")
	}
	// The change at index 1 is undefined, so the first real change is at index 2 and the first
	// output at 2+n-1.
	if first != n+1 {
		t.Errorf("first valid = %d, want %d", first, n+1)
	}
	if math.IsNaN(got[len(got)-1]) {
		t.Error("the last RSI value is NaN")
	}

	// And the recovered values must equal the RSI of the same data without the leading NaN.
	plain := RSI(xs, n)
	for i := n + 1; i < len(got); i++ {
		if !closeOrNaN(got[i], plain[i-1]) {
			t.Fatalf("composed RSI[%d] = %v, want %v", i, got[i], plain[i-1])
		}
	}
}

// TestRSIRejectsAnAllNaNInput pins the degenerate case: no change is ever defined.
func TestRSIRejectsAnAllNaNInput(t *testing.T) {
	xs := make([]float64, 30)
	for i := range xs {
		xs[i] = math.NaN()
	}
	if got := FirstValid(RSI(xs, 5)); got != -1 {
		t.Errorf("RSI of an all-NaN series has a valid value at %d", got)
	}
}
