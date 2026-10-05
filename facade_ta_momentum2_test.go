package numa

import (
	"math"
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the second-tier momentum, volume, stop and range indicators.
//
// Several of these take three or four periods in a row, which is the signature shape where a
// transposition cannot be seen. Each is driven with all-distinct values and compared against ta
// directly, and the multi-period ones are additionally checked to be sensitive to each argument.
// ---------------------------------------------------------------------------

func TestMomentum2ForwardersMatchTa(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	eqSeries(t, "PercentRank", PercentRank(close, 20), ta.PercentRank(close, 20))
	eqSeries(t, "RelativeVolatilityIndex", RelativeVolatilityIndex(close, 14), ta.RelativeVolatilityIndex(close, 14))
	eqSeries(t, "ConnorsRSI", ConnorsRSI(close, 3, 2, 50), ta.ConnorsRSI(close, 3, 2, 50))

	gs, gg := StochasticMomentumIndex(high, low, close, 10, 3, 3)
	ws, wg := ta.StochasticMomentumIndex(high, low, close, 10, 3, 3)
	eqSeries(t, "SMI", gs, ws)
	eqSeries(t, "SMI signal", gg, wg)

	gb, gr := ElderRay(high, low, close, 13)
	wb, wr := ta.ElderRay(high, low, close, 13)
	eqSeries(t, "ElderRay bull", gb, wb)
	eqSeries(t, "ElderRay bear", gr, wr)
}

// TestConnorsRSIForwarderKeepsParameterOrder uses three distinct periods and checks each one reaches
// the implementation.
func TestConnorsRSIForwarderKeepsParameterOrder(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	base := ConnorsRSI(close, 3, 2, 50)
	if eqSeriesEqual(base, ConnorsRSI(close, 5, 2, 50)) {
		t.Error("ConnorsRSI ignores rsiPeriod")
	}
	if eqSeriesEqual(base, ConnorsRSI(close, 3, 4, 50)) {
		t.Error("ConnorsRSI ignores streakPeriod")
	}
	if eqSeriesEqual(base, ConnorsRSI(close, 3, 2, 20)) {
		t.Error("ConnorsRSI ignores rankPeriod")
	}
}

func TestVolume2ForwardersMatchTa(t *testing.T) {
	open, high, low, close, volume := testOHLCV()

	eqSeries(t, "BalanceOfPower", BalanceOfPower(open, high, low, close, 14),
		ta.BalanceOfPower(open, high, low, close, 14))
	eqSeries(t, "EaseOfMovement", EaseOfMovement(high, low, volume, 14),
		ta.EaseOfMovement(high, low, volume, 14))
	eqSeries(t, "ChaikinVolatility", ChaikinVolatility(high, low, 10),
		ta.ChaikinVolatility(high, low, 10))

	gk, gs := KlingerOscillator(high, low, close, volume, 34, 55, 13)
	wk, ws := ta.KlingerOscillator(high, low, close, volume, 34, 55, 13)
	eqSeries(t, "KVO", gk, wk)
	eqSeries(t, "KVO signal", gs, ws)

	gl, gsh := ChandeKrollStop(high, low, close, 10, 1, 9)
	wl, wsh := ta.ChandeKrollStop(high, low, close, 10, 1, 9)
	eqSeries(t, "ChandeKrollStop long", gl, wl)
	eqSeries(t, "ChandeKrollStop short", gsh, wsh)

	gh, glo := FiftyTwoWeekHighLow(high, low, 100)
	wh, wlo := ta.FiftyTwoWeekHighLow(high, low, 100)
	eqSeries(t, "52w high", gh, wh)
	eqSeries(t, "52w low", glo, wlo)
}

// TestSMIForwarderKeepsParameterOrder drives the three SMI periods apart.
func TestSMIForwarderKeepsParameterOrder(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	base, _ := StochasticMomentumIndex(high, low, close, 10, 3, 3)
	if a, _ := StochasticMomentumIndex(high, low, close, 14, 3, 3); eqSeriesEqual(base, a) {
		t.Error("SMI ignores its length")
	}
	if a, _ := StochasticMomentumIndex(high, low, close, 10, 5, 3); eqSeriesEqual(base, a) {
		t.Error("SMI ignores smoothK")
	}
	_, sig3 := StochasticMomentumIndex(high, low, close, 10, 3, 3)
	_, sig5 := StochasticMomentumIndex(high, low, close, 10, 3, 5)
	if eqSeriesEqual(sig3, sig5) {
		t.Error("SMI ignores smoothD")
	}
}

// TestKlingerForwarderKeepsParameterOrder drives the three Klinger periods apart.
func TestKlingerForwarderKeepsParameterOrder(t *testing.T) {
	_, high, low, close, volume := testOHLCV()

	base, _ := KlingerOscillator(high, low, close, volume, 34, 55, 13)
	if a, _ := KlingerOscillator(high, low, close, volume, 20, 55, 13); eqSeriesEqual(base, a) {
		t.Error("KlingerOscillator ignores fast")
	}
	if a, _ := KlingerOscillator(high, low, close, volume, 34, 40, 13); eqSeriesEqual(base, a) {
		t.Error("KlingerOscillator ignores slow")
	}
	_, s13 := KlingerOscillator(high, low, close, volume, 34, 55, 13)
	_, s5 := KlingerOscillator(high, low, close, volume, 34, 55, 5)
	if eqSeriesEqual(s13, s5) {
		t.Error("KlingerOscillator ignores signalPeriod")
	}
}

// TestChandeKrollStopForwarderKeepsParameterOrder separates the period, the multiplier and the
// smoothing window, which are three different kinds of number in a row.
func TestChandeKrollStopForwarderKeepsParameterOrder(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	base, _ := ChandeKrollStop(high, low, close, 10, 1, 9)
	if a, _ := ChandeKrollStop(high, low, close, 20, 1, 9); eqSeriesEqual(base, a) {
		t.Error("ChandeKrollStop ignores n")
	}
	if a, _ := ChandeKrollStop(high, low, close, 10, 3, 9); eqSeriesEqual(base, a) {
		t.Error("ChandeKrollStop ignores x")
	}
	if a, _ := ChandeKrollStop(high, low, close, 10, 1, 5); eqSeriesEqual(base, a) {
		t.Error("ChandeKrollStop ignores q")
	}
}

func TestMomentum2ForwardersPanicLikeTa(t *testing.T) {
	_, high, low, close, volume := testOHLCV()
	open := make([]float64, len(close))
	short := make([]float64, 2)
	long := make([]float64, 3)

	cases := []struct {
		name string
		fn   func()
	}{
		{"PercentRank period", func() { PercentRank(close, 0) }},
		{"SMI period", func() { StochasticMomentumIndex(high, low, close, 0, 3, 3) }},
		{"SMI length", func() { StochasticMomentumIndex(short, short, long, 10, 3, 3) }},
		{"RVI period", func() { RelativeVolatilityIndex(close, 0) }},
		{"ConnorsRSI period", func() { ConnorsRSI(close, 3, 0, 50) }},
		{"ElderRay period", func() { ElderRay(high, low, close, 0) }},
		{"BOP length", func() { BalanceOfPower(short, high, low, long, 5) }},
		{"EOM period", func() { EaseOfMovement(high, low, volume, 0) }},
		{"KVO reversed", func() { KlingerOscillator(high, low, close, volume, 55, 34, 13) }},
		{"KVO signal", func() { KlingerOscillator(high, low, close, volume, 34, 55, 0) }},
		{"Chaikin period", func() { ChaikinVolatility(high, low, 0) }},
		{"CKS negative x", func() { ChandeKrollStop(high, low, close, 10, -1, 9) }},
		{"CKS length", func() { ChandeKrollStop(short, short, long, 10, 1, 9) }},
		{"52w period", func() { FiftyTwoWeekHighLow(high, low, 0) }},
	}
	for _, tc := range cases {
		fn := tc.fn
		_ = open
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", tc.name)
				}
			}()
			fn()
		})
	}
}

