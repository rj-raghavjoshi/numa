package numa

import (
	"testing"

	"github.com/rj-raghavjoshi/numa/ta"
)

// ---------------------------------------------------------------------------
// Forwarder tests for the catalogue-completion family.
//
// The cross family has three near-identical signatures that differ only in which average each side
// uses, so the tests drive all three and check they produce *different* lines -- a transposition
// between them would otherwise pass every value comparison, because each would agree with its own
// arguments.
// ---------------------------------------------------------------------------

func TestCatalogueForwardersMatchTa(t *testing.T) {
	open, high, low, close, _ := testOHLCV()

	eqSeries(t, "StandardDeviation", StandardDeviation(close, 14), ta.StandardDeviation(close, 14))
	eqSeries(t, "Ratio", Ratio(close, high), ta.Ratio(close, high))
	eqSeries(t, "Spread", Spread(close, low), ta.Spread(close, low))
	eqSeries(t, "ChopZone", ChopZone(high, low, close, 14), ta.ChopZone(high, low, close, 14))
	eqSeries(t, "HammingMA", HammingMA(close, 9), ta.HammingMA(close, 9))

	gr, gs := RelativeVigorIndex(open, high, low, close, 10, 4)
	wr, ws := ta.RelativeVigorIndex(open, high, low, close, 10, 4)
	eqSeries(t, "RVI", gr, wr)
	eqSeries(t, "RVI signal", gs, ws)

	ge, gg, gosc := SMIErgodic(close, 25, 13, 7)
	we, wg, wosc := ta.SMIErgodic(close, 25, 13, 7)
	eqSeries(t, "ergodic", ge, we)
	eqSeries(t, "ergodic signal", gg, wg)
	eqSeries(t, "ergodic oscillator", gosc, wosc)

	eqSeries(t, "AdvanceDecline", AdvanceDecline(close, low), ta.AdvanceDecline(close, low))
}

// TestCrossForwardersDiffer checks that the three cross indicators really use different averages.
func TestCrossForwardersDiffer(t *testing.T) {
	_, _, _, close, _ := testOHLCV()

	smFast, smSlow, _ := MACross(close, 10, 30)
	emFast, emSlow, _ := EMACross(close, 10, 30)
	mixFast, mixSlow, _ := EMASMACross(close, 10, 30)

	// The three differ only in which average each side uses, so the lines that must differ are the
	// ones where the choice differs. EMACross and EMASMACross share a fast EMA by definition, and
	// MACross and EMASMACross share a slow SMA; asserting that all three are pairwise different
	// would be asserting the wrong property.
	if eqSeriesEqual(smFast, emFast) {
		t.Error("MACross and EMACross produced the same fast line, but one is simple and one exponential")
	}
	if eqSeriesEqual(smFast, mixFast) {
		t.Error("MACross and EMASMACross produced the same fast line")
	}
	if !eqSeriesEqual(emFast, mixFast) {
		t.Error("EMACross and EMASMACross should share a fast EMA")
	}
	if eqSeriesEqual(emSlow, mixSlow) {
		t.Error("EMACross and EMASMACross produced the same slow line, but one is exponential and one simple")
	}
	if eqSeriesEqual(smSlow, emSlow) {
		t.Error("MACross and EMACross produced the same slow line")
	}
	if !eqSeriesEqual(smSlow, mixSlow) {
		t.Error("MACross and EMASMACross should share a slow SMA")
	}

	// And each must match the average it claims to use.
	eqSeries(t, "MACross fast is SMA", smFast, SMA(close, 10))
	eqSeries(t, "EMACross fast is EMA", emFast, EMA(close, 10))
	eqSeries(t, "EMACross slow is EMA", emSlow, EMA(close, 30))
	eqSeries(t, "EMASMACross slow is SMA", mixSlow, SMA(close, 30))
}

func TestCrossForwarderSignals(t *testing.T) {
	_, _, _, close, _ := testOHLCV()
	_, _, cross := MACross(close, 10, 30)

	for i, c := range cross {
		if c != 1 && c != -1 && c != 0 {
			t.Fatalf("cross[%d] = %d, want -1, 0 or 1", i, c)
		}
	}
	// The signal must be sparse: a crossing is an event.
	events := 0
	for _, c := range cross {
		if c != 0 {
			events++
		}
	}
	if events == 0 || events > len(close)/4 {
		t.Fatalf("%d crossings over %d bars", events, len(close))
	}
}

func TestCatalogueForwardersPanicLikeTa(t *testing.T) {
	open, high, low, close, _ := synthOHLCVFacade(40)
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
		{"ChopZone period", func() { ChopZone(high, low, close, 0) }},
		{"HammingMA period", func() { HammingMA(close, 0) }},
		{"SMIErgodic signalPeriod", func() { SMIErgodic(close, 25, 13, 0) }},
		{"MACross reversed", func() { MACross(close, 30, 10) }},
		{"EMACross reversed", func() { EMACross(close, 30, 10) }},
		{"EMASMACross reversed", func() { EMASMACross(close, 30, 10) }},
	} {
		tc := tc
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

func TestCatalogueForwardersEmptyInputs(t *testing.T) {
	if StandardDeviation(nil, 5) != nil || Ratio(nil, nil) != nil || Spread(nil, nil) != nil ||
		ChopZone(nil, nil, nil, 5) != nil || HammingMA(nil, 5) != nil ||
		AdvanceDecline(nil, nil) != nil {
		t.Error("empty inputs should return nil")
	}
	if e, s, o := SMIErgodic(nil, 25, 13, 7); e != nil || s != nil || o != nil {
		t.Error("empty SMIErgodic should return nils")
	}
	if r, s := RelativeVigorIndex(nil, nil, nil, nil, 10, 4); r != nil || s != nil {
		t.Error("empty RelativeVigorIndex should return nils")
	}
}

// synthOHLCVFacade builds a valid OHLC series through the facade package's own fixture helper.
func synthOHLCVFacade(n int) (open, high, low, close, volume []float64) {
	open, high, low, close, volume = testOHLCV()
	if n <= len(close) {
		return open[:n], high[:n], low[:n], close[:n], volume[:n]
	}
	return
}
