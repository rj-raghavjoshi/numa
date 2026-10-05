package ta

// ---------------------------------------------------------------------------
// Trailing-stop indicators.
//
// A stop level is not a smoothed average or an oscillator: it is a rule for where to exit, and the
// rule has a direction. That is why these return two series rather than one, and why the doc comments
// state which side is for which position -- getting that backwards inverts the indicator without
// changing any number in it.
// ---------------------------------------------------------------------------

// ChandeKrollStop returns the two Chande Kroll stops:
//
//	atr            = ATR(high, low, close, n)
//	firstStopLong  = highest(high, n) - x*atr
//	firstStopShort = lowest(low, n) + x*atr
//	stopLong       = highest(firstStopLong, q)
//	stopShort      = lowest(firstStopShort, q)
//
// The first pair places the stops a multiple of the average true range inside the n-bar extreme,
// which is a volatility-scaled version of the classic extreme-minus-a-fixed-distance rule. The
// second pair applies the direction: the long stop is the *highest* of the last q first-pass values
// and the short stop the lowest, so each is pulled towards the side a stop should be on rather than
// following the first pass directly.
//
// # The second pass smooths; it does not guarantee monotonicity
//
// Because it is a rolling extreme over a window, the long stop *can* step down when the highest
// first-pass value leaves that window. It is a common belief that a trailing stop only ever moves in
// the favourable direction, and this formulation does not promise that; a caller who needs a stop
// that literally never loosens has to carry it forward against the previous bar, which is a
// different indicator and is not what Chande and Kroll published. The guaranteed property, and the
// one the test asserts, is that the long stop is never below the current first-pass value.
//
// stopLong is the level for a long position and sits below price; stopShort is for a short and sits
// above. x is a multiple of the ATR and must be non-negative; TradingView's defaults are n = 10,
// x = 1 and q = 9. The first n+q-2 values are NaN.
func ChandeKrollStop(high, low, close []float64, n int, x float64, q int) (stopLong, stopShort []float64) {
	checkPeriod("ChandeKrollStop", n)
	checkPeriod("ChandeKrollStop q", q)
	if x < 0 || x != x {
		panic("ta: ChandeKrollStop multiplier must be >= 0")
	}
	size := requireSameLen("ChandeKrollStop", high, low, close)
	if size == 0 {
		return nil, nil
	}

	atr := ATR(high, low, close, n)
	hi := Highest(high, n)
	lo := Lowest(low, n)

	firstLong := allNaN(size)
	firstShort := allNaN(size)
	for i := 0; i < size; i++ {
		if atr[i] != atr[i] || hi[i] != hi[i] || lo[i] != lo[i] {
			continue
		}
		firstLong[i] = hi[i] - x*atr[i]
		firstShort[i] = lo[i] + x*atr[i]
	}

	// The second pass is what makes the stops trail: a maximum for the long side and a minimum for
	// the short, so neither can loosen.
	return Highest(firstLong, q), Lowest(firstShort, q)
}
