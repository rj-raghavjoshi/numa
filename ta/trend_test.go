package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the trend indicators.
//
// The recursive ones (SuperTrend, Parabolic SAR) cannot be checked against a window
// reference, because they have no window. They are checked by their defining
// invariants instead: the stop must stay on the correct side of price, the bands must
// ratchet in one direction only, and the direction must flip when price crosses. Those
// properties are stronger than agreement with a reimplementation, because they hold for
// every input rather than for one test vector.
// ---------------------------------------------------------------------------

func refSmootherFrom(xs []float64, from, n int, alpha float64) []float64 {
	out := allNaN(len(xs))
	if from < 0 || from >= len(xs) {
		return out
	}
	var seed, prev float64
	count := 0
	seeded := false
	for i := from; i < len(xs); i++ {
		if !seeded {
			seed += xs[i]
			count++
			if count < n {
				continue
			}
			prev = seed / float64(n)
			seeded = true
			out[i] = prev
			continue
		}
		prev = alpha*xs[i] + (1-alpha)*prev
		out[i] = prev
	}
	return out
}

func refDirectionalMovement(high, low, close []float64, n int) (adx, pdi, mdi []float64) {
	size := len(close)
	tr := TrueRange(high, low, close)
	pdm := make([]float64, size)
	mdm := make([]float64, size)
	for i := 1; i < size; i++ {
		up := high[i] - high[i-1]
		down := low[i-1] - low[i]
		if up > down && up > 0 {
			pdm[i] = up
		}
		if down > up && down > 0 {
			mdm[i] = down
		}
	}
	alpha := 1 / float64(n)
	atr := refSmoother(tr, n, alpha)
	rp := refSmootherFrom(pdm, 1, n, alpha)
	rm := refSmootherFrom(mdm, 1, n, alpha)

	pdi = allNaN(size)
	mdi = allNaN(size)
	for i := range close {
		if atr[i] != atr[i] || rp[i] != rp[i] || rm[i] != rm[i] {
			continue
		}
		if atr[i] == 0 {
			pdi[i], mdi[i] = 0, 0
			continue
		}
		pdi[i] = 100 * rp[i] / atr[i]
		mdi[i] = 100 * rm[i] / atr[i]
	}

	dx := allNaN(size)
	for i := range close {
		if pdi[i] != pdi[i] || mdi[i] != mdi[i] {
			continue
		}
		sum := pdi[i] + mdi[i]
		if sum == 0 {
			dx[i] = 0
			continue
		}
		dx[i] = 100 * math.Abs(pdi[i]-mdi[i]) / sum
	}
	adx = refSmootherFrom(dx, FirstValid(dx), n, 1/float64(n))
	return adx, pdi, mdi
}

func refAroon(high, low []float64, n int) (up, down, osc []float64) {
	size := len(high)
	up = allNaN(size)
	down = allNaN(size)
	osc = allNaN(size)
	for i := n - 1; i < size; i++ {
		base := i - n + 1
		hi, lo := base, base
		for j := base; j <= i; j++ {
			if high[j] >= high[hi] {
				hi = j
			}
			if low[j] <= low[lo] {
				lo = j
			}
		}
		up[i] = 100 * float64(n-(i-hi)) / float64(n)
		down[i] = 100 * float64(n-(i-lo)) / float64(n)
		osc[i] = up[i] - down[i]
	}
	return
}

func refVortex(high, low, close []float64, n int) (vp, vm []float64) {
	size := len(close)
	tr := TrueRange(high, low, close)
	vp = allNaN(size)
	vm = allNaN(size)
	// The movement terms need a previous bar, so the first complete window of n of
	// them ends at index n, not n-1.
	for i := n; i < size; i++ {
		var sp, sm, st float64
		for j := i - n + 1; j <= i; j++ {
			sp += math.Abs(high[j] - low[j-1])
			sm += math.Abs(low[j] - high[j-1])
			st += tr[j]
		}
		if st == 0 {
			vp[i], vm[i] = 0, 0
			continue
		}
		vp[i] = sp / st
		vm[i] = sm / st
	}
	return
}

