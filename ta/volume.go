package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/series"
)

// ---------------------------------------------------------------------------
// Volume indicators.
//
// These are the indicators that need a volume column. They fall into three shapes:
//
//   - **Cumulative** (VWAP, OBV, PVT, A/D): a running total from the start of the
//     slice to the current bar. There is no warm-up, because the first bar has a
//     defined contribution, and there is no window, because the anchor is the start
//     of the data. Anchoring to a session or a year is the caller's job: pass the
//     slice that starts at the anchor.
//
//   - **Windowed** (VWMA, Chaikin Money Flow, Money Flow Index, Volume Oscillator):
//     a window over the volume-weighted quantities, warm-up n-1 or n. These are
//     maintained with rolling sums, which are O(1) per element where resumming the
//     window would be O(window) -- the same substitution that made CMO flat in its
//     period, and the same reason it is worth doing here rather than later.
//
//   - **Recursive** (Chaikin Oscillator, Force Index): a smoother over one of the
//     above, which inherits that indicator's warm-up.
//
// # A zero volume denominator
//
// Every ratio here divides by a sum of volumes. A window with no volume at all is
// possible in illiquid data, and an infinity produced there would poison everything
// downstream, so the result is 0 rather than an infinity. Each such function says so.
// ---------------------------------------------------------------------------

// VWAP returns the volume-weighted average price,
//
//	sum(typicalPrice * volume) / sum(volume)
//
// accumulated from the first element of the slices, with the typical price taken as
// (high + low + close) / 3.
//
// It has no warm-up: the first bar's value is its own typical price. **Anchoring is
// the caller's responsibility** -- this function anchors at index 0, so a caller who
// wants a session, day, week or year to date passes the slice that begins at that
// boundary. That keeps the anchor policy out of the numeric layer.
//
// A zero cumulative volume yields 0.
func VWAP(high, low, close, volume []float64) []float64 {
	size := requireSameLen("VWAP", high, low, close, volume)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	var pv, vol float64
	for i := 0; i < size; i++ {
		tp := (high[i] + low[i] + close[i]) / 3
		pv += tp * volume[i]
		vol += volume[i]
		if vol == 0 {
			out[i] = 0
			continue
		}
		out[i] = pv / vol
	}
	return out
}

// VWMA returns the n-period volume-weighted moving average of close,
//
//	sum(close * volume, n) / sum(volume, n)
//
// The first n-1 values are NaN, and a window whose volume sum is zero yields 0.
//
// The two sums are maintained with rolling accumulators rather than resummed over the
// window, so the cost is independent of n.
func VWMA(close, volume []float64, n int) []float64 {
	checkPeriod("VWMA", n)
	size := requireSameLen("VWMA", close, volume)
	if size == 0 {
		return nil
	}
	out := allNaN(size)
	cvSum := series.NewSum(n)
	vSum := series.NewSum(n)
	for i := 0; i < size; i++ {
		cv := cvSum.Push(close[i] * volume[i])
		v := vSum.Push(volume[i])
		if i+1 < n {
			continue
		}
		if cv != cv || v != v {
			out[i] = math.NaN()
			continue
		}
		if v == 0 {
			out[i] = 0
			continue
		}
		out[i] = cv / v
	}
	return out
}

// OBV returns the On Balance Volume line: a running total that adds the bar's volume
// when close rose, subtracts it when close fell, and leaves it unchanged when close
// was flat.
//
// The first value is 0, because the first bar has no previous close and therefore no
// direction. Some platforms start the line at the first bar's volume instead; the
// choice here is documented because it shifts the whole line by a constant.
//
// There is no warm-up.
func OBV(close, volume []float64) []float64 {
	size := requireSameLen("OBV", close, volume)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	var total float64
	for i := 1; i < size; i++ {
		switch {
		case close[i] != close[i] || close[i-1] != close[i-1]:
			total = math.NaN()
		case close[i] > close[i-1]:
			total += volume[i]
		case close[i] < close[i-1]:
			total -= volume[i]
		}
		out[i] = total
	}
	return out
}

