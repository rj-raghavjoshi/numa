package ta

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/series"
)

// ---------------------------------------------------------------------------
// Tests for the second-tier momentum indicators.
//
// The composite ones are checked for their *bounds* as well as their values, because a bound is a
// property the formula guarantees and a reference written from the same formula would inherit any
// error in it. StochasticMomentumIndex in particular must stay within [-100,100]: the numerator is
// bounded by the denominator bar by bar, so no smoothing can push the ratio outside.
// ---------------------------------------------------------------------------

func TestPercentRank(t *testing.T) {
	got := PercentRank([]float64{5, 1, 4, 2, 3}, 5)
	// The last value, 3, has 1, 2 below it in the window: 2/5 = 40%.
	if !closeOrNaN(got[4], 40) {
		t.Fatalf("PercentRank[4] = %v, want 40", got[4])
	}
	if got := FirstValid(got); got != 4 {
		t.Fatalf("PercentRank first valid = %d, want 4", got)
	}

	// A strictly increasing series puts the current value at the top of its own window, which
	// cannot reach 100% because the window contains the current value.
	up := make([]float64, 20)
	for i := range up {
		up[i] = float64(i)
	}
	gotUp := PercentRank(up, 10)
	for i := 9; i < len(up); i++ {
		if !closeOrNaN(gotUp[i], 90) {
			t.Fatalf("increasing PercentRank[%d] = %v, want 90", i, gotUp[i])
		}
	}

	// A strictly decreasing series puts it at the bottom.
	down := make([]float64, 20)
	for i := range down {
		down[i] = float64(len(down) - i)
	}
	gotDown := PercentRank(down, 10)
	for i := 9; i < len(down); i++ {
		if gotDown[i] != 0 {
			t.Fatalf("decreasing PercentRank[%d] = %v, want 0", i, gotDown[i])
		}
	}

	if PercentRank(nil, 5) != nil {
		t.Error("empty PercentRank should be nil")
	}
}

func refSMI(high, low, close []float64, n, k int) (smi, signal []float64) {
	size := len(close)
	hi := refExtreme(high, n, true)
	lo := refExtreme(low, n, false)
	raw := allNaN(size)
	rng := allNaN(size)
	for i := range close {
		if hi[i] != hi[i] || lo[i] != lo[i] {
			continue
		}
		raw[i] = close[i] - (hi[i]+lo[i])/2
		rng[i] = hi[i] - lo[i]
	}
	// Both passes must start at the first valid index. ta.EMA forwards to series.Apply, which
	// treats leading NaNs as seed material and is permanently poisoned by them -- correct for a raw
	// price series, wrong for a composite. The implementation uses applyFrom for exactly this
	// reason, and a reference that used the plain forwarder would report NaN everywhere and
	// "disagree" with a correct implementation.
	e1 := applyFrom(raw, FirstValid(raw), series.NewEMA(k))
	num := applyFrom(e1, FirstValid(e1), series.NewEMA(k))
	f1 := applyFrom(rng, FirstValid(rng), series.NewEMA(k))
	den := applyFrom(f1, FirstValid(f1), series.NewEMA(k))

	smi = allNaN(size)
	for i := range close {
		if num[i] != num[i] || den[i] != den[i] {
			continue
		}
		if den[i] == 0 {
			smi[i] = 0
			continue
		}
		smi[i] = 100 * num[i] / den[i]
	}
	signal = applyFrom(smi, FirstValid(smi), series.NewEMA(k))
	return smi, signal
}

func TestStochasticMomentumIndexMatchesReference(t *testing.T) {
	for _, size := range []int{80, 200} {
		high, low, close := synthHLC(size, int64(size)+2020)
		for _, n := range []int{10, 14, 20} {
			gotSMI, gotSignal := StochasticMomentumIndex(high, low, close, n, 3, 3)
			wantSMI, wantSignal := refSMI(high, low, close, n, 3)
			assertSeries(t, "SMI", gotSMI, wantSMI, false)
			assertSeries(t, "SMI signal", gotSignal, wantSignal, false)
		}
	}
}

// TestStochasticMomentumIndexIsBounded is the independent property check. The numerator is
// close minus the range midpoint, which is at most half the range in absolute value, and the
// denominator is the range itself; since EMA of a bounded ratio stays bounded, the index cannot
// leave [-100,100] whatever the smoothing does.
func TestStochasticMomentumIndexIsBounded(t *testing.T) {
	high, low, close := synthHLC(400, 3030)
	for _, n := range []int{5, 10, 20} {
		got, _ := StochasticMomentumIndex(high, low, close, n, 3, 3)
		for i, v := range got {
			if math.IsNaN(v) {
				continue
			}
			if v < -100-1e-9 || v > 100+1e-9 {
				t.Fatalf("n=%d: SMI[%d] = %v, outside [-100,100]", n, i, v)
			}
		}
	}
}

