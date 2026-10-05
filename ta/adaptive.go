package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Moving averages beyond the basic set.
//
// `moving.go` holds the averages whose only choice is the period: SMA, EMA, WMA and their
// composites. The ones here each add a second idea, and the file is organised around what that
// idea is:
//
//	ALMA      a fixed non-symmetric weight profile
//	SWMA      a fixed weight profile with no period at all
//	KAMA      a smoothing constant driven by an efficiency ratio
//	VIDYA     a smoothing constant driven by a momentum index
//	ZLEMA     a de-lagged input fed to a plain EMA
//	T3        repeated EMA passes combined with a volume factor
//	McGinley a feedback term that resists the average being left behind
//
// The distinction matters because the adaptive ones are *recursive*: their state is not a window,
// so a NaN in their input is permanent rather than lasting for n bars. That is documented on each,
// and it is the same behaviour `series.NewEMA` has and `series.NewSum` deliberately does not.
// ---------------------------------------------------------------------------

// ALMA returns the Arnaud Legoux Moving Average.
//
//	weights w_j = exp(-(j - m)^2 / (2 s^2)),  m = offset*(n-1),  s = n/sigma
//	ALMA[i]      = sum_j w_j * xs[i-n+1+j] / sum_j w_j
//
// The weight profile is a Gaussian whose centre and width are set by offset and sigma, which is
// what separates it from a WMA: the weights are not monotone and can be concentrated anywhere in
// the window. A higher offset shifts the centre towards the newest bar and reduces lag; a lower
// sigma narrows the window the average effectively looks at.
//
// The weights depend only on n, offset and sigma, so they are computed once per call rather than
// per bar. TradingView's defaults are offset = 0.85 and sigma = 6; offset is a fraction of the
// window and must lie in [0,1], and sigma must be positive.
//
// The cost is O(n) per element. The first n-1 values are NaN.
func ALMA(xs []float64, n int, offset, sigma float64) []float64 {
	checkPeriod("ALMA", n)
	if sigma <= 0 {
		panic("ta: ALMA sigma must be > 0")
	}
	if offset < 0 || offset > 1 {
		panic("ta: ALMA offset must be in [0,1]")
	}
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	out := allNaN(size)
	if size < n {
		return out
	}

	weights := make([]float64, n)
	m := offset * float64(n-1)
	s := float64(n) / sigma
	twoS2 := 2 * s * s
	var wsum float64
	for j := 0; j < n; j++ {
		d := float64(j) - m
		w := math.Exp(-(d * d) / twoS2)
		weights[j] = w
		wsum += w
	}

	for i := n - 1; i < size; i++ {
		base := i - n + 1
		var acc float64
		bad := false
		for j := 0; j < n; j++ {
			v := xs[base+j]
			if v != v {
				bad = true
				break
			}
			acc += weights[j] * v
		}
		if bad {
			continue
		}
		out[i] = acc / wsum
	}
	return out
}

// SWMA returns the Symmetrically Weighted Moving Average, the fixed four-bar average with weights
// 1/6, 2/6, 2/6, 1/6 from oldest to newest.
//
// There is no period parameter, and that is the definition rather than a simplification: the
// indicator *is* the centred weighting, which has the same shape as the 1-2-2-1 row of a smoothing
// filter. A caller wanting the same profile over a different window wants a WMA or an ALMA.
//
// The first three values are NaN.
func SWMA(xs []float64) []float64 {
	const n = 4
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	out := allNaN(size)
	if size < n {
		return out
	}
	for i := n - 1; i < size; i++ {
		base := i - n + 1
		var acc float64
		bad := false
		for j := 0; j < n; j++ {
			v := xs[base+j]
			if v != v {
				bad = true
				break
			}
			acc += swmaWeights[j] * v
		}
		if bad {
			continue
		}
		out[i] = acc
	}
	return out
}

// swmaWeights are the fixed 1-2-2-1 profile divided by six, oldest first.
var swmaWeights = [4]float64{1.0 / 6, 2.0 / 6, 2.0 / 6, 1.0 / 6}

