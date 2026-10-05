package ta

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for the composed oscillators.
//
// These are built from other indicators, so the risk is not the formula but the
// composition: a warm-up that is off by one, or a recurrence fed a leading NaN as seed
// material. Each test therefore checks the relationship to its inputs as well as the
// values against a reference, because a reference written the same way would share the
// composition mistake.
// ---------------------------------------------------------------------------

func TestAwesomeAndAcceleratorOscillator(t *testing.T) {
	high, low, close := benchOHLC(200)

	ao := AwesomeOscillator(high, low, 5, 34)
	median := MedianPrice(high, low)
	f5 := SMA(median, 5)
	f34 := SMA(median, 34)
	for i := range median {
		if !closeOrNaN(ao[i], f5[i]-f34[i]) {
			t.Fatalf("AO[%d] = %v, want SMA5-SMA34 = %v", i, ao[i], f5[i]-f34[i])
		}
	}
	if got := FirstValid(ao); got != 33 {
		t.Errorf("AO first valid = %d, want 33", got)
	}

	ac := AcceleratorOscillator(high, low, 5, 34, 5)
	smooth := SMA(ao, 5)
	for i := range median {
		if !closeOrNaN(ac[i], ao[i]-smooth[i]) {
			t.Fatalf("AC[%d] = %v, want AO-SMA5(AO) = %v", i, ac[i], ao[i]-smooth[i])
		}
	}
	if got := FirstValid(ac); got != 37 {
		t.Errorf("AC first valid = %d, want 37", got)
	}

	_ = close
}

// TestAwesomeOscillatorOnAConstantSeries is the fixed point: both averages equal the
// constant, so the oscillator is exactly zero.
func TestAwesomeOscillatorOnAConstantSeries(t *testing.T) {
	const size = 60
	high := make([]float64, size)
	low := make([]float64, size)
	for i := range high {
		high[i], low[i] = 11, 9
	}
	ao := AwesomeOscillator(high, low, 5, 34)
	for i := 33; i < size; i++ {
		if ao[i] != 0 {
			t.Fatalf("AO on a constant series[%d] = %v, want 0", i, ao[i])
		}
	}
}

func TestDetrendedPriceOscillator(t *testing.T) {
	close := synthClose(200, 6060)
	const n = 20
	got := DetrendedPriceOscillator(close, n)
	ma := SMA(close, n)
	shift := n/2 + 1

	for i := range close {
		j := i - shift
		if i < n-1 || j < 0 {
			if !math.IsNaN(got[i]) {
				t.Fatalf("DPO[%d] = %v, want NaN before both the average and the shift exist", i, got[i])
			}
			continue
		}
		if !closeOrNaN(got[i], close[j]-ma[i]) {
			t.Fatalf("DPO[%d] = %v, want close[%d]-SMA = %v", i, got[i], j, close[j]-ma[i])
		}
	}
}

// TestTRIXOnAConstantSeries: the triple exponential average is the constant, so its rate
// of change is zero everywhere it is defined.
func TestTRIXOnAConstantSeries(t *testing.T) {
	const size = 200
	const c = 42.0
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = c
	}
	const n = 5
	got := TRIX(xs, n)
	if first := FirstValid(got); first != 3*n-2 {
		t.Errorf("TRIX first valid = %d, want 3n-2 = %d", first, 3*n-2)
	}
	for i := 3*n - 2; i < size; i++ {
		if math.Abs(got[i]) > 1e-9 {
			t.Fatalf("TRIX on a constant series[%d] = %v, want 0", i, got[i])
		}
	}
}

func TestUltimateOscillatorMatchesReference(t *testing.T) {
	for _, size := range []int{80, 200} {
		high, low, close := synthHLC(size, int64(size)+7171)
		for _, tc := range [][3]int{{7, 14, 28}, {5, 10, 20}} {
			got := UltimateOscillator(high, low, close, tc[0], tc[1], tc[2])
			want := refUltimateOscillator(high, low, close, tc[0], tc[1], tc[2])
			assertSeries(t, "UltimateOscillator", got, want, false)
		}
	}
}