func TestMomentum2ForwarderRanges(t *testing.T) {
	_, high, low, close, _ := testOHLCV()

	for i, v := range PercentRank(close, 20) {
		if math.IsNaN(v) {
			continue
		}
		if v < 0 || v >= 100 {
			t.Fatalf("PercentRank[%d] = %v, outside [0,100)", i, v)
		}
	}
	for i, v := range ConnorsRSI(close, 3, 2, 50) {
		if math.IsNaN(v) {
			continue
		}
		if v < 0 || v > 100 {
			t.Fatalf("ConnorsRSI[%d] = %v, outside [0,100]", i, v)
		}
	}
	smi, _ := StochasticMomentumIndex(high, low, close, 10, 3, 3)
	for i, v := range smi {
		if math.IsNaN(v) {
			continue
		}
		if v < -100-1e-9 || v > 100+1e-9 {
			t.Fatalf("SMI[%d] = %v, outside [-100,100]", i, v)
		}
	}
}

func TestMomentum2ForwardersEmptyInputs(t *testing.T) {
	if PercentRank(nil, 5) != nil || RelativeVolatilityIndex(nil, 5) != nil ||
		ConnorsRSI(nil, 3, 2, 50) != nil {
		t.Error("empty momentum inputs should return nil")
	}
	if s, g := StochasticMomentumIndex(nil, nil, nil, 10, 3, 3); s != nil || g != nil {
		t.Error("empty SMI should return nils")
	}
	if b, r := ElderRay(nil, nil, nil, 13); b != nil || r != nil {
		t.Error("empty ElderRay should return nils")
	}
	if k, s := KlingerOscillator(nil, nil, nil, nil, 34, 55, 13); k != nil || s != nil {
		t.Error("empty KlingerOscillator should return nils")
	}
	if l, s := ChandeKrollStop(nil, nil, nil, 10, 1, 9); l != nil || s != nil {
		t.Error("empty ChandeKrollStop should return nils")
	}
}
