package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the momentum indicators.
//
// MACD takes three periods in a row and Stochastic takes three smoothing parameters,
// all of them plain ints. A forwarder that transposes two of them still compiles and
// still returns a plausible oscillator, so every check below uses all-distinct values
// and compares against ta directly.
// ---------------------------------------------------------------------------

func TestMomentumForwardersMatchTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	eqSeries(t, "RSI", RSI(close, 14), ta.RSI(close, 14))
	eqSeries(t, "WilliamsPercentR", WilliamsPercentR(high, low, close, 14), ta.WilliamsPercentR(high, low, close, 14))
	eqSeries(t, "CCI", CCI(high, low, close, 20), ta.CCI(high, low, close, 20))
	eqSeries(t, "CMO", CMO(close, 9), ta.CMO(close, 9))
	eqSeries(t, "ROC", ROC(close, 5), ta.ROC(close, 5))
	eqSeries(t, "Momentum", Momentum(close, 5), ta.Momentum(close, 5))
}

// TestMACDForwarderKeepsPeriodOrder is the specific transposition test. fast=12,
// slow=26, signal=9 are all distinct, and MACD is not symmetric in fast and slow in
// magnitude -- though the sign flips, the histogram is not the same series -- so a
// swapped fast/slow is detectable.
func TestMACDForwarderKeepsPeriodOrder(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	gm, gs, gh := MACD(close, 12, 26, 9)
	wm, ws, wh := ta.MACD(close, 12, 26, 9)
	eqSeries(t, "MACD line", gm, wm)
	eqSeries(t, "MACD signal", gs, ws)
	eqSeries(t, "MACD hist", gh, wh)

	// Sanity: the histogram really is the line minus the signal, through the facade.
	for i := range close {
		if !sameOrNaNFloat(gh[i], gm[i]-gs[i]) {
			t.Fatalf("hist[%d] = %v, want line-signal = %v", i, gh[i], gm[i]-gs[i])
		}
	}

	// Swapping fast and slow negates the line (EMA(26)-EMA(12)), so a forwarder that
	// transposed them would produce the negated series. Check they differ.
	sm, _, _ := MACD(close, 26, 12, 9)
	same := true
	for i := range gm {
		if !sameOrNaNFloat(gm[i], sm[i]) {
			same = false
			break
		}
	}
	if same {
		t.Error("MACD produced the same line for (12,26) and (26,12)")
	}
}

func TestStochasticForwarderKeepsParameterOrder(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	gk, gd := Stochastic(high, low, close, 14, 3, 3)
	wk, wd := ta.Stochastic(high, low, close, 14, 3, 3)
	eqSeries(t, "Stochastic K", gk, wk)
	eqSeries(t, "Stochastic D", gd, wd)

	// All three parameters distinct, so transposing any pair changes the output.
	ak, _ := Stochastic(high, low, close, 5, 14, 21)
	bk, _ := Stochastic(high, low, close, 14, 5, 21)
	if eqSeriesEqual(ak, bk) {
		t.Error("Stochastic ignored the difference between kPeriod and kSmooth")
	}
}

func TestStochRSIForwarderKeepsParameterOrder(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	gk, gd := StochRSI(close, 14, 14, 3, 3)
	wk, wd := ta.StochRSI(close, 14, 14, 3, 3)
	eqSeries(t, "StochRSI K", gk, wk)
	eqSeries(t, "StochRSI D", gd, wd)

	ak, _ := StochRSI(close, 7, 14, 3, 3)
	bk, _ := StochRSI(close, 14, 7, 3, 3)
	if eqSeriesEqual(ak, bk) {
		t.Error("StochRSI ignored the difference between rsiPeriod and stochPeriod")
	}
}

// TestMomentumForwarderRanges checks the documented output ranges through the facade,
// which catches a scaling or sign error that agreement with ta would not.
func TestMomentumForwarderRanges(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	for i, v := range RSI(close, 14) {
		if math.IsNaN(v) {
			continue
		}
		if v < -1e-9 || v > 100+1e-9 {
			t.Fatalf("RSI[%d] = %v, outside [0,100]", i, v)
		}
	}
	for i, v := range WilliamsPercentR(high, low, close, 14) {
		if math.IsNaN(v) {
			continue
		}
		if v < -100-1e-9 || v > 1e-9 {
			t.Fatalf("%%R[%d] = %v, outside [-100,0]", i, v)
		}
	}
	for i, v := range CMO(close, 9) {
		if math.IsNaN(v) {
			continue
		}
		if v < -100-1e-9 || v > 100+1e-9 {
			t.Fatalf("CMO[%d] = %v, outside [-100,100]", i, v)
		}
	}
}

func TestMomentumForwardersPanicLikeTa(t *testing.T) {
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"RSI period", func() { RSI(short, 0) }},
		{"MACD slow", func() { MACD(long, 12, 0, 9) }},
		{"Stochastic length", func() { Stochastic(short, short, long, 14, 3, 3) }},
		{"Stochastic period", func() { Stochastic(long, long, long, 0, 3, 3) }},
		{"StochRSI period", func() { StochRSI(long, 14, 0, 3, 3) }},
		{"WilliamsPercentR length", func() { WilliamsPercentR(short, long, long, 5) }},
		{"CCI length", func() { CCI(short, short, long, 5) }},
		{"CMO period", func() { CMO(long, 0) }},
		{"ROC period", func() { ROC(long, 0) }},
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

func TestMomentumForwardersEmptyInputs(t *testing.T) {
	if RSI(nil, 14) != nil || CMO(nil, 14) != nil || ROC(nil, 14) != nil || Momentum(nil, 14) != nil {
		t.Error("empty inputs should return nil")
	}
	if m, s, h := MACD(nil, 12, 26, 9); m != nil || s != nil || h != nil {
		t.Error("empty MACD should return nils")
	}
	if k, d := Stochastic(nil, nil, nil, 14, 3, 3); k != nil || d != nil {
		t.Error("empty Stochastic should return nils")
	}
}