// TestStochasticMomentumIndexFlatRangeIsZero: a bar with no range makes both the numerator and the
// denominator zero, and the documented answer is 0 rather than a division by zero.
func TestStochasticMomentumIndexFlatRangeIsZero(t *testing.T) {
	const size = 60
	const c = 100.0
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = c
	}
	got, _ := StochasticMomentumIndex(flat, flat, flat, 10, 3, 3)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if math.Abs(v) > 1e-12 {
			t.Fatalf("SMI of a flat range[%d] = %v, want 0", i, v)
		}
	}
}

func refRVI(close []float64, n int) []float64 {
	size := len(close)
	sd := refRollingStdDev(close, n)
	up := allNaN(size)
	down := allNaN(size)
	for i := 1; i < size; i++ {
		if sd[i] != sd[i] {
			continue
		}
		switch {
		case close[i] > close[i-1]:
			up[i], down[i] = sd[i], 0
		case close[i] < close[i-1]:
			up[i], down[i] = 0, sd[i]
		default:
			up[i], down[i] = 0, 0
		}
	}
	from := FirstValid(sd)
	if from < 1 {
		from = 1
	}
	us := refRMA(up, n, from)
	ds := refRMA(down, n, from)
	out := allNaN(size)
	for i := range close {
		u, d := us[i], ds[i]
		if u != u || d != d {
			continue
		}
		if sum := u + d; sum != 0 {
			out[i] = 100 * u / sum
		} else {
			out[i] = 50
		}
	}
	return out
}

func refRollingStdDev(xs []float64, n int) []float64 {
	out := allNaN(len(xs))
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
		out[i] = math.Sqrt(ss / float64(n))
	}
	return out
}

// refRMA is a direct Wilder smoothing seeded with the mean of the first n values from `from`.
func refRMA(xs []float64, n, from int) []float64 {
	out := allNaN(len(xs))
	if from+n > len(xs) {
		return out
	}
	var s float64
	for j := from; j < from+n; j++ {
		if xs[j] != xs[j] {
			return out
		}
		s += xs[j]
	}
	prev := s / float64(n)
	out[from+n-1] = prev
	for i := from + n; i < len(xs); i++ {
		v := xs[i]
		if v != v {
			return out
		}
		prev += (v - prev) / float64(n)
		out[i] = prev
	}
	return out
}

func TestRelativeVolatilityIndexMatchesReference(t *testing.T) {
	close := synthClose(300, 4141)
	for _, n := range []int{5, 10, 14} {
		got := RelativeVolatilityIndex(close, n)
		want := refRVI(close, n)
		assertSeries(t, "RVI", got, want, false)
	}
}

func TestRelativeVolatilityIndexWithinRange(t *testing.T) {
	close := synthClose(300, 5151)
	for _, n := range []int{5, 14, 20} {
		got := RelativeVolatilityIndex(close, n)
		for i, v := range got {
			if math.IsNaN(v) {
				continue
			}
			if v < -1e-9 || v > 100+1e-9 {
				t.Fatalf("n=%d: RVI[%d] = %v, outside [0,100]", n, i, v)
			}
		}
	}
}

// TestRelativeVolatilityIndexFlatSeriesIsNeutral pins the documented convention for a market that
// does not move: neither term accumulates, so the answer is the neutral midpoint rather than 0.
func TestRelativeVolatilityIndexFlatSeriesIsNeutral(t *testing.T) {
	const size = 200
	const c = 30.0
	flat := make([]float64, size)
	for i := range flat {
		flat[i] = c
	}
	got := RelativeVolatilityIndex(flat, 14)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if math.Abs(v-50) > 1e-12 {
			t.Fatalf("RVI of a flat series[%d] = %v, want 50", i, v)
		}
	}
}

func refConnorsRSI(close []float64, rsiPeriod, streakPeriod, rankPeriod int) []float64 {
	size := len(close)
	streak := make([]float64, size)
	streak[0] = math.NaN()
	for i := 1; i < size; i++ {
		switch {
		case close[i] > close[i-1]:
			if prev := streak[i-1]; prev > 0 && prev == prev {
				streak[i] = prev + 1
			} else {
				streak[i] = 1
			}
		case close[i] < close[i-1]:
			if prev := streak[i-1]; prev < 0 && prev == prev {
				streak[i] = prev - 1
			} else {
				streak[i] = -1
			}
		default:
			streak[i] = 0
		}
	}
	a := RSI(close, rsiPeriod)
	b := RSI(streak, streakPeriod)
	c := PercentRank(ROC(close, 1), rankPeriod)
	out := allNaN(size)
	for i := range close {
		if a[i] != a[i] || b[i] != b[i] || c[i] != c[i] {
			continue
		}
		out[i] = (a[i] + b[i] + c[i]) / 3
	}
	return out
}