// moneyFlowVolume is the Accumulation/Distribution term for one bar:
//
//	((close - low) - (high - close)) / (high - low) * volume
//
// It is the position of the close within the bar's range, scaled to [-1, 1] by the
// numerator over the range, multiplied by volume. A bar with no range (high == low)
// contributes 0 rather than dividing by zero, which is the standard treatment: the
// close cannot be located within a range that does not exist.
func moneyFlowVolume(high, low, close, volume float64) float64 {
	span := high - low
	if span == 0 || span != span {
		return 0
	}
	return ((close - low) - (high - close)) / span * volume
}

// AccumulationDistribution returns the Accumulation/Distribution line: the running
// total of [moneyFlowVolume].
//
// The first value is the first bar's contribution, following the usual definition
// `ad[i] = nz(ad[i-1]) + mfv[i]`. There is no warm-up.
func AccumulationDistribution(high, low, close, volume []float64) []float64 {
	size := requireSameLen("AccumulationDistribution", high, low, close, volume)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	var total float64
	for i := 0; i < size; i++ {
		total += moneyFlowVolume(high[i], low[i], close[i], volume[i])
		out[i] = total
	}
	return out
}

// ChaikinMoneyFlow returns the n-period Chaikin Money Flow:
//
//	sum(moneyFlowVolume, n) / sum(volume, n)
//
// It is the accumulation/distribution term normalized by the volume that produced it,
// so it lies in [-1, 1] and is comparable across instruments. A window with no volume
// yields 0.
//
// The first n-1 values are NaN. Both sums use rolling accumulators.
func ChaikinMoneyFlow(high, low, close, volume []float64, n int) []float64 {
	checkPeriod("ChaikinMoneyFlow", n)
	size := requireSameLen("ChaikinMoneyFlow", high, low, close, volume)
	if size == 0 {
		return nil
	}
	out := allNaN(size)
	mfvSum := series.NewSum(n)
	vSum := series.NewSum(n)
	for i := 0; i < size; i++ {
		mfv := mfvSum.Push(moneyFlowVolume(high[i], low[i], close[i], volume[i]))
		v := vSum.Push(volume[i])
		if i+1 < n {
			continue
		}
		if mfv != mfv || v != v {
			out[i] = math.NaN()
			continue
		}
		if v == 0 {
			out[i] = 0
			continue
		}
		out[i] = mfv / v
	}
	return out
}

// ChaikinOscillator returns the difference between two exponential moving averages of
// the Accumulation/Distribution line:
//
//	EMA(ad, fast) - EMA(ad, slow)
//
// The first slow-1 values are NaN. It is a momentum view of accumulation: the fast
// average leads, so a positive reading means accumulation is accelerating.
func ChaikinOscillator(high, low, close, volume []float64, fast, slow int) []float64 {
	checkPeriod("ChaikinOscillator fast", fast)
	checkPeriod("ChaikinOscillator slow", slow)
	size := requireSameLen("ChaikinOscillator", high, low, close, volume)
	if size == 0 {
		return nil
	}
	ad := AccumulationDistribution(high, low, close, volume)
	fastEMA := EMA(ad, fast)
	slowEMA := EMA(ad, slow)
	out := make([]float64, size)
	for i := range ad {
		out[i] = fastEMA[i] - slowEMA[i]
	}
	return out
}

// MoneyFlowIndex returns the n-period Money Flow Index, a volume-weighted RSI in
// [0,100].
//
//	typical = (high + low + close) / 3
//	raw     = typical * volume
//	MFI     = 100 - 100 / (1 + positiveFlow / negativeFlow)
//
// where positive flow is the sum of `raw` over bars whose typical price rose and
// negative flow the sum over bars whose typical price fell. A flat typical price
// contributes to neither.
//
// The first n values are NaN: n flow terms each need a previous typical price, so the
// earliest complete window ends at index n.
//
// If all flow in the window is positive the result is 100, and if none of it is
// positive or negative -- a window of unchanged typical prices -- the result is 50
// rather than 100. That distinction matters: 100 would claim overwhelming buying
// pressure where there was none.
func MoneyFlowIndex(high, low, close, volume []float64, n int) []float64 {
	checkPeriod("MoneyFlowIndex", n)
	size := requireSameLen("MoneyFlowIndex", high, low, close, volume)
	if size == 0 {
		return nil
	}
	out := allNaN(size)
	if size <= n {
		return out
	}

	pos := series.NewSum(n)
	neg := series.NewSum(n)
	for i := 1; i < size; i++ {
		tp := (high[i] + low[i] + close[i]) / 3
		tpPrev := (high[i-1] + low[i-1] + close[i-1]) / 3
		raw := tp * volume[i]

		var up, down float64
		switch {
		case tp != tp || tpPrev != tpPrev:
			up, down = math.NaN(), math.NaN()
		case tp > tpPrev:
			up = raw
		case tp < tpPrev:
			down = raw
		}
		p := pos.Push(up)
		q := neg.Push(down)

		if i < n {
			continue
		}
		switch {
		case p != p || q != q:
			out[i] = math.NaN()
		case p == 0 && q == 0:
			out[i] = 50
		case q == 0:
			out[i] = 100
		default:
			out[i] = 100 - 100/(1+p/q)
		}
	}
	return out
}