func refChoppiness(high, low, close []float64, n int) []float64 {
	size := len(close)
	tr := TrueRange(high, low, close)
	out := allNaN(size)
	logN := math.Log10(float64(n))
	for i := n - 1; i < size; i++ {
		var s float64
		for j := i - n + 1; j <= i; j++ {
			s += tr[j]
		}
		hi, lo := high[i], low[i]
		for j := i - n + 1; j <= i; j++ {
			if high[j] > hi {
				hi = high[j]
			}
			if low[j] < lo {
				lo = low[j]
			}
		}
		span := hi - lo
		if span <= 0 {
			out[i] = 0
			continue
		}
		out[i] = 100 * math.Log10(s/span) / logN
	}
	return out
}

// ---------------------------------------------------------------------------

func TestDirectionalMovementMatchesReference(t *testing.T) {
	for _, n := range []int{60, 120, 240} {
		high, low, close := synthHLC(n, int64(n)+1111)
		for _, p := range []int{5, 14, 20} {
			gadx, gp, gm := DirectionalMovement(high, low, close, p)
			wadx, wp, wm := refDirectionalMovement(high, low, close, p)
			assertSeries(t, "+DI", gp, wp, false)
			assertSeries(t, "-DI", gm, wm, false)
			assertSeries(t, "ADX", gadx, wadx, false)
		}
	}
}

// TestDirectionalMovementWarmup pins the nested-smoothing boundary.
func TestDirectionalMovementWarmup(t *testing.T) {
	const size = 120
	high, low, close := synthHLC(size, 77)
	for _, n := range []int{5, 14} {
		adx, pdi, mdi := DirectionalMovement(high, low, close, n)
		if got := FirstValid(pdi); got != n {
			t.Errorf("n=%d: +DI first valid = %d, want n = %d", n, got, n)
		}
		if got := FirstValid(mdi); got != n {
			t.Errorf("n=%d: -DI first valid = %d, want n = %d", n, got, n)
		}
		if got := FirstValid(adx); got != 2*n-1 {
			t.Errorf("n=%d: ADX first valid = %d, want 2n-1 = %d", n, got, 2*n-1)
		}
	}
}

// TestDirectionalMovementMonotonicSeries checks the direction of the signal on data
// where the answer is not in doubt, and pins the DI magnitude with a worked value.
//
// The bars here have a two-point range while moving one point per bar, so the true
// range is 2 and the directional movement is 1: +DI = 100 * 1/2 = 50, not 100. The
// ratio is the share of the true range that was directional movement, which is exactly
// what the indicator reports. A series whose bars gap so that the whole range is
// movement would give 100; that is a different data set, not a different formula.
func TestDirectionalMovementMonotonicSeries(t *testing.T) {
	const size = 80
	upHigh := make([]float64, size)
	upLow := make([]float64, size)
	closeUp := make([]float64, size)
	downHigh := make([]float64, size)
	downLow := make([]float64, size)
	closeDown := make([]float64, size)
	for i := 0; i < size; i++ {
		p := 100 + float64(i)
		upHigh[i], upLow[i], closeUp[i] = p+1, p-1, p
		q := 200 - float64(i)
		downHigh[i], downLow[i], closeDown[i] = q+1, q-1, q
	}

	adx, pdi, mdi := DirectionalMovement(upHigh, upLow, closeUp, 14)
	if math.Abs(pdi[size-1]-50) > 1e-9 {
		t.Errorf("+DI on a rising series = %v, want 50 (DM 1 over TR 2)", pdi[size-1])
	}
	if math.Abs(mdi[size-1]) > 1e-9 {
		t.Errorf("-DI on a rising series = %v, want 0", mdi[size-1])
	}
	// With -DI at zero, DX is 100 at every valid bar, so ADX converges to 100.
	if math.Abs(adx[size-1]-100) > 1e-9 {
		t.Errorf("ADX on a rising series = %v, want 100", adx[size-1])
	}

	_, pdi2, mdi2 := DirectionalMovement(downHigh, downLow, closeDown, 14)
	if math.Abs(pdi2[size-1]) > 1e-9 {
		t.Errorf("+DI on a falling series = %v, want 0", pdi2[size-1])
	}
	if math.Abs(mdi2[size-1]-50) > 1e-9 {
		t.Errorf("-DI on a falling series = %v, want 50", mdi2[size-1])
	}
}

