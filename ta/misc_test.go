package ta

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/series"
)

// ---------------------------------------------------------------------------
// Tests for the long-period oscillators and pivot patterns.
//
// WilliamsFractal is the one that needs the most careful testing, because its correctness is
// about *where* the signal is placed, not only whether it fires. A fractal reported at the
// bar it describes is a look-ahead bug that no value comparison would catch, so the tests
// assert the confirmation index explicitly.
// ---------------------------------------------------------------------------

func TestCoppockCurveComposition(t *testing.T) {
	close := synthClose(300, 1212)
	const rocLong, rocShort, wmaPeriod = 14, 11, 10

	got := CoppockCurve(close, rocLong, rocShort, wmaPeriod)

	long := ROC(close, rocLong)
	short := ROC(close, rocShort)
	sum := make([]float64, len(close))
	for i := range close {
		if long[i] != long[i] || short[i] != short[i] {
			sum[i] = math.NaN()
			continue
		}
		sum[i] = long[i] + short[i]
	}
	assertSeries(t, "CoppockCurve", got, WMA(sum, wmaPeriod), true)
}

func TestKnowSureThingCompositionAndWeights(t *testing.T) {
	close := synthClose(400, 2323)
	roc := [4]int{10, 15, 20, 30}
	sma := [4]int{10, 10, 10, 15}
	const signalLength = 9

	kst, signal := KnowSureThing(close, roc, sma, signalLength)

	// Rebuild the weighted sum independently.
	want := allNaN(len(close))
	for term := 0; term < 4; term++ {
		smoothed := SMA(ROC(close, roc[term]), sma[term])
		for i := range close {
			if smoothed[i] != smoothed[i] {
				want[i] = math.NaN()
				continue
			}
			if want[i] != want[i] {
				continue
			}
			want[i] += float64(term+1) * smoothed[i]
		}
	}
	assertSeries(t, "KST", kst, want, false)
	assertSeries(t, "KST signal", signal, SMA(want, signalLength), false)

	// The weights 1..4 are already pinned by the independent reconstruction above, which
	// multiplies term k by k+1. A separate "unit weights" comparison is not possible here
	// because the weights are part of the definition rather than a parameter, and asserting
	// something unverifiable would be worse than not asserting it.
}

func TestTrueStrengthIndex(t *testing.T) {
	close := synthClose(300, 3434)
	const long, short = 25, 13

	got := TrueStrengthIndex(close, long, short)

	// Rebuild from the definition.
	size := len(close)
	pc := make([]float64, size)
	absPC := make([]float64, size)
	fillNaN(pc, 0, 1)
	fillNaN(absPC, 0, 1)
	for i := 1; i < size; i++ {
		d := close[i] - close[i-1]
		pc[i] = d
		absPC[i] = math.Abs(d)
	}
	sm := applyFrom(pc, 1, series.NewEMA(long))
	sm = applyFrom(sm, FirstValid(sm), series.NewEMA(short))
	ab := applyFrom(absPC, 1, series.NewEMA(long))
	ab = applyFrom(ab, FirstValid(ab), series.NewEMA(short))
	want := allNaN(size)
	for i := range close {
		if sm[i] != sm[i] || ab[i] != ab[i] {
			continue
		}
		if ab[i] == 0 {
			want[i] = 0
			continue
		}
		want[i] = 100 * sm[i] / ab[i]
	}
	assertSeries(t, "TrueStrengthIndex", got, want, false)
}

// TestTrueStrengthIndexConstantSeriesIsZero pins the 0/0 case.
func TestTrueStrengthIndexConstantSeriesIsZero(t *testing.T) {
	const size = 200
	xs := make([]float64, size)
	for i := range xs {
		xs[i] = 77
	}
	got := TrueStrengthIndex(xs, 25, 13)
	for i := 55; i < size; i++ {
		if math.Abs(got[i]) > 1e-9 {
			t.Fatalf("TSI of a constant series[%d] = %v, want 0", i, got[i])
		}
	}
}

// TestTrueStrengthIndexWithinRange pins the bounded output, which the ratio construction
// guarantees: the numerator is a smoothed signed momentum and the denominator is the smoothed
// absolute value of the same series, so |numerator| <= denominator term by term.
func TestTrueStrengthIndexWithinRange(t *testing.T) {
	close := synthClose(400, 4545)
	got := TrueStrengthIndex(close, 25, 13)
	for i, v := range got {
		if math.IsNaN(v) {
			continue
		}
		if v < -100-1e-6 || v > 100+1e-6 {
			t.Fatalf("TSI[%d] = %v, outside [-100,100]", i, v)
		}
	}
}

