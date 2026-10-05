package ta

import "math"

// ---------------------------------------------------------------------------
// Wilder's Swing Index and its accumulation.
//
// The Swing Index is the one indicator in this package whose output is not a comparison across bars
// but a *model of a single bar*: it asks how far this bar's range, body and close moved relative to
// the previous bar's, and it answers in a unit that is meant to be comparable across instruments
// once the caller supplies that instrument's limit move.
//
// It is also the only one here that cannot be written without the open. Every other price indicator
// in this package works from the high, the low and the close; the Swing Index uses the body's
// direction, and the body needs the open.
// ---------------------------------------------------------------------------

// swingIndexAt returns the one-bar Swing Index for bar i, or NaN if any input is missing.
//
// It is the single definition of the formula; the two exported functions differ only in whether they
// accumulate it. Duplicating it would have been the obvious mistake here, because the two would then
// have to be kept in step across three branch conditions that are easy to mistype.
func swingIndexAt(open, high, low, close []float64, i int, limitMove float64) float64 {
	o1, c1 := open[i-1], close[i-1]
	o, h, l, c := open[i], high[i], low[i], close[i]
	if o != o || h != h || l != l || c != c || o1 != o1 || c1 != c1 {
		return math.NaN()
	}

	k := math.Max(h, c1) - math.Min(l, c1)
	absHC := math.Abs(h - c1)
	absLC := math.Abs(l - c1)
	absCO := math.Abs(c1 - o1)
	barRange := h - l

	var r float64
	switch {
	case absHC >= absLC && absHC >= barRange:
		r = absHC - 0.5*absLC + 0.25*absCO
	case absLC >= absHC && absLC >= barRange:
		r = absLC - 0.5*absHC + 0.25*absCO
	default:
		r = barRange + 0.25*absCO
	}
	// A bar with no displacement in any direction has no index; zero is the additive identity and
	// keeps an accumulator defined, which is the useful answer for a market that did not move.
	if r == 0 {
		return 0
	}
	return 50 * (c - c1 + 0.5*(c-o) + 0.25*(c1-o1)) / r * (k / limitMove)
}

// AccumulativeSwingIndex returns Wilder's Accumulative Swing Index: the running total of the
// one-bar Swing Index.
//
// # The per-bar index
//
//	K  = max(high, prevClose) - min(low, prevClose)
//	R  = one of three forms, chosen by which displacement dominates:
//	     |high - prevClose| when that is largest, minus half of |low - prevClose|,
//	     |low  - prevClose| when that is largest, minus half of |high - prevClose|,
//	     (high - low) when the bar's own range is largest,
//	     each plus 0.25*|prevClose - prevOpen|
//	SI = 50 * (close - prevClose + 0.5*(close - open) + 0.25*(prevClose - prevOpen)) / R * (K / limitMove)
//
// The three-way choice of R is the part worth reading twice. It is not a maximum: each branch
// subtracts half of the *other* displacement, so a bar that gapped far in one direction scores lower
// than its gap alone would suggest, which is the intended damping. `limitMove` is Wilder's T -- the
// largest move the instrument is allowed to make in one session, so that a limit-up bar does not
// produce an unbounded reading. It is a property of the instrument and has no default, because a
// wrong value scales every output.
//
// # Accumulation and NaN
//
// The result is a cumulative sum, so it is defined from index 0 with a seed of 0 and the first
// increment at index 1. A NaN in any input makes that bar's increment NaN, and because the series
// accumulates, the total is NaN from that bar onward: this is the [EMA]-like asymmetry, not the
// [series.Sum] one, and it is inherent to an accumulator rather than a choice.
//
// It panics if limitMove is not positive, since the parameter divides the result.
func AccumulativeSwingIndex(open, high, low, close []float64, limitMove float64) []float64 {
	checkLimitMove("AccumulativeSwingIndex", limitMove)
	size := requireSameLen("AccumulativeSwingIndex", open, high, low, close)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	total := 0.0
	for i := 1; i < size; i++ {
		si := swingIndexAt(open, high, low, close, i, limitMove)
		total += si
		out[i] = total
	}
	return out
}

// SwingIndex returns the one-bar Swing Index that [AccumulativeSwingIndex] accumulates.
//
// It is provided separately because the unaccumulated series is what a caller plots when they want to
// see which bars moved the line, and an accumulator hides that. The first value is 0 rather than NaN,
// matching the accumulation's seed, and a bar with no displacement contributes 0.
//
// The formula and the meaning of limitMove are documented on [AccumulativeSwingIndex].
func SwingIndex(open, high, low, close []float64, limitMove float64) []float64 {
	checkLimitMove("SwingIndex", limitMove)
	size := requireSameLen("SwingIndex", open, high, low, close)
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	for i := 1; i < size; i++ {
		out[i] = swingIndexAt(open, high, low, close, i, limitMove)
	}
	return out
}

// checkLimitMove panics unless the limit move is a positive, non-NaN number.
func checkLimitMove(name string, limitMove float64) {
	if limitMove <= 0 || limitMove != limitMove {
		panic("ta: " + name + " limitMove must be > 0")
	}
}