// TestDirectionalMovementFlatSeriesIsZero pins the degenerate case: no range, no
// movement, so no directional information. Every line is 0 rather than NaN or an
// infinity.
func TestDirectionalMovementFlatSeriesIsZero(t *testing.T) {
	const size = 60
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = 50
	}
	adx, pdi, mdi := DirectionalMovement(flat, flat, flat, 14)
	for i := 2 * 14; i < size; i++ {
		if adx[i] != 0 || pdi[i] != 0 || mdi[i] != 0 {
			t.Fatalf("flat series at %d: adx=%v +DI=%v -DI=%v, want all 0", i, adx[i], pdi[i], mdi[i])
		}
	}
}

func TestAroonMatchesReference(t *testing.T) {
	for _, n := range []int{60, 120} {
		high, low, _ := synthHLC(n, int64(n)+2222)
		for _, p := range []int{5, 14, 25} {
			gu, gd, go_ := Aroon(high, low, p)
			wu, wd, wo := refAroon(high, low, p)
			assertSeries(t, "Aroon up", gu, wu, false)
			assertSeries(t, "Aroon down", gd, wd, false)
			assertSeries(t, "Aroon osc", go_, wo, false)
		}
	}
}

// TestAroonMonotonicSeries: a rising high makes the current bar the highest in every
// window, so Aroon up saturates at 100 while Aroon down falls to its floor.
func TestAroonMonotonicSeries(t *testing.T) {
	const size = 60
	const n = 14
	high := make([]float64, size)
	low := make([]float64, size)
	for i := 0; i < size; i++ {
		high[i] = 100 + float64(i)
		low[i] = 99 + float64(i)
	}
	up, down, osc := Aroon(high, low, n)
	if up[size-1] != 100 {
		t.Errorf("Aroon up on a rising series = %v, want 100", up[size-1])
	}
	// The lowest low is the oldest in the window, giving the floor 100/n.
	if math.Abs(down[size-1]-100.0/float64(n)) > 1e-9 {
		t.Errorf("Aroon down on a rising series = %v, want %v", down[size-1], 100.0/float64(n))
	}
	if math.Abs(osc[size-1]-(up[size-1]-down[size-1])) > 1e-12 {
		t.Errorf("Aroon osc = %v, want up-down", osc[size-1])
	}
}

// TestAroonTiesResolveToTheMostRecent pins the documented tie rule with a window whose
// extreme value appears twice.
func TestAroonTiesResolveToTheMostRecent(t *testing.T) {
	// high = {5, 9, 7, 9}: for n=4 at index 3 the highest value 9 appears at indices 1
	// and 3. Most recent gives barsSince 0 -> Aroon up 100.
	high := []float64{5, 9, 7, 9}
	low := []float64{1, 1, 1, 1}
	up, _, _ := Aroon(high, low, 4)
	if up[3] != 100 {
		t.Errorf("Aroon up with a repeated extreme = %v, want 100 (most recent)", up[3])
	}
}

func TestVortexMatchesReference(t *testing.T) {
	for _, n := range []int{60, 120} {
		high, low, close := synthHLC(n, int64(n)+3333)
		for _, p := range []int{5, 14, 21} {
			gp, gm := Vortex(high, low, close, p)
			wp, wm := refVortex(high, low, close, p)
			assertSeries(t, "VI+", gp, wp, false)
			assertSeries(t, "VI-", gm, wm, false)
		}
	}
}

// TestVortexNonNegative pins the sign: both lines are ratios of magnitudes, so neither
// can go negative. A sign error in the absolute values would show here.
func TestVortexNonNegative(t *testing.T) {
	high, low, close := synthHLC(120, 4444)
	vp, vm := Vortex(high, low, close, 14)
	for i := range close {
		if math.IsNaN(vp[i]) {
			continue
		}
		if vp[i] < 0 || vm[i] < 0 {
			t.Fatalf("Vortex at %d: VI+=%v VI-=%v, both must be non-negative", i, vp[i], vm[i])
		}
	}
}

func TestChoppinessMatchesReference(t *testing.T) {
	for _, n := range []int{60, 120} {
		high, low, close := synthHLC(n, int64(n)+5555)
		for _, p := range []int{5, 14} {
			assertSeries(t, "Choppiness", Choppiness(high, low, close, p), refChoppiness(high, low, close, p), false)
		}
	}
}

