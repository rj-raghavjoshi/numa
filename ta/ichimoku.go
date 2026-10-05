package ta

import "github.com/rj-raghavjoshi/numa/vec"

// ---------------------------------------------------------------------------
// Ichimoku Kinko Hyo.
//
// Five lines, three of them plain midpoints of a rolling range, and the whole indicator is
// therefore an exercise in displacement: two of the lines are plotted *ahead* of the data
// that produces them and one is plotted behind.
// ---------------------------------------------------------------------------

// Ichimoku returns the five Ichimoku lines, aligned to the bar they are plotted at:
//
//	conversion = midpoint(high, low, conversionPeriod)               (tenkan-sen)
//	base       = midpoint(high, low, basePeriod)                     (kijun-sen)
//	spanA      = (conversion + base) / 2, displaced forward
//	spanB      = midpoint(high, low, spanBPeriod), displaced forward
//	lagging    = close, displaced backward                            (chikou span)
//
// where midpoint(h, l, n) is (highest(high, n) + lowest(low, n)) / 2.
//
// # Displacement, and why the output is aligned to the plot
//
// The two leading spans are plotted `displacement` bars to the right of the bar whose
// range produced them, and the lagging span is plotted `displacement` bars to the left.
// This function returns series indexed by the bar each value is *drawn* at, not the bar it
// was computed from:
//
//	spanA[i]   = midpointPair[i - displacement]
//	lagging[i] = close[i + displacement]
//
// The consequence is that the leading spans start with `displacement` NaNs while the
// lagging span ends with them. A caller who instead wants the value as known at bar i should
// use the conversion, base and spanB series, which are not displaced at all.
//
// # Warm-up
//
// The conversion line starts at conversionPeriod-1, the base line at basePeriod-1, the
// leading spans `displacement` bars after their inputs are ready, and the lagging span is
// defined immediately except for its final `displacement` bars.
//
// TradingView's defaults are 9, 26, 52 and 26.
func Ichimoku(high, low, close []float64, conversionPeriod, basePeriod, spanBPeriod, displacement int) (conversion, base, spanA, spanB, lagging []float64) {
	checkPeriod("Ichimoku conversionPeriod", conversionPeriod)
	checkPeriod("Ichimoku basePeriod", basePeriod)
	checkPeriod("Ichimoku spanBPeriod", spanBPeriod)
	if displacement < 0 {
		panic("ta: Ichimoku displacement must be >= 0")
	}
	size := requireSameLen("Ichimoku", high, low, close)
	if size == 0 {
		return nil, nil, nil, nil, nil
	}

	conversion = midpointLine(high, low, conversionPeriod)
	base = midpointLine(high, low, basePeriod)
	spanBMid := midpointLine(high, low, spanBPeriod)

	spanA = make([]float64, size)
	for i := range close {
		spanA[i] = (conversion[i] + base[i]) / 2
	}
	if displacement > 0 {
		spanA = vec.Shift(spanA, displacement)
		spanBMid = vec.Shift(spanBMid, displacement)
		lagging = vec.Shift(close, -displacement)
	} else {
		lagging = make([]float64, size)
		copy(lagging, close)
	}
	return conversion, base, spanA, spanBMid, lagging
}

// midpointLine returns (highest(high, n) + lowest(low, n)) / 2, the rolling range midpoint
// the Ichimoku lines are built from.
func midpointLine(high, low []float64, n int) []float64 {
	hi := Highest(high, n)
	lo := Lowest(low, n)
	out := make([]float64, len(high))
	for i := range high {
		out[i] = (hi[i] + lo[i]) / 2
	}
	return out
}
