package ta

// ---------------------------------------------------------------------------
// Envelopes, channels and the price oscillator.
//
// These wrap a moving average rather than being one: an envelope offsets it by a percentage, a
// channel replaces it with the averages of the high and the low, and the price oscillator
// differences two of them. Grouping them here keeps the moving averages in moving.go about
// smoothing and makes it visible that these are all one step removed.
// ---------------------------------------------------------------------------

// MovingAverageEnvelope returns a percentage envelope around a simple moving average:
//
//	middle = SMA(xs, n)
//	upper  = middle * (1 + percent)
//	lower  = middle * (1 - percent)
//
// percent is a *fraction*, so 0.05 is five percent; passing 5 would produce a band five times the
// average wide, which is why the unit is stated on the function rather than left to the caller.
//
// The band is constant relative width, which is what distinguishes an envelope from a Bollinger
// band: an envelope assumes volatility is proportional to price and never changes, a Bollinger band
// measures it. An envelope is therefore a statement about where price should be, and a Bollinger
// band a statement about where it has been.
//
// percent may be zero, which collapses the band onto the average, but not negative, which would
// swap the two sides. The first n-1 values are NaN.
func MovingAverageEnvelope(xs []float64, n int, percent float64) (upper, middle, lower []float64) {
	checkPeriod("MovingAverageEnvelope", n)
	if percent < 0 || percent != percent {
		panic("ta: MovingAverageEnvelope percent must be >= 0")
	}
	if len(xs) == 0 {
		return nil, nil, nil
	}
	size := len(xs)
	middle = SMA(xs, n)
	upper = make([]float64, size)
	lower = make([]float64, size)
	for i := 0; i < size; i++ {
		m := middle[i]
		upper[i] = m * (1 + percent)
		lower[i] = m * (1 - percent)
	}
	return upper, middle, lower
}

// MovingAverageChannel returns the channel formed by simple moving averages of the high and the low:
//
//	upper = SMA(high, n)
//	lower = SMA(low, n)
//
// It is the smooth counterpart of a Donchian channel: where a Donchian channel spans the extremes
// actually reached, this spans where the two sides of the range have *averaged*, so it is narrower
// and turns earlier but never contains the extremes. A caller wanting a breakout level wants
// Donchian; a caller wanting a trend channel wants this.
//
// The first n-1 values of both outputs are NaN.
func MovingAverageChannel(high, low []float64, n int) (upper, lower []float64) {
	checkPeriod("MovingAverageChannel", n)
	size := requireSameLen("MovingAverageChannel", high, low)
	if size == 0 {
		return nil, nil
	}
	return SMA(high, n), SMA(low, n)
}

// PriceOscillator returns the percentage difference between two simple moving averages:
//
//	100 * (SMA(xs, fast) - SMA(xs, slow)) / SMA(xs, slow)
//
// Differencing two averages of the same series removes the level and leaves the trend, and dividing
// by the slow average makes the result comparable across instruments -- which is the whole reason to
// prefer this over the raw MACD line, whose scale is in the instrument's own units.
//
// fast must be less than slow; a caller who wants the reverse is looking for the negated series.
// A zero slow average yields 0, following the package's division convention. The first slow-1 values
// are NaN.
func PriceOscillator(xs []float64, fast, slow int) []float64 {
	checkPeriod("PriceOscillator fast", fast)
	checkPeriod("PriceOscillator slow", slow)
	if fast >= slow {
		panic("ta: PriceOscillator fast period must be less than slow period")
	}
	if len(xs) == 0 {
		return nil
	}
	fastMA := SMA(xs, fast)
	slowMA := SMA(xs, slow)
	out := allNaN(len(xs))
	for i := 0; i < len(xs); i++ {
		s := slowMA[i]
		if s != s || fastMA[i] != fastMA[i] {
			continue
		}
		out[i] = 100 * (fastMA[i] - s) / s
	}
	return out
}