// TestChoppinessWithinRange checks the [0,100] clamp implicitly: the sum of true ranges
// is at least the window range, so the logarithm ratio is at most 1 and the index at
// most 100.
func TestChoppinessWithinRange(t *testing.T) {
	high, low, close := synthHLC(200, 6666)
	got := Choppiness(high, low, close, 14)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v < -1e-9 || v > 100+1e-9 {
			t.Fatalf("Choppiness[%d] = %v, outside [0,100]", i, v)
		}
	}
}

func TestChoppinessFlatSeriesIsZero(t *testing.T) {
	const size = 40
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = 10
	}
	got := Choppiness(flat, flat, flat, 14)
	for i := 13; i < size; i++ {
		if got[i] != 0 {
			t.Fatalf("Choppiness on a flat series[%d] = %v, want 0", i, got[i])
		}
	}
}

func TestKeltnerChannelsRelation(t *testing.T) {
	const size = 120
	const n, atrN = 20, 10
	const mult = 2.0
	high, low, close := synthHLC(size, 7777)

	upper, middle, lower := KeltnerChannels(high, low, close, n, atrN, mult)
	wantMiddle := EMA(close, n)
	atr := ATR(high, low, close, atrN)

	for i := range close {
		if !closeOrNaN(middle[i], wantMiddle[i]) {
			t.Fatalf("Keltner middle[%d] = %v, want EMA = %v", i, middle[i], wantMiddle[i])
		}
		if math.IsNaN(upper[i]) {
			continue
		}
		if !closeOrNaN(upper[i]-middle[i], mult*atr[i]) {
			t.Fatalf("Keltner upper gap[%d] = %v, want %v", i, upper[i]-middle[i], mult*atr[i])
		}
		if !closeOrNaN(middle[i]-lower[i], mult*atr[i]) {
			t.Fatalf("Keltner lower gap[%d] = %v, want %v", i, middle[i]-lower[i], mult*atr[i])
		}
	}
}

// TestSuperTrendInvariants checks the properties that define the indicator rather than
// comparing against a reimplementation: the direction is always ±1, the line stays on
// the correct side of price, and the bands ratchet in one direction within a trend.
func TestSuperTrendInvariants(t *testing.T) {
	const size = 300
	high, low, close := synthHLC(size, 8888)

	line, dir := SuperTrend(high, low, close, 10, 3)
	for i := range close {
		if math.IsNaN(line[i]) {
			if !math.IsNaN(dir[i]) {
				t.Fatalf("SuperTrend[%d]: line is NaN but direction is %v", i, dir[i])
			}
			continue
		}
		if dir[i] != 1 && dir[i] != -1 {
			t.Fatalf("SuperTrend direction[%d] = %v, want ±1", i, dir[i])
		}
		// Uptrend: the line is a stop below price. Downtrend: an overlay above it.
		if dir[i] == 1 && line[i] > high[i]+1e-9 {
			t.Fatalf("SuperTrend[%d]: uptrend line %v is above the high %v", i, line[i], high[i])
		}
		if dir[i] == -1 && line[i] < low[i]-1e-9 {
			t.Fatalf("SuperTrend[%d]: downtrend line %v is below the low %v", i, line[i], low[i])
		}
		// Within a run of the same direction the stop may only tighten.
		if i > 0 && !math.IsNaN(line[i-1]) && dir[i] == dir[i-1] {
			if dir[i] == 1 && line[i] < line[i-1]-1e-9 {
				t.Fatalf("SuperTrend[%d]: uptrend stop loosened from %v to %v", i, line[i-1], line[i])
			}
			if dir[i] == -1 && line[i] > line[i-1]+1e-9 {
				t.Fatalf("SuperTrend[%d]: downtrend stop loosened from %v to %v", i, line[i-1], line[i])
			}
		}
	}

	// The direction must flip at least once over a long random walk.
	flips := 0
	for i := 1; i < size; i++ {
		if dir[i] != dir[i-1] {
			flips++
		}
	}
	if flips == 0 {
		t.Error("SuperTrend never changed direction over a random walk; the test data is not exercising it")
	}
}