// KAMA returns Kaufman's Adaptive Moving Average.
//
//	change     = |close[i] - close[i-n]|
//	volatility = sum of |close[j] - close[j-1]| over the last n bars
//	ER         = change / volatility                 (0 when volatility is 0)
//	SC         = (ER*(2/(fast+1) - 2/(slow+1)) + 2/(slow+1))^2
//	KAMA[i]    = KAMA[i-1] + SC*(close[i] - KAMA[i-1])
//
// The efficiency ratio measures how much of the bar-to-bar movement went anywhere: a straight line
// gives ER near 1 and a fast smoothing constant, while a market that oscillates without progress
// gives ER near 0 and the slowest constant. That is the adaptation -- the average speeds up in a
// trend and slows down in noise, without a separate regime switch.
//
// The smoothing constant is *squared*, which is Kaufman's original choice and is what makes the
// slow end genuinely slow. TradingView's defaults are fast = 2 and slow = 30, and fast must be less
// than slow.
//
// # Warm-up and seeding
//
// The first value is at index n, seeded with the close at that bar, because the efficiency ratio
// needs n bars of change behind it. Every later value is recursive, so a NaN is permanent: unlike
// the windowed averages, KAMA cannot forget one.
func KAMA(xs []float64, n, fast, slow int) []float64 {
	checkPeriod("KAMA", n)
	checkPeriod("KAMA fast", fast)
	checkPeriod("KAMA slow", slow)
	if fast >= slow {
		panic("ta: KAMA fast period must be less than slow period")
	}
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	out := allNaN(size)
	if size <= n {
		return out
	}

	// The one-bar absolute changes, with NaN at index 0 so the rolling sum cannot include it.
	changes := make([]float64, size)
	changes[0] = math.NaN()
	for i := 1; i < size; i++ {
		changes[i] = math.Abs(xs[i] - xs[i-1])
	}
	volatility := vec.RollingSum(changes, n)

	fastSC := 2 / float64(fast+1)
	slowSC := 2 / float64(slow+1)
	diff := fastSC - slowSC

	prev := xs[n]
	out[n] = prev
	for i := n + 1; i < size; i++ {
		v := volatility[i]
		if v != v || xs[i] != xs[i] || xs[i-n] != xs[i-n] || prev != prev {
			out[i] = math.NaN()
			prev = math.NaN()
			continue
		}
		er := 0.0
		if v != 0 {
			er = math.Abs(xs[i]-xs[i-n]) / v
		}
		sc := er*diff + slowSC
		sc *= sc
		prev += sc * (xs[i] - prev)
		out[i] = prev
	}
	return out
}

// VIDYA returns Chande's Variable Index Dynamic Average.
//
//	alpha[i] = (2/(n+1)) * |CMO[i]| / 100
//	VIDYA[i] = alpha[i]*xs[i] + (1-alpha[i])*VIDYA[i-1]
//
// The smoothing constant is scaled by the absolute Chande Momentum Oscillator, so the average
// tracks price closely when momentum is strong in either direction and barely moves when momentum
// is absent. Note the absolute value: a strong *down* move is as informative as a strong up move,
// and using the signed oscillator would make the average fast only in uptrends.
//
// The first value is seeded with the close at the bar where the CMO first becomes defined (see
// [CMO], which needs cmoPeriod bars behind it), and every later value is recursive, so a NaN is
// permanent.
func VIDYA(xs []float64, n, cmoPeriod int) []float64 {
	checkPeriod("VIDYA", n)
	checkPeriod("VIDYA cmoPeriod", cmoPeriod)
	if len(xs) == 0 {
		return nil
	}
	out := allNaN(len(xs))
	cmo := CMO(xs, cmoPeriod)
	start := FirstValid(cmo)
	if start < 0 || start >= len(xs) {
		return out
	}

	k := 2 / float64(n+1)
	prev := xs[start]
	out[start] = prev
	for i := start + 1; i < len(xs); i++ {
		if cmo[i] != cmo[i] || xs[i] != xs[i] || prev != prev {
			out[i] = math.NaN()
			prev = math.NaN()
			continue
		}
		alpha := k * math.Abs(cmo[i]) / 100
		prev = alpha*xs[i] + (1-alpha)*prev
		out[i] = prev
	}
	return out
}

// ZLEMA returns the Zero-Lag Exponential Moving Average.
//
//	lag  = (n-1)/2
//	zl[i] = 2*xs[i] - xs[i-lag]
//	ZLEMA = EMA(zl, n)
//
// The de-lagged input extrapolates each bar by the amount the average would have lagged it, which
// is a first-order correction and not a removal of lag: the result is still causal, still smooths,
// and overshoots at turning points precisely because it is extrapolating. That overshoot is the
// trade, not a defect.
//
// lag is integer division, so n = 1 gives lag 0 and ZLEMA reduces to the EMA. The first
// lag+n-1 values are NaN.
func ZLEMA(xs []float64, n int) []float64 {
	checkPeriod("ZLEMA", n)
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	lag := (n - 1) / 2

	zl := allNaN(size)
	for i := lag; i < size; i++ {
		if xs[i] != xs[i] || xs[i-lag] != xs[i-lag] {
			continue
		}
		zl[i] = 2*xs[i] - xs[i-lag]
	}
	return applyEMA(zl, lag, n)
}

