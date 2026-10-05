package ta

// ---------------------------------------------------------------------------
// Guppy Multiple Moving Average and the Zig Zag pivot detector.
//
// Both of these describe structure rather than level: the GMMA through the *spread* between
// two groups of averages, and the Zig Zag through the sequence of confirmed swing points. So
// neither has a single meaningful output value, and both are shaped differently from the rest
// of the package.
// ---------------------------------------------------------------------------

// gmmaShortPeriods and gmmaLongPeriods are Daryl Guppy's original period sets. They are fixed
// rather than parameters because the indicator *is* the pair of groups; changing the periods
// produces a different indicator, not a different setting of this one.
var (
	gmmaShortPeriods = [6]int{3, 5, 8, 10, 12, 15}
	gmmaLongPeriods  = [6]int{30, 35, 40, 45, 50, 60}
)

// GMMA returns the Guppy Multiple Moving Average: six short exponential averages and six long
// ones.
//
// The signal is the relationship between the groups rather than any line's value. When the
// short group is above the long group and both are fanning apart, the trend is established;
// when the short group compresses into the long group, it is stalling. A single line from
// either group carries almost none of that information, which is why both groups are returned
// rather than a summary.
//
// The periods are Guppy's: 3, 5, 8, 10, 12, 15 for the short group and 30, 35, 40, 45, 50, 60
// for the long. Each element is a full-length series, so a caller has twelve slices indexed by
// the same bar as the input. The longest warm-up is 59 bars.
func GMMA(close []float64) (short, long [6][]float64) {
	if len(close) == 0 {
		return short, long
	}
	for i := 0; i < 6; i++ {
		short[i] = EMA(close, gmmaShortPeriods[i])
		long[i] = EMA(close, gmmaLongPeriods[i])
	}
	return short, long
}

// ZigZag returns the confirmed pivot sequence implied by a minimum retracement of deviation,
// a fraction such as 0.05 for five percent.
//
// The returned slice holds, at each bar, the value of the most recently *confirmed* pivot,
// forward-filled; kind is -1 for a confirmed low, +1 for a confirmed high, and 0 before any
// pivot is confirmed. Forward-filling rather than marking only the pivot bar makes the series
// usable as a level -- the last confirmed swing low is a meaningful support price on every bar
// after it.
//
// # It is causal by construction, and that costs latency
//
// A swing high is not knowable when it happens; it is confirmed only once price has retraced
// by the threshold. This implementation reports the pivot on the bar that confirms it, never
// earlier, so a strategy reading this series cannot be trading on the future. The price paid
// is that the reported pivot is always `deviation` of movement old, which is what the
// indicator's apparent prescience on a historical chart actually is.
//
// # The initial pivot is an assumption
//
// The first bar has no preceding swing, so it is taken as an initial low pivot by convention,
// and kind[0] is -1. A series that actually opened at a high will correct itself on the first
// confirmed reversal; the alternative, emitting nothing until a full cycle has completed,
// leaves the first leg of every series unusable.
//
// A zero deviation confirms a pivot on every bar and is legal but useless; a negative one
// panics. Both the high and low of a bar are used, so a bar whose range exceeds the threshold
// on both sides confirms the extreme in the current direction.
func ZigZag(high, low []float64, deviation float64) (pivot []float64, kind []int8) {
	if deviation < 0 {
		panic("ta: ZigZag deviation must be >= 0")
	}
	size := requireSameLen("ZigZag", high, low)
	if size == 0 {
		return nil, nil
	}

	pivot = make([]float64, size)
	kind = make([]int8, size)

	// Start in an up-leg from an assumed low pivot at bar 0.
	upLeg := true
	lastPivot := low[0]
	lastKind := int8(-1)
	pivot[0] = lastPivot
	kind[0] = lastKind
	running := high[0]

	for i := 1; i < size; i++ {
		if upLeg {
			if high[i] > running {
				running = high[i]
			}
			// A retracement of `deviation` from the running high confirms it.
			if low[i] <= running*(1-deviation) {
				lastPivot, lastKind = running, 1
				upLeg = false
				running = low[i]
			}
		} else {
			if low[i] < running {
				running = low[i]
			}
			if high[i] >= running*(1+deviation) {
				lastPivot, lastKind = running, -1
				upLeg = true
				running = high[i]
			}
		}
		pivot[i] = lastPivot
		kind[i] = lastKind
	}
	return pivot, kind
}
