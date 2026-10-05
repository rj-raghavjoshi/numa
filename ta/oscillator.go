package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/series"
	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Oscillators built from other indicators.
//
// These are the "second-order" indicators: each is a transformation of a moving average,
// a range, or a rate of change that is already defined elsewhere in this package. They
// are grouped here because they share a failure mode -- composing indicators means
// composing their warm-ups, and a boundary that is derived by hand rather than measured
// is exactly where an off-by-one hides.
//
// Every function below therefore builds its inputs through the exported primitives and
// lets [FirstValid] report the resulting warm-up, rather than restating the arithmetic.
// The tests pin the index that follows, so a change in a primitive's warm-up shows up
// here as a failure instead of as a silently shifted series.
// ---------------------------------------------------------------------------

// AwesomeOscillator returns Bill Williams' Awesome Oscillator: the difference between a
// fast and a slow simple moving average of the median price.
//
//	AO = SMA(MedianPrice, fast) - SMA(MedianPrice, slow)
//
// The scale is price units, so it is not comparable across instruments; the sign changes
// and the zero crossings are the signal. TradingView's default is fast = 5, slow = 34.
// The first slow-1 values are NaN.
func AwesomeOscillator(high, low []float64, fast, slow int) []float64 {
	checkPeriod("AwesomeOscillator fast", fast)
	checkPeriod("AwesomeOscillator slow", slow)
	size := requireSameLen("AwesomeOscillator", high, low)
	if size == 0 {
		return nil
	}
	median := MedianPrice(high, low)
	fastMA := SMA(median, fast)
	slowMA := SMA(median, slow)
	out := make([]float64, size)
	for i := range median {
		out[i] = fastMA[i] - slowMA[i]
	}
	return out
}

// AcceleratorOscillator returns the Awesome Oscillator minus its own smoothing:
//
//	AC = AO - SMA(AO, smooth)
//
// It measures the change in the Awesome Oscillator's momentum, so it turns before the AO
// does. TradingView's default smooth is 5 with AO at 5/34, giving a warm-up of
// slow+smooth-2. The first slow+smooth-2 values are NaN.
func AcceleratorOscillator(high, low []float64, fast, slow, smooth int) []float64 {
	checkPeriod("AcceleratorOscillator smooth", smooth)
	size := len(high)
	ao := AwesomeOscillator(high, low, fast, slow)
	if ao == nil {
		return nil
	}
	smoothed := SMA(ao, smooth)
	out := make([]float64, size)
	for i := range ao {
		out[i] = ao[i] - smoothed[i]
	}
	return out
}

// DetrendedPriceOscillator returns the n-period DPO:
//
//	DPO[i] = close[i - (n/2 + 1)] - SMA(close, n)[i]
//
// It removes the trend from price by comparing a past close with the current average,
// which makes cycles easier to see; the shift is what stops the moving average from
// absorbing the cycle it is meant to reveal.
//
// The first max(n-1, n/2+1) values are NaN, because both the average and the shifted
// price must be defined. A shifted close of zero is a real price and is not special-cased.
func DetrendedPriceOscillator(close []float64, n int) []float64 {
	checkPeriod("DetrendedPriceOscillator", n)
	if len(close) == 0 {
		return nil
	}
	ma := SMA(close, n)
	out := allNaN(len(close))
	shift := n/2 + 1
	for i := n - 1; i < len(close); i++ {
		j := i - shift
		if j < 0 {
			continue
		}
		out[i] = close[j] - ma[i]
	}
	return out
}

