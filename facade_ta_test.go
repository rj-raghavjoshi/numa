package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the ta facade.
//
// Indicator signatures put several ints and floats in a row, so a forwarder that
// transposes two of them still compiles and still returns a plausible-looking series.
// Every check below therefore compares against ta with identical arguments, and the
// deliberately asymmetric cases (fast != slow, n != k) make a transposition visible
// rather than merely possible.
// ---------------------------------------------------------------------------

func testOHLCV() (open, high, low, close, volume []float64) {
	n := 60
	open = make([]float64, n)
	high = make([]float64, n)
	low = make([]float64, n)
	close = make([]float64, n)
	volume = make([]float64, n)
	p := 50.0
	for i := 0; i < n; i++ {
		open[i] = p
		p += float64((i%7)-3) * 0.5
		close[i] = p
		high[i] = math.Max(open[i], close[i]) + 0.75
		low[i] = math.Min(open[i], close[i]) - 0.75
		volume[i] = 1000 + float64(i*13%97)
	}
	return
}

func eqSeries(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
	}
	for i := range want {
		if sameOrNaNFloat(got[i], want[i]) {
			continue
		}
		t.Fatalf("%s[%d] = %v, want %v", name, i, got[i], want[i])
	}
}

func sameOrNaNFloat(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return a == b
}

func TestPriceForwardersMatchTa(t *testing.T) {
	open, high, low, close, _ := testOHLCV()

	eqSeries(t, "MedianPrice", MedianPrice(high, low), ta.MedianPrice(high, low))
	eqSeries(t, "TypicalPrice", TypicalPrice(high, low, close), ta.TypicalPrice(high, low, close))
	eqSeries(t, "AveragePrice", AveragePrice(open, high, low, close), ta.AveragePrice(open, high, low, close))
	eqSeries(t, "TrueRange", TrueRange(high, low, close), ta.TrueRange(high, low, close))
	eqSeries(t, "Change", Change(close, 3), ta.Change(close, 3))

	if got, want := FirstValid([]float64{math.NaN(), 1}), ta.FirstValid([]float64{math.NaN(), 1}); got != want {
		t.Errorf("FirstValid = %d, want %d", got, want)
	}
}

func TestMovingAverageForwardersMatchTa(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	eqSeries(t, "SMA", SMA(close, 9), ta.SMA(close, 9))
	eqSeries(t, "EMA", EMA(close, 9), ta.EMA(close, 9))
	eqSeries(t, "RMA", RMA(close, 9), ta.RMA(close, 9))
	eqSeries(t, "WMA", WMA(close, 9), ta.WMA(close, 9))
	eqSeries(t, "DEMA", DEMA(close, 9), ta.DEMA(close, 9))
	eqSeries(t, "TEMA", TEMA(close, 9), ta.TEMA(close, 9))
	eqSeries(t, "TRIMA", TRIMA(close, 9), ta.TRIMA(close, 9))
	eqSeries(t, "HMA", HMA(close, 9), ta.HMA(close, 9))

	// The period argument must actually reach the implementation.
	if eqSeriesEqual(SMA(close, 5), SMA(close, 9)) {
		t.Error("SMA ignores its period argument")
	}
}

func eqSeriesEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameOrNaNFloat(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestExtremeAndChannelForwardersMatchTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	eqSeries(t, "Highest", Highest(high, 14), ta.Highest(high, 14))
	eqSeries(t, "Lowest", Lowest(low, 14), ta.Lowest(low, 14))

	gu, gm, gl := Donchian(high, low, 14)
	wu, wm, wl := ta.Donchian(high, low, 14)
	eqSeries(t, "Donchian upper", gu, wu)
	eqSeries(t, "Donchian middle", gm, wm)
	eqSeries(t, "Donchian lower", gl, wl)

	eqSeries(t, "ATR", ATR(high, low, close, 14), ta.ATR(high, low, close, 14))
}

// TestBollingerForwardersKeepArgumentOrder is the specific test for the (n, k)
// ordering: n=10, k=3 and n=3, k=10 are both legal and produce entirely different
// bands, so a transposed forwarder cannot pass silently.
func TestBollingerForwardersKeepArgumentOrder(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	gu, gm, gl := BollingerBands(close, 10, 3)
	wu, wm, wl := ta.BollingerBands(close, 10, 3)
	eqSeries(t, "Bollinger upper", gu, wu)
	eqSeries(t, "Bollinger middle", gm, wm)
	eqSeries(t, "Bollinger lower", gl, wl)

	eqSeries(t, "BBPercentB", BBPercentB(close, 10, 3), ta.BBPercentB(close, 10, 3))
	eqSeries(t, "BBWidth", BBWidth(close, 10, 3), ta.BBWidth(close, 10, 3))

	// Transposing n and k must produce a different result, or this test proves
	// nothing.
	su, _, _ := BollingerBands(close, 3, 10)
	if eqSeriesEqual(gu, su) {
		t.Error("BollingerBands produced the same bands for (n=10,k=3) and (n=3,k=10)")
	}
}

func TestTaForwardersPanicLikeTa(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"MedianPrice", func() { MedianPrice(short, long) }},
		{"TypicalPrice", func() { TypicalPrice(short, short, long) }},
		{"AveragePrice", func() { AveragePrice(short, short, short, long) }},
		{"TrueRange", func() { TrueRange(short, short, long) }},
		{"SMA period", func() { SMA(short, 0) }},
		{"ATR period", func() { ATR(short, short, short, 0) }},
		{"Donchian length", func() { Donchian(short, long, 2) }},
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

// TestTaForwarderEmptyInputs checks the nil contract end to end.
func TestTaForwarderEmptyInputs(t *testing.T) {
	if SMA(nil, 5) != nil || EMA(nil, 5) != nil || WMA(nil, 5) != nil {
		t.Error("empty moving averages should be nil")
	}
	if Highest(nil, 5) != nil || Lowest(nil, 5) != nil {
		t.Error("empty extremes should be nil")
	}
	if u, m, l := BollingerBands(nil, 5, 2); u != nil || m != nil || l != nil {
		t.Error("empty BollingerBands should be nils")
	}
}

// TestTaForwardersAgreeWithVecWhereTheyOverlap checks the one place ta and vec compute
// the same quantity, so a change in either is caught.
func TestTaForwardersAgreeWithVecWhereTheyOverlap(t *testing.T) {
	_, _, _, close, _ := testOHLCV()
	n := 10

	// SMA is the mean of the trailing window; vec.Mean over the same window must
	// agree.
	sma := SMA(close, n)
	for i := n - 1; i < len(close); i++ {
		want := vec.Mean(close[i-n+1 : i+1])
		if math.Abs(sma[i]-want) > 1e-9 {
			t.Fatalf("SMA[%d] = %v, vec.Mean of the window = %v", i, sma[i], want)
		}
	}
}