// T3 returns Tillson's T3 moving average.
//
//	E1 = EMA(xs, n), E2 = EMA(E1, n), ... E6 = EMA(E5, n)
//	T3 = c1*E6 + c2*E5 + c3*E4 + c4*E3
//
// with the coefficients derived from the volume factor v:
//
//	c1 = -v^3
//	c2 = 3v^2 + 3v^3
//	c3 = -6v^2 - 3v - 3v^3
//	c4 = 1 + 3v + 3v^2 + v^3
//
// Six smoothing passes make T3 exceptionally smooth, and the weighted combination of the last four
// puts back some responsiveness without reintroducing the lag that a plain sixth-order average would
// have. It follows that the warm-up is long: six chained EMAs cost 6(n-1) bars, and the
// coefficients are only one constraint among several that Tillson chose by fitting.
//
// v must be in [0,1]; TradingView's default is 0.7.
//
// The warm-up is 6(n-1) except when a coefficient vanishes: at v = 0 three of the four are zero and
// the indicator collapses to the third EMA, so it becomes defined at 3(n-1) instead. That is the
// formula being read literally rather than a special case in the code -- see the comment there.
func T3(xs []float64, n int, v float64) []float64 {
	checkPeriod("T3", n)
	if v < 0 || v > 1 || v != v {
		panic("ta: T3 volume factor must be in [0,1]")
	}
	if len(xs) == 0 {
		return nil
	}

	e1 := EMA(xs, n)
	e2 := applyEMA(e1, FirstValid(e1), n)
	e3 := applyEMA(e2, FirstValid(e2), n)
	e4 := applyEMA(e3, FirstValid(e3), n)
	e5 := applyEMA(e4, FirstValid(e4), n)
	e6 := applyEMA(e5, FirstValid(e5), n)

	v2 := v * v
	v3 := v2 * v
	c1 := -v3
	c2 := 3*v2 + 3*v3
	c3 := -6*v2 - 3*v - 3*v3
	c4 := 1 + 3*v + 3*v2 + v3

	// Only terms with a non-zero coefficient are required to be valid. For any v strictly between 0
	// and 1 all four coefficients are non-zero and the warm-up is the full 6(n-1), but v = 0 zeroes
	// three of them and the formula degenerates to the third EMA -- which is defined 3(n-1) bars
	// earlier. Gating on all six regardless would report NaN for bars where the stated formula has a
	// perfectly good value, and the algebra is the definition.
	terms := [4]struct {
		coef   float64
		series []float64
	}{
		{c1, e6}, {c2, e5}, {c3, e4}, {c4, e3},
	}
	out := allNaN(len(xs))
	for i := range xs {
		var acc float64
		ok := true
		for _, t := range terms {
			if t.coef == 0 {
				continue
			}
			if t.series[i] != t.series[i] {
				ok = false
				break
			}
			acc += t.coef * t.series[i]
		}
		if ok {
			out[i] = acc
		}
	}
	return out
}

// McGinleyDynamic returns the McGinley Dynamic indicator.
//
//	MD[i] = MD[i-1] + (xs[i] - MD[i-1]) / (0.6*n*(xs[i]/MD[i-1])^4)
//
// The divisor is the whole idea: when price runs away from the average the ratio grows and the
// fourth power makes the divisor grow very fast, so the average accelerates towards price; when
// price is near the average the divisor approaches 0.6n and the average moves slowly. A plain
// moving average is a compromise between those two regimes, and this tracks both.
//
// The fourth power also makes the indicator sensitive to the sign of the ratio, which is handled by
// the even exponent. It is seeded with the first close and is defined from index 0.
//
// A zero or non-positive previous value would make the ratio undefined; the result is NaN from that
// bar onward, because the recurrence cannot recover.
func McGinleyDynamic(xs []float64, n int) []float64 {
	checkPeriod("McGinleyDynamic", n)
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	out := allNaN(size)
	prev := xs[0]
	out[0] = prev
	for i := 1; i < size; i++ {
		v := xs[i]
		if v != v || prev != prev || prev == 0 {
			out[i] = math.NaN()
			prev = math.NaN()
			continue
		}
		ratio := v / prev
		divisor := 0.6 * float64(n) * ratio * ratio * ratio * ratio
		if divisor == 0 {
			out[i] = math.NaN()
			prev = math.NaN()
			continue
		}
		prev += (v - prev) / divisor
		out[i] = prev
	}
	return out
}