// TRIX returns the n-period TRIX: the one-bar rate of change of a triple exponential
// moving average, as a percentage.
//
//	Triple = EMA(EMA(EMA(close, n), n), n)
//	TRIX   = 100 * (Triple[i] - Triple[i-1]) / Triple[i-1]
//
// Triple smoothing removes everything but the longest swings, so TRIX crosses zero far
// less often than a raw rate of change and is used as a slow trend filter. The first
// 3n-2 values are NaN: the triple EMA warms up in 3(n-1) bars and the rate of change
// needs one more.
//
// A zero previous value yields 0, following the package's division convention.
func TRIX(close []float64, n int) []float64 {
	checkPeriod("TRIX", n)
	if len(close) == 0 {
		return nil
	}
	e1 := EMA(close, n)
	e2 := applyEMA(e1, FirstValid(e1), n)
	e3 := applyEMA(e2, FirstValid(e2), n)

	out := allNaN(len(close))
	start := FirstValid(e3)
	if start < 0 {
		return out
	}
	for i := start + 1; i < len(close); i++ {
		prev := e3[i-1]
		switch {
		case e3[i] != e3[i] || prev != prev:
			out[i] = math.NaN()
		case prev == 0:
			out[i] = 0
		default:
			out[i] = 100 * (e3[i] - prev) / prev
		}
	}
	return out
}

// UltimateOscillator returns the Ultimate Oscillator over three lookbacks:
//
//	buyingPressure = close - min(low, prevClose)
//	trueRange      = max(high, prevClose) - min(low, prevClose)
//	UO = 100 * (4*avg(short) + 2*avg(mid) + avg(long)) / 7
//
// where avg(k) is the ratio of the summed buying pressure to the summed true range over
// k bars. Weighting three horizons together is the point: a single-period oscillator
// gives the same reading for a brief spike and a sustained move, and this does not.
//
// Note that the true range here is computed against the previous *close* rather than
// being [TrueRange]; it is the same quantity, stated inline because the buying pressure
// needs the same two bounds.
//
// TradingView's defaults are 7, 14 and 28. The first long values are NaN, because each
// ratio needs a previous close and the longest window sets the boundary. A window with
// no true range contributes an average of 0 rather than an infinity.
func UltimateOscillator(high, low, close []float64, short, mid, long int) []float64 {
	checkPeriod("UltimateOscillator short", short)
	checkPeriod("UltimateOscillator mid", mid)
	checkPeriod("UltimateOscillator long", long)
	size := requireSameLen("UltimateOscillator", high, low, close)
	if size == 0 {
		return nil
	}

	bp := make([]float64, size)
	tr := make([]float64, size)
	fillNaN(bp, 0, 1)
	fillNaN(tr, 0, 1)
	for i := 1; i < size; i++ {
		prev := close[i-1]
		lo := math.Min(low[i], prev)
		hi := math.Max(high[i], prev)
		bp[i] = close[i] - lo
		tr[i] = hi - lo
	}

	// Six batch rolling sums over the two materialised series.
	//
	// This was expected to be a large win, because a standalone benchmark put the streaming roller
	// at roughly five times the batch kernel's per-element cost. It measured 39.7 ns/element before
	// the change and 37.6 after: about five percent, not five times. The per-call difference does
	// not accumulate the way it appears to, which is why the numbers in docs/benchmarks.md are the
	// indicators' own and not the sum kernel's. The batch form is kept because it is what the rest
	// of the package uses, not because it was faster here.
	s1, r1 := vec.RollingSum(bp, short), vec.RollingSum(tr, short)
	s2, r2 := vec.RollingSum(bp, mid), vec.RollingSum(tr, mid)
	s3, r3 := vec.RollingSum(bp, long), vec.RollingSum(tr, long)

	out := allNaN(size)
	for i := long; i < size; i++ {
		if s1[i] != s1[i] || r1[i] != r1[i] || s2[i] != s2[i] || r2[i] != r2[i] || s3[i] != s3[i] || r3[i] != r3[i] {
			continue
		}
		a1, a2, a3 := ratioOr0(s1[i], r1[i]), ratioOr0(s2[i], r2[i]), ratioOr0(s3[i], r3[i])
		out[i] = 100 * (4*a1 + 2*a2 + a3) / 7
	}
	return out
}