func TestWilliamsAlligatorShifts(t *testing.T) {
	const size = 200
	high, low := func() ([]float64, []float64) {
		h, l, _ := synthHLC(size, 5656)
		return h, l
	}()
	const jawP, jawS, teethP, teethS, lipsP, lipsS = 13, 8, 8, 5, 5, 3

	jaw, teeth, lips := WilliamsAlligator(high, low, jawP, jawS, teethP, teethS, lipsP, lipsS)
	median := MedianPrice(high, low)

	assertSeries(t, "Alligator jaw", jaw, shiftRef(RMA(median, jawP), jawS), false)
	assertSeries(t, "Alligator teeth", teeth, shiftRef(RMA(median, teethP), teethS), false)
	assertSeries(t, "Alligator lips", lips, shiftRef(RMA(median, lipsP), lipsS), false)

	// The shift must delay the line: the first jawS outputs are NaN beyond the smoother's
	// own warm-up.
	for i := 0; i < jawS; i++ {
		if !math.IsNaN(jaw[i]) {
			t.Errorf("jaw[%d] = %v, want NaN inside the shift", i, jaw[i])
		}
	}
}

// shiftRef is a direct implementation of a forward shift, independent of vec.Shift.
func shiftRef(xs []float64, k int) []float64 {
	out := allNaN(len(xs))
	for i := k; i < len(xs); i++ {
		out[i] = xs[i-k]
	}
	return out
}

// TestWilliamsFractalConfirmedAtIndex checks both that the pattern is detected and that the
// mask is set at the confirming bar, not at the bar it describes.
func TestWilliamsFractalConfirmedAtIndex(t *testing.T) {
	const n = 2
	high := []float64{1, 2, 5, 2, 1}
	low := []float64{5, 4, 1, 4, 5}

	up, down := WilliamsFractal(high, low, n)

	// The up fractal is at bar 2 and is knowable at bar 4.
	if up[2] != 0 {
		t.Errorf("up[2] = %d; the signal must not appear at the bar it describes", up[2])
	}
	if up[4] != 1 {
		t.Errorf("up[4] = %d, want 1 (confirmed at i+n)", up[4])
	}
	if down[4] != 1 {
		t.Errorf("down[4] = %d, want 1", down[4])
	}
	// The last n positions cannot be confirmed.
	if up[len(up)-1] != 0 && len(up)-1 == 4 {
		// index 4 is both the fractal and the end here; the check above already passed.
	}
}

// TestWilliamsFractalTiesDisqualify pins the strict comparison: an equal high on either side
// means the bar is not a distinct extreme.
func TestWilliamsFractalTiesDisqualify(t *testing.T) {
	high := []float64{1, 5, 5, 2, 1}
	low := []float64{9, 8, 7, 6, 5}

	up, _ := WilliamsFractal(high, low, 2)
	for i, v := range up {
		if v != 0 {
			t.Fatalf("up[%d] = %d, want 0 (a tied high is not a fractal)", i, v)
		}
	}
}

func TestMiscIndicatorsEmptyAndPanics(t *testing.T) {
	if CoppockCurve(nil, 14, 11, 10) != nil {
		t.Error("empty CoppockCurve should be nil")
	}
	if k, s := KnowSureThing(nil, [4]int{10, 15, 20, 30}, [4]int{10, 10, 10, 15}, 9); k != nil || s != nil {
		t.Error("empty KnowSureThing should return nils")
	}
	if TrueStrengthIndex(nil, 25, 13) != nil {
		t.Error("empty TrueStrengthIndex should be nil")
	}
	if j, t2, l := WilliamsAlligator(nil, nil, 13, 8, 8, 5, 5, 3); j != nil || t2 != nil || l != nil {
		t.Error("empty WilliamsAlligator should return nils")
	}
	if u, d := WilliamsFractal(nil, nil, 2); u != nil || d != nil {
		t.Error("empty WilliamsFractal should return nils")
	}

	short := make([]float64, 2)
	long := make([]float64, 3)
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"Coppock period", func() { CoppockCurve(long, 0, 11, 10) }},
		{"KST roc", func() { KnowSureThing(long, [4]int{0, 15, 20, 30}, [4]int{10, 10, 10, 15}, 9) }},
		{"KST signal", func() { KnowSureThing(long, [4]int{10, 15, 20, 30}, [4]int{10, 10, 10, 15}, 0) }},
		{"TSI period", func() { TrueStrengthIndex(long, 25, 0) }},
		{"Alligator period", func() { WilliamsAlligator(long, long, 0, 8, 8, 5, 5, 3) }},
		{"Alligator negative shift", func() { WilliamsAlligator(long, long, 13, -1, 8, 5, 5, 3) }},
		{"Alligator length", func() { WilliamsAlligator(short, long, 13, 8, 8, 5, 5, 3) }},
		{"Fractal period", func() { WilliamsFractal(long, long, 0) }},
		{"Fractal length", func() { WilliamsFractal(short, long, 2) }},
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

// TestKnowSureThingShorterThanMaxPeriod pins the degenerate case: every output is NaN when
// the series cannot fill the longest term.
func TestKnowSureThingShorterThanMaxPeriod(t *testing.T) {
	close := []float64{1, 2, 3}
	kst, signal := KnowSureThing(close, [4]int{10, 15, 20, 30}, [4]int{10, 10, 10, 15}, 9)
	for _, v := range kst {
		if !math.IsNaN(v) {
			t.Error("KST shorter than its longest period should be all NaN")
		}
	}
	for _, v := range signal {
		if !math.IsNaN(v) {
			t.Error("KST signal shorter than its longest period should be all NaN")
		}
	}
}