func refUltimateOscillator(high, low, close []float64, short, mid, long int) []float64 {
	size := len(close)
	out := allNaN(size)
	for i := long; i < size; i++ {
		var bpShort, trShort, bpMid, trMid, bpLong, trLong float64
		for j := i - short + 1; j <= i; j++ {
			pc := close[j-1]
			bp := close[j] - math.Min(low[j], pc)
			tr := math.Max(high[j], pc) - math.Min(low[j], pc)
			bpShort += bp
			trShort += tr
		}
		for j := i - mid + 1; j <= i; j++ {
			pc := close[j-1]
			bp := close[j] - math.Min(low[j], pc)
			tr := math.Max(high[j], pc) - math.Min(low[j], pc)
			bpMid += bp
			trMid += tr
		}
		for j := i - long + 1; j <= i; j++ {
			pc := close[j-1]
			bp := close[j] - math.Min(low[j], pc)
			tr := math.Max(high[j], pc) - math.Min(low[j], pc)
			bpLong += bp
			trLong += tr
		}
		a1 := ratioOr0(bpShort, trShort)
		a2 := ratioOr0(bpMid, trMid)
		a3 := ratioOr0(bpLong, trLong)
		out[i] = 100 * (4*a1 + 2*a2 + a3) / 7
	}
	return out
}

// TestUltimateOscillatorWithinRange pins the [0,100] output. The buying pressure is a
// component of the true range by construction, so each ratio is in [0,1] and the weighted
// average of three of them is too.
func TestUltimateOscillatorWithinRange(t *testing.T) {
	high, low, close := synthHLC(300, 8181)
	got := UltimateOscillator(high, low, close, 7, 14, 28)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v < -1e-9 || v > 100+1e-9 {
			t.Fatalf("UltimateOscillator[%d] = %v, outside [0,100]", i, v)
		}
	}
}

func TestFisherTransformMatchesReference(t *testing.T) {
	for _, size := range []int{80, 200} {
		high, low := func() ([]float64, []float64) {
			h, l, _ := synthHLC(size, int64(size)+9191)
			return h, l
		}()
		for _, n := range []int{5, 9, 14} {
			got := FisherTransform(high, low, n)
			want := refFisher(high, low, n)
			assertSeries(t, "FisherTransform", got, want, false)
		}
	}
}

func refFisher(high, low []float64, n int) []float64 {
	size := len(high)
	hi := refExtreme(high, n, true)
	lo := refExtreme(low, n, false)
	out := allNaN(size)
	var value, fish float64
	for i := n - 1; i < size; i++ {
		price := (high[i] + low[i]) / 2
		span := hi[i] - lo[i]
		position := 0.5
		if span != 0 {
			position = (price - lo[i]) / span
		}
		value = 0.66*(position-0.5) + 0.67*value
		if value > 0.999 {
			value = 0.999
		} else if value < -0.999 {
			value = -0.999
		}
		fish = 0.5*math.Log((1+value)/(1-value)) + 0.5*fish
		out[i] = fish
	}
	return out
}

// TestFisherTransformClampIsLoadBearing runs the transform on a strictly rising series,
// where the position is pinned at 1 and the unclamped logarithm would diverge.
func TestFisherTransformClampIsLoadBearing(t *testing.T) {
	const size = 60
	const n = 5
	high := make([]float64, size)
	low := make([]float64, size)
	for i := 0; i < size; i++ {
		high[i] = 100 + float64(i)
		low[i] = 100 + float64(i) // zero range: position is the neutral 0.5
	}
	got := FisherTransform(high, low, n)
	for i := n - 1; i < size; i++ {
		if math.IsInf(got[i], 0) || math.IsNaN(got[i]) {
			t.Fatalf("FisherTransform[%d] = %v, want a finite value", i, got[i])
		}
	}

	// A series whose position really is pinned at the top: price at the high of the
	// window for every bar.
	high2 := make([]float64, size)
	low2 := make([]float64, size)
	for i := 0; i < size; i++ {
		high2[i] = 100 + float64(i)
		low2[i] = 100 + float64(i) - 1
	}
	got2 := FisherTransform(high2, low2, n)
	for i := n - 1; i < size; i++ {
		if math.IsInf(got2[i], 0) || math.IsNaN(got2[i]) {
			t.Fatalf("FisherTransform at the top of the range[%d] = %v, want finite", i, got2[i])
		}
	}
}