// TestParabolicSARInvariants checks that the SAR stays on the correct side of price,
// which is what makes it a usable stop, and that it reverses on a trend change.
func TestParabolicSARInvariants(t *testing.T) {
	const size = 300
	high, low, close := synthHLC(size, 9999)

	sar, dir := ParabolicSAR(high, low, start01, increment02, max02)
	for i := range close {
		if dir[i] != 1 && dir[i] != -1 {
			t.Fatalf("PSAR direction[%d] = %v, want ±1", i, dir[i])
		}
		// On the bar where the direction changes, the SAR is set to the extreme
		// point of the trend that just ended. A wide reversal bar can therefore
		// contain its own SAR, which is a property of Wilder's definition rather
		// than an error, so the invariant is asserted only while the trend persists.
		reversal := i > 0 && dir[i] != dir[i-1]
		if !reversal && dir[i] == 1 && sar[i] > low[i]+1e-9 {
			t.Fatalf("PSAR[%d]: uptrend SAR %v is above the low %v", i, sar[i], low[i])
		}
		if !reversal && dir[i] == -1 && sar[i] < high[i]-1e-9 {
			t.Fatalf("PSAR[%d]: downtrend SAR %v is below the high %v", i, sar[i], high[i])
		}
	}

	flips := 0
	for i := 1; i < size; i++ {
		if dir[i] != dir[i-1] {
			flips++
		}
	}
	if flips == 0 {
		t.Error("PSAR never reversed over a random walk; the test data is not exercising it")
	}
}

// Constants for the PSAR tests, named so the call sites read as Wilder's defaults.
const (
	start01     = 0.02
	increment02 = 0.02
	max02       = 0.2
)

// TestParabolicSARTrendingSeries keeps the SAR on one side through a monotonic move.
func TestParabolicSARTrendingSeries(t *testing.T) {
	const size = 60
	high := make([]float64, size)
	low := make([]float64, size)
	for i := 0; i < size; i++ {
		p := 100 + 2*float64(i)
		high[i], low[i] = p+1, p-1
	}
	sar, dir := ParabolicSAR(high, low, start01, increment02, max02)
	for i := range high {
		if dir[i] != 1 {
			t.Fatalf("PSAR[%d] direction = %v on a rising series, want +1", i, dir[i])
		}
		if sar[i] > low[i] {
			t.Fatalf("PSAR[%d] = %v is above the low %v", i, sar[i], low[i])
		}
	}
	// The SAR must ratchet upward as the trend extends.
	if sar[size-1] <= sar[0] {
		t.Errorf("PSAR did not rise with the trend: %v -> %v", sar[0], sar[size-1])
	}
}

func TestTrendIndicatorsEmptyAndPanics(t *testing.T) {
	if adx, p, m := DirectionalMovement(nil, nil, nil, 14); adx != nil || p != nil || m != nil {
		t.Error("empty DirectionalMovement should return nils")
	}
	if u, d, o := Aroon(nil, nil, 14); u != nil || d != nil || o != nil {
		t.Error("empty Aroon should return nils")
	}
	if p, m := Vortex(nil, nil, nil, 14); p != nil || m != nil {
		t.Error("empty Vortex should return nils")
	}
	if Choppiness(nil, nil, nil, 14) != nil {
		t.Error("empty Choppiness should be nil")
	}
	if u, m, l := KeltnerChannels(nil, nil, nil, 20, 10, 2); u != nil || m != nil || l != nil {
		t.Error("empty KeltnerChannels should return nils")
	}
	if l, d := SuperTrend(nil, nil, nil, 10, 3); l != nil || d != nil {
		t.Error("empty SuperTrend should return nils")
	}
	if s, d := ParabolicSAR(nil, nil, start01, increment02, max02); s != nil || d != nil {
		t.Error("empty ParabolicSAR should return nils")
	}

	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"DM period", func() { DirectionalMovement(long, long, long, 0) }},
		{"DM length", func() { DirectionalMovement(short, short, long, 5) }},
		{"Aroon period", func() { Aroon(long, long, 0) }},
		{"Aroon length", func() { Aroon(short, long, 5) }},
		{"Vortex period", func() { Vortex(long, long, long, -1) }},
		{"Choppiness period", func() { Choppiness(long, long, long, 0) }},
		{"Keltner atrN", func() { KeltnerChannels(long, long, long, 20, 0, 2) }},
		{"SuperTrend atrN", func() { SuperTrend(long, long, long, 0, 3) }},
		{"PSAR increment", func() { ParabolicSAR(long, long, 0.02, 0, 0.2) }},
		{"PSAR max", func() { ParabolicSAR(long, long, 0.5, 0.02, 0.2) }},
		{"PSAR length", func() { ParabolicSAR(short, long, start01, increment02, max02) }},
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
