package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for correlation, volatility statistics and the long-period
// oscillators.
//
// Two of these have signatures where a transposition is particularly hard to see:
// HistoricalVolatility takes (period, annual) -- two numbers that are both plausible in
// either role -- and WilliamsAlligator takes six ints in three (period, shift) pairs.
// Both are driven with all-distinct values and compared against ta directly.
// ---------------------------------------------------------------------------

func TestStatisticsForwardersMatchTa(t *testing.T) {
	open, _, _, close, _ := testOHLCV()
	_, high, low, _, _ := testOHLCV()
	other := make([]float64, len(close))
	for i := range close {
		other[i] = close[i]*1.5 + float64(i)
	}

	eqSeries(t, "CorrelationCoefficient", CorrelationCoefficient(close, other, 14),
		ta.CorrelationCoefficient(close, other, 14))
	eqSeries(t, "CorrelationLog", CorrelationLog(close, other, 14), ta.CorrelationLog(close, other, 14))
	eqSeries(t, "RankCorrelation", RankCorrelation(close, other, 14), ta.RankCorrelation(close, other, 14))
	eqSeries(t, "HistoricalVolatility", HistoricalVolatility(close, 20, 252),
		ta.HistoricalVolatility(close, 20, 252))
	eqSeries(t, "VolatilityOHLC", VolatilityOHLC(open, high, low, close, 20, 252),
		ta.VolatilityOHLC(open, high, low, close, 20, 252))
}

func TestStatisticsForwarderAnnualization(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	base := HistoricalVolatility(close, 20, 100)
	quad := HistoricalVolatility(close, 20, 400)
	for i := 20; i < len(close); i++ {
		if math.Abs(quad[i]-2*base[i]) > 1e-9 {
			t.Fatalf("annualization[%d]: %v at 400 != 2x %v at 100", i, quad[i], base[i])
		}
	}
}

func TestAlligatorForwarderKeepsParameterOrder(t *testing.T) {
	_, high, low, _, _ := testOHLCV()

	jaw, teeth, lips := WilliamsAlligator(high, low, 13, 8, 8, 5, 5, 3)
	wj, wt, wl := ta.WilliamsAlligator(high, low, 13, 8, 8, 5, 5, 3)
	eqSeries(t, "Alligator jaw", jaw, wj)
	eqSeries(t, "Alligator teeth", teeth, wt)
	eqSeries(t, "Alligator lips", lips, wl)

	// Swapping the period and shift of the jaw must change the line.
	swapped, _, _ := WilliamsAlligator(high, low, 8, 13, 8, 5, 5, 3)
	if eqSeriesEqual(jaw, swapped) {
		t.Error("WilliamsAlligator is insensitive to exchanging the jaw period and shift")
	}
}

func TestFractalForwarderPlacesSignalAtConfirmation(t *testing.T) {
	high := []float64{1, 2, 5, 2, 1}
	low := []float64{5, 4, 1, 4, 5}

	up, down := WilliamsFractal(high, low, 2)
	if up[2] != 0 {
		t.Errorf("up[2] = %d; the facade must not move the signal to the bar it describes", up[2])
	}
	if up[4] != 1 || down[4] != 1 {
		t.Errorf("up[4], down[4] = %d, %d; want 1, 1", up[4], down[4])
	}
}

func TestOscillatorStatForwardersMatchTa(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	eqSeries(t, "CoppockCurve", CoppockCurve(close, 14, 11, 10), ta.CoppockCurve(close, 14, 11, 10))
	eqSeries(t, "TrueStrengthIndex", TrueStrengthIndex(close, 25, 13), ta.TrueStrengthIndex(close, 25, 13))

	roc := [4]int{10, 15, 20, 30}
	sma := [4]int{10, 10, 10, 15}
	gk, gs := KnowSureThing(close, roc, sma, 9)
	wk, ws := ta.KnowSureThing(close, roc, sma, 9)
	eqSeries(t, "KST", gk, wk)
	eqSeries(t, "KST signal", gs, ws)
}

func TestStatisticsForwardersPanicLikeTa(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"CorrelationCoefficient period", func() { CorrelationCoefficient(long, long, 0) }},
		{"CorrelationLog length", func() { CorrelationLog(short, long, 5) }},
		{"RankCorrelation period", func() { RankCorrelation(long, long, 0) }},
		{"HistoricalVolatility period", func() { HistoricalVolatility(long, 0, 252) }},
		{"VolatilityOHLC period", func() { VolatilityOHLC(long, long, long, long, 0, 252) }},
		{"CoppockCurve period", func() { CoppockCurve(long, 14, 11, 0) }},
		{"KnowSureThing period", func() { KnowSureThing(long, [4]int{0, 15, 20, 30}, [4]int{10, 10, 10, 15}, 9) }},
		{"TrueStrengthIndex period", func() { TrueStrengthIndex(long, 0, 13) }},
		{"WilliamsAlligator period", func() { WilliamsAlligator(long, long, 0, 8, 8, 5, 5, 3) }},
		{"WilliamsFractal period", func() { WilliamsFractal(long, long, 0) }},
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

func TestStatisticsForwardersEmptyInputs(t *testing.T) {
	if CorrelationCoefficient(nil, nil, 10) != nil || CorrelationLog(nil, nil, 10) != nil ||
		RankCorrelation(nil, nil, 10) != nil || HistoricalVolatility(nil, 10, 252) != nil {
		t.Error("empty statistical inputs should return nil")
	}
	if CoppockCurve(nil, 14, 11, 10) != nil || TrueStrengthIndex(nil, 25, 13) != nil {
		t.Error("empty oscillators should return nil")
	}
}