// ratioOr0 returns num/den, or 0 when the denominator is zero.
func ratioOr0(num, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}

// FisherTransform returns Ehlers' Fisher Transform of the n-bar price range.
//
//	position = (price - lowest(low, n)) / (highest(high, n) - lowest(low, n))
//	value    = 0.66*(position - 0.5) + 0.67*value[1]     clamped to ±0.999
//	fisher   = 0.5*ln((1+value)/(1-value)) + 0.5*fisher[1]
//
// price is the median price. The transform maps a bounded position onto a roughly
// Gaussian distribution, so its extremes stand out more sharply than the raw position's
// would -- which is the point, since it makes turning points easier to call.
//
// # The clamp is load-bearing
//
// `value` is clamped to ±0.999 before the logarithm, because at exactly ±1 the logarithm
// is infinite and the indicator would be destroyed for every later bar. The clamp is part
// of the definition, not a numerical guard.
//
// A window with no range puts the price at the middle of nothing, which is taken as
// position 0.5, so `value` decays towards zero and the line flattens rather than jumping.
//
// The first n-1 values are NaN. A NaN inside the range window makes the output NaN and,
// because the recurrence cannot forget, every later output too.
func FisherTransform(high, low []float64, n int) []float64 {
	checkPeriod("FisherTransform", n)
	size := requireSameLen("FisherTransform", high, low)
	if size == 0 {
		return nil
	}

	price := MedianPrice(high, low)
	hi := Highest(high, n)
	lo := Lowest(low, n)

	out := allNaN(size)
	var value, fish float64
	for i := n - 1; i < size; i++ {
		switch {
		case hi[i] != hi[i] || lo[i] != lo[i] || price[i] != price[i]:
			value, fish = math.NaN(), math.NaN()
			out[i] = math.NaN()
			continue
		}
		span := hi[i] - lo[i]
		position := 0.5
		if span != 0 {
			position = (price[i] - lo[i]) / span
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

// MassIndex returns the Mass Index over the given EMA period and summation period:
//
//	range[i] = high[i] - low[i]
//	ratio    = EMA(range, emaPeriod) / EMA(range, sumPeriod)
//	MI       = sum(ratio, sumPeriod)
//
// It is a range-expansion detector: a bulge above the traditional threshold of 27 is read
// as a coming reversal, because the two EMAs separate when the daily range widens
// faster than its own trend. TradingView's defaults are 9 and 25.
//
// The first max(emaPeriod, sumPeriod) + sumPeriod - 2 values are NaN: the ratio needs both
// EMAs, and the summation needs a full window of ratios. A zero slow EMA makes the ratio 0.
func MassIndex(high, low []float64, emaPeriod, sumPeriod int) []float64 {
	checkPeriod("MassIndex emaPeriod", emaPeriod)
	checkPeriod("MassIndex sumPeriod", sumPeriod)
	size := requireSameLen("MassIndex", high, low)
	if size == 0 {
		return nil
	}

	rng := make([]float64, size)
	for i := range high {
		rng[i] = high[i] - low[i]
	}
	fastEMA := EMA(rng, emaPeriod)
	slowEMA := EMA(rng, sumPeriod)

	ratio := make([]float64, size)
	for i := range rng {
		switch {
		case fastEMA[i] != fastEMA[i] || slowEMA[i] != slowEMA[i]:
			ratio[i] = math.NaN()
		case slowEMA[i] == 0:
			ratio[i] = 0
		default:
			ratio[i] = fastEMA[i] / slowEMA[i]
		}
	}

	// The summation window counts NaN ratios rather than accumulating them, so the
	// output becomes defined as soon as the window holds only real ratios.
	sum := series.NewSum(sumPeriod)
	out := allNaN(size)
	for i := range ratio {
		out[i] = sum.Push(ratio[i])
	}
	return out
}