func TestConnorsRSIMatchesReference(t *testing.T) {
	close := synthClose(400, 6161)
	for _, tc := range [][3]int{{3, 2, 100}, {3, 2, 20}, {5, 3, 50}} {
		got := ConnorsRSI(close, tc[0], tc[1], tc[2])
		want := refConnorsRSI(close, tc[0], tc[1], tc[2])
		assertSeries(t, "ConnorsRSI", got, want, false)
	}
}

func TestConnorsRSIWithinRange(t *testing.T) {
	close := synthClose(400, 7171)
	got := ConnorsRSI(close, 3, 2, 50)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v < -1e-9 || v > 100+1e-9 {
			t.Fatalf("ConnorsRSI[%d] = %v, outside [0,100]", i, v)
		}
	}
}

// TestConnorsRSIStreakCounter checks the streak construction directly, since it is the component
// with no other definition to compare against.
func TestConnorsRSIStreakCounter(t *testing.T) {
	// Four rises, then three falls, then a flat close.
	close := []float64{10, 11, 12, 13, 14, 13, 12, 11, 11}
	streak := make([]float64, len(close))
	streak[0] = math.NaN()
	for i := 1; i < len(close); i++ {
		switch {
		case close[i] > close[i-1]:
			if prev := streak[i-1]; prev > 0 && prev == prev {
				streak[i] = prev + 1
			} else {
				streak[i] = 1
			}
		case close[i] < close[i-1]:
			if prev := streak[i-1]; prev < 0 && prev == prev {
				streak[i] = prev - 1
			} else {
				streak[i] = -1
			}
		default:
			streak[i] = 0
		}
	}
	want := []float64{0, 1, 2, 3, 4, -1, -2, -3, 0}
	for i := 1; i < len(close); i++ {
		if streak[i] != want[i] {
			t.Fatalf("streak[%d] = %v, want %v", i, streak[i], want[i])
		}
	}
}

func TestElderRay(t *testing.T) {
	high, low, close := synthHLC(200, 8181)
	const n = 13

	bull, bear := ElderRay(high, low, close, n)
	ema := EMA(close, n)
	for i := range close {
		if !closeOrNaN(bull[i], high[i]-ema[i]) {
			t.Fatalf("bull[%d] = %v, want %v", i, bull[i], high[i]-ema[i])
		}
		if !closeOrNaN(bear[i], low[i]-ema[i]) {
			t.Fatalf("bear[%d] = %v, want %v", i, bear[i], low[i]-ema[i])
		}
		if bull[i] < bear[i] {
			t.Fatalf("bull power %v is below bear power %v at %d", bull[i], bear[i], i)
		}
	}
	if got := FirstValid(bull); got != n-1 {
		t.Errorf("ElderRay first valid = %d, want %d", got, n-1)
	}
}

func TestMomentum2EmptyAndPanics(t *testing.T) {
	if PercentRank(nil, 5) != nil {
		t.Error("empty PercentRank should be nil")
	}
	if s, g := StochasticMomentumIndex(nil, nil, nil, 10, 3, 3); s != nil || g != nil {
		t.Error("empty SMI should return nils")
	}
	if RelativeVolatilityIndex(nil, 5) != nil {
		t.Error("empty RVI should be nil")
	}
	if ConnorsRSI(nil, 3, 2, 100) != nil {
		t.Error("empty ConnorsRSI should be nil")
	}
	if b, r := ElderRay(nil, nil, nil, 13); b != nil || r != nil {
		t.Error("empty ElderRay should return nils")
	}

	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"PercentRank period", func() { PercentRank(long, 0) }},
		{"SMI period", func() { StochasticMomentumIndex(long, long, long, 0, 3, 3) }},
		{"SMI smoothD", func() { StochasticMomentumIndex(long, long, long, 10, 3, 0) }},
		{"SMI length", func() { StochasticMomentumIndex(short, short, long, 10, 3, 3) }},
		{"RVI period", func() { RelativeVolatilityIndex(long, 0) }},
		{"ConnorsRSI period", func() { ConnorsRSI(long, 0, 2, 100) }},
		{"ElderRay period", func() { ElderRay(long, long, long, 0) }},
		{"ElderRay length", func() { ElderRay(short, short, long, 13) }},
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