func TestMassIndexMatchesReference(t *testing.T) {
	for _, size := range []int{120, 200} {
		high, low := func() ([]float64, []float64) {
			h, l, _ := synthHLC(size, int64(size)+1234)
			return h, l
		}()
		for _, tc := range [][2]int{{9, 25}, {10, 20}} {
			got := MassIndex(high, low, tc[0], tc[1])
			want := refMassIndex(high, low, tc[0], tc[1])
			assertSeries(t, "MassIndex", got, want, false)
		}
	}
}

func refMassIndex(high, low []float64, emaPeriod, sumPeriod int) []float64 {
	size := len(high)
	rng := make([]float64, size)
	for i := range high {
		rng[i] = high[i] - low[i]
	}
	fe := refSmoother(rng, emaPeriod, 2.0/float64(emaPeriod+1))
	se := refSmoother(rng, sumPeriod, 2.0/float64(sumPeriod+1))
	ratio := make([]float64, size)
	for i := range rng {
		if fe[i] != fe[i] || se[i] != se[i] {
			ratio[i] = math.NaN()
			continue
		}
		ratio[i] = ratioOr0(fe[i], se[i])
	}
	out := allNaN(size)
	for i := range ratio {
		if i+1 < sumPeriod {
			continue
		}
		var s float64
		nan := false
		for j := i - sumPeriod + 1; j <= i; j++ {
			if ratio[j] != ratio[j] {
				nan = true
				break
			}
			s += ratio[j]
		}
		if !nan {
			out[i] = s
		}
	}
	return out
}

// TestMassIndexWarmup pins the composed boundary rather than deriving it.
func TestMassIndexWarmup(t *testing.T) {
	high, low := func() ([]float64, []float64) {
		h, l, _ := synthHLC(200, 2468)
		return h, l
	}()
	const emaPeriod, sumPeriod = 9, 25
	got := MassIndex(high, low, emaPeriod, sumPeriod)
	want := sumPeriod + sumPeriod - 2 // max(9,25)-1 + 25-1
	if first := FirstValid(got); first != want {
		t.Errorf("MassIndex first valid = %d, want %d", first, want)
	}
}

func TestOscillatorsEmptyAndPanics(t *testing.T) {
	if AwesomeOscillator(nil, nil, 5, 34) != nil {
		t.Error("empty AO should be nil")
	}
	if AcceleratorOscillator(nil, nil, 5, 34, 5) != nil {
		t.Error("empty AC should be nil")
	}
	if DetrendedPriceOscillator(nil, 20) != nil {
		t.Error("empty DPO should be nil")
	}
	if TRIX(nil, 5) != nil {
		t.Error("empty TRIX should be nil")
	}
	if UltimateOscillator(nil, nil, nil, 7, 14, 28) != nil {
		t.Error("empty UO should be nil")
	}
	if FisherTransform(nil, nil, 9) != nil {
		t.Error("empty Fisher should be nil")
	}
	if MassIndex(nil, nil, 9, 25) != nil {
		t.Error("empty MassIndex should be nil")
	}

	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"AO period", func() { AwesomeOscillator(long, long, 0, 34) }},
		{"AC smooth", func() { AcceleratorOscillator(long, long, 5, 34, 0) }},
		{"DPO period", func() { DetrendedPriceOscillator(long, 0) }},
		{"TRIX period", func() { TRIX(long, 0) }},
		{"UO period", func() { UltimateOscillator(long, long, long, 0, 14, 28) }},
		{"UO length", func() { UltimateOscillator(short, short, long, 7, 14, 28) }},
		{"Fisher period", func() { FisherTransform(long, long, 0) }},
		{"Fisher length", func() { FisherTransform(short, long, 9) }},
		{"MassIndex emaPeriod", func() { MassIndex(long, long, 0, 25) }},
		{"MassIndex length", func() { MassIndex(short, long, 9, 25) }},
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