// PriceVolumeTrend returns the Price Volume Trend line: the running total of
//
//	(close[i] - close[i-1]) / close[i-1] * volume[i]
//
// which is cumulative volume weighted by the percentage change in price. The first
// value is 0, because the first bar has no percentage change. A zero previous close
// contributes 0.
//
// Unlike [OBV], which adds or subtracts the whole volume, PVT scales by how far price
// moved, so a 5% bar counts five times a 1% bar.
func PriceVolumeTrend(close, volume []float64) []float64 {
	size := requireSameLen("PriceVolumeTrend", close, volume)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	var total float64
	for i := 1; i < size; i++ {
		if close[i-1] != 0 {
			total += (close[i] - close[i-1]) / close[i-1] * volume[i]
		}
		out[i] = total
	}
	return out
}

// NetVolume returns volume signed by the direction of the close:
// +volume when close rose, -volume when it fell, and 0 when it was unchanged or when
// there is no previous close.
//
// It is the per-bar ingredient of [OBV] exposed on its own, which is what a caller
// wants when they intend to aggregate it differently -- by week, by symbol, or masked
// by some other condition.
func NetVolume(close, volume []float64) []float64 {
	size := requireSameLen("NetVolume", close, volume)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	for i := 1; i < size; i++ {
		switch {
		case close[i] != close[i] || close[i-1] != close[i-1]:
			out[i] = math.NaN()
		case close[i] > close[i-1]:
			out[i] = volume[i]
		case close[i] < close[i-1]:
			out[i] = -volume[i]
		}
	}
	return out
}

// VolumeOscillator returns the difference between a fast and a slow simple moving
// average of volume, as a percentage of the slow one:
//
//	100 * (SMA(volume, fast) - SMA(volume, slow)) / SMA(volume, slow)
//
// A zero slow average yields 0. The first slow-1 values are NaN. It is a pure volume
// measure and ignores price entirely.
func VolumeOscillator(volume []float64, fast, slow int) []float64 {
	checkPeriod("VolumeOscillator fast", fast)
	checkPeriod("VolumeOscillator slow", slow)
	if len(volume) == 0 {
		return nil
	}
	fastMA := SMA(volume, fast)
	slowMA := SMA(volume, slow)
	out := make([]float64, len(volume))
	for i := range volume {
		switch {
		case fastMA[i] != fastMA[i] || slowMA[i] != slowMA[i]:
			out[i] = math.NaN()
		case slowMA[i] == 0:
			out[i] = 0
		default:
			out[i] = 100 * (fastMA[i] - slowMA[i]) / slowMA[i]
		}
	}
	return out
}

// ForceIndex returns Elder's n-period Force Index: the volume-weighted price change,
// smoothed.
//
//	raw[i] = (close[i] - close[i-1]) * volume[i]
//	Force  = EMA(raw, n)
//
// The first raw value is undefined because there is no previous close, so the smoother
// is started at index 1 rather than being fed that leading NaN as seed material; see
// [applyFrom]. The first n values are therefore NaN.
func ForceIndex(close, volume []float64, n int) []float64 {
	checkPeriod("ForceIndex", n)
	size := requireSameLen("ForceIndex", close, volume)
	if size == 0 {
		return nil
	}
	raw := make([]float64, size)
	fillNaN(raw, 0, 1)
	for i := 1; i < size; i++ {
		raw[i] = (close[i] - close[i-1]) * volume[i]
	}
	return applyEMA(raw, 1, n)
}
