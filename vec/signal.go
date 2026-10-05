package vec

import "math"

// ---------------------------------------------------------------------------
// Lag, difference, and signal primitives.
//
// These are the operations that compare an element with another element at a
// fixed offset. They fall into two families with different performance shapes:
//
//   - Lagged maps (Shift, Diff, Rate, Rising, Falling, CrossOver, CrossUnder,
//     Cross). No loop-carried dependency: output i depends on xs[i] and xs[i-lag].
//     They read two streams and are memory-bound. There is no arch variant. Diff,
//     Rate and the masks use the four-wide unroll (1.49x for Diff); Shift does not,
//     because it is pure movement and the built-in copy beats any hand loop
//     (2.28x). See [ShiftTo].
//
//   - Runs (BarsSince, ValueWhen, HighestSince, LowestSince). A loop-carried
//     value that resets on a condition. Sequential, not blocked: the reset makes
//     the carry position-dependent, so there is no block-prefix form.
//
// # Aliasing: the second documented exception
//
// The lagged operations here do NOT permit dst to alias xs. ReverseTo was the
// first exception; this is a family of them, for a related reason.
//
// The problem is concrete. At iteration i, the loop reads xs[i-lag]. If dst and xs
// share storage, then iteration i-lag already overwrote that element with an
// output, so the read returns the wrong value. Reading a whole block before writing
// it does not help either: the needed element was written several iterations ago.
//
// An alias-safe version would have to buffer the input, which buys safety by paying
// an allocation or a copy that a caller who simply passes two slices never pays.
// The honest choice is to document the restriction. Every *To form here states it.
//
// # Lag convention
//
// A positive lag looks backward in time, matching Pine's `x[n]`: `Shift(xs, 1)`
// yields yesterday's value today. A negative lag looks forward. Out-of-range
// positions are NaN, never zero, so a caller can tell "not available" from "the
// value was zero".
// ---------------------------------------------------------------------------

// lagRange returns the half-open interval [start, end) of output indices for which
// both i and i-lag are inside [0, n). Outside it the result is NaN.
func lagRange(n, lag int) (start, end int) {
	switch {
	case lag > 0:
		if lag > n {
			lag = n
		}
		return lag, n
	case lag < 0:
		return 0, n + lag // end may be <= 0, which the loops handle as empty
	default:
		return 0, n
	}
}

// fillNaN sets dst[lo:hi] to NaN, clamped to the slice.
func fillNaN(dst []float64, lo, hi int) {
	if lo < 0 {
		lo = 0
	}
	if hi > len(dst) {
		hi = len(dst)
	}
	for i := lo; i < hi; i++ {
		dst[i] = math.NaN()
	}
}

// Shift returns xs displaced by lag bars.
//
// A positive lag moves values later in time, so out[i] is xs[i-lag] and the first
// lag outputs are NaN. A negative lag moves them earlier. Shift(xs, 0) returns a
// copy.
//
// Shift allocates; for hot loops that can supply a buffer, use [ShiftTo].
func Shift(xs []float64, lag int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return ShiftTo(make([]float64, len(xs)), xs, lag)
}

// ShiftTo stores xs displaced by lag bars into dst and returns dst.
//
// dst and xs must be the same length. ShiftTo panics if they are not.
//
// dst must NOT alias xs; see the file header for why. Passing the same slice
// produces silently wrong results, because the NaN prefix is written before the
// body is copied.
//
// # Implementation
//
// Shift is pure data movement, so the in-range region is moved with the built-in
// copy, which lowers to the runtime's SIMD memmove. Hand-unrolling it is measurably
// worse, and so is the plain loop: at n=4M this reaches **57.4 GB/s**, against
// 33.7 GB/s for the four-wide unrolled loop and 25.2 GB/s for the naive loop. This
// is the one place in the package where the tuned implementation is a library call
// rather than a loop, because the library call *is* the tuned implementation. See
// docs/benchmarks.md.
//
// The first benchmark of the naive loop appeared *faster* than the unrolled version,
// and the reason is worth recording: with a compile-time constant lag of 1 the
// compiler recognises the naive loop as a shift and substitutes a memmove, so it was
// being compared against memmove rather than against a real loop. Passing the lag as
// a runtime value -- which is what this API does, and what the corrected benchmark
// does -- removes that recognition. The lesson is that a benchmark can accidentally
// measure the compiler's special case instead of the code under test.
func ShiftTo(dst, xs []float64, lag int) []float64 {
	checkSameLen1("ShiftTo", dst, xs)
	n := len(dst)
	if lag == 0 {
		copy(dst, xs)
		return dst
	}
	start, end := lagRange(n, lag)
	// Fill the out-of-range region, which is a prefix for lag > 0 and a suffix for
	// lag < 0.
	if lag > 0 {
		fillNaN(dst, 0, start)
	} else {
		fillNaN(dst, end, n)
	}
	if end > start {
		copy(dst[start:end], xs[start-lag:end-lag])
	}
	return dst
}

// Diff returns the change in xs over lag bars: out[i] is xs[i] - xs[i-lag].
//
// A positive lag is a backward difference (the usual "change since n bars ago");
// a negative lag is a forward difference. Positions without lag bars of history are
// NaN. Diff(xs, 0) is all zeros, since the change over no bars is exactly zero.
//
// Diff allocates; for hot loops that can supply a buffer, use [DiffTo].
func Diff(xs []float64, lag int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return DiffTo(make([]float64, len(xs)), xs, lag)
}

// DiffTo stores the lagged difference into dst and returns dst.
//
// dst and xs must be the same length. DiffTo panics if they are not.
//
// dst must NOT alias xs; see the file header.
func DiffTo(dst, xs []float64, lag int) []float64 {
	checkSameLen1("DiffTo", dst, xs)
	n := len(dst)
	if lag == 0 {
		for i := range dst {
			dst[i] = 0
		}
		return dst
	}
	start, end := lagRange(n, lag)
	if lag > 0 {
		fillNaN(dst, 0, start)
	} else {
		fillNaN(dst, end, n)
	}

	i := start
	limit := end - 3
	for i < limit {
		dst[i] = xs[i] - xs[i-lag]
		dst[i+1] = xs[i+1] - xs[i+1-lag]
		dst[i+2] = xs[i+2] - xs[i+2-lag]
		dst[i+3] = xs[i+3] - xs[i+3-lag]
		i += 4
	}
	for ; i < end; i++ {
		dst[i] = xs[i] - xs[i-lag]
	}
	return dst
}

// Rate returns the simple rate of change of xs over n bars:
// (xs[i] - xs[i-n]) / xs[i-n].
//
// This is the fraction that a rate-of-change indicator multiplies by 100. A zero
// denominator yields 0 rather than an infinity, following [SafeDiv], because a
// flat or absent base is routine in a price series and an infinity would poison
// every downstream value. Positions without n bars of history are NaN, so
// "not enough history" stays distinguishable from "no change".
//
// Rate panics if n < 1: a rate of change over zero bars is not a rate.
//
// Rate allocates; for hot loops that can supply a buffer, use [RateTo].
func Rate(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RateTo(make([]float64, len(xs)), xs, n)
}

// RateTo stores the rate of change over n bars into dst and returns dst.
//
// dst and xs must be the same length. RateTo panics if they are not, or if n < 1.
//
// dst must NOT alias xs; see the file header.
func RateTo(dst, xs []float64, n int) []float64 {
	checkSameLen1("RateTo", dst, xs)
	if n < 1 {
		panic("vec: RateTo lag must be >= 1")
	}
	size := len(dst)
	if n >= size {
		fillNaN(dst, 0, size)
		return dst
	}
	fillNaN(dst, 0, n)

	i := n
	limit := size - 3
	for i < limit {
		dst[i] = safeRate(xs[i], xs[i-n])
		dst[i+1] = safeRate(xs[i+1], xs[i+1-n])
		dst[i+2] = safeRate(xs[i+2], xs[i+2-n])
		dst[i+3] = safeRate(xs[i+3], xs[i+3-n])
		i += 4
	}
	for ; i < size; i++ {
		dst[i] = safeRate(xs[i], xs[i-n])
	}
	return dst
}

// safeRate is (cur-base)/base with a zero base mapped to 0.
func safeRate(cur, base float64) float64 {
	if base == 0 {
		return 0
	}
	return (cur - base) / base
}

// Rising returns a mask that is 1 where xs[i] > xs[i-n], over n bars.
//
// Positions without n bars of history are 0, not 1: an unknown comparison is not a
// rising one. RateTo-style NaN handling does not apply because the result is a
// mask; see the package's predicate documentation for why NaN never satisfies an
// ordering.
//
// Rising panics if n < 1.
func Rising(xs []float64, n int) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return risingFalling(xs, n, true)
}

// Falling returns a mask that is 1 where xs[i] < xs[i-n], over n bars.
//
// Positions without n bars of history are 0. Falling panics if n < 1.
func Falling(xs []float64, n int) []uint8 {
	if len(xs) == 0 {
		return nil
	}
	return risingFalling(xs, n, false)
}

func risingFalling(xs []float64, n int, rising bool) []uint8 {
	if n < 1 {
		panic("vec: Rising/Falling lag must be >= 1")
	}
	out := make([]uint8, len(xs))
	if n >= len(xs) {
		return out
	}
	i := n
	if rising {
		for ; i < len(xs); i++ {
			out[i] = boolToU8(xs[i] > xs[i-n])
		}
	} else {
		for ; i < len(xs); i++ {
			out[i] = boolToU8(xs[i] < xs[i-n])
		}
	}
	return out
}

// CrossOver returns a mask that is 1 at each index where a crosses above b: it is
// true when a[i-1] <= b[i-1] and a[i] > b[i].
//
// Index 0 is always 0, because a cross needs a previous bar. NaN never satisfies an
// ordering, so a NaN on either side suppresses the signal rather than inventing one.
func CrossOver(a, b []float64) []uint8 {
	n := min(len(a), len(b))
	if n == 0 {
		return nil
	}
	return crossOverUnder(a[:n], b[:n], true)
}

// CrossUnder returns a mask that is 1 where a crosses below b: a[i-1] >= b[i-1] and
// a[i] < b[i]. Index 0 is always 0.
func CrossUnder(a, b []float64) []uint8 {
	n := min(len(a), len(b))
	if n == 0 {
		return nil
	}
	return crossOverUnder(a[:n], b[:n], false)
}

func crossOverUnder(a, b []float64, over bool) []uint8 {
	out := make([]uint8, len(a))
	if over {
		for i := 1; i < len(a); i++ {
			out[i] = boolToU8(a[i-1] <= b[i-1] && a[i] > b[i])
		}
	} else {
		for i := 1; i < len(a); i++ {
			out[i] = boolToU8(a[i-1] >= b[i-1] && a[i] < b[i])
		}
	}
	return out
}

// Cross returns +1 where a crosses above b, -1 where it crosses below, and 0
// elsewhere. Index 0 is always 0.
//
// It is the signed form of [CrossOver] and [CrossUnder], for callers that want one
// pass and one buffer rather than two masks.
func Cross(a, b []float64) []int8 {
	n := min(len(a), len(b))
	if n == 0 {
		return nil
	}
	out := make([]int8, n)
	for i := 1; i < n; i++ {
		switch {
		case a[i-1] <= b[i-1] && a[i] > b[i]:
			out[i] = 1
		case a[i-1] >= b[i-1] && a[i] < b[i]:
			out[i] = -1
		}
	}
	return out
}

// BarsSince returns, for each index, the number of bars since mask was last
// non-zero.
//
// The value is 0 at an index where mask is non-zero, 1 at the index after one, and
// so on. Before the first non-zero entry there is no answer, and the value is -1
// rather than a large number that could be mistaken for a real distance.
//
// BarsSince returns nil for an empty input.
func BarsSince(mask []uint8) []int {
	if len(mask) == 0 {
		return nil
	}
	out := make([]int, len(mask))
	last := -1
	for i, m := range mask {
		if m != 0 {
			last = i
		}
		if last < 0 {
			out[i] = -1
		} else {
			out[i] = i - last
		}
	}
	return out
}

// ValueWhen returns, at each index, the most recent value of xs at an index where
// cond was non-zero, or NaN if cond has not yet been true.
//
// This is a forward-fill of the selected values, which is what a signal handler
// usually wants: "the price at the last entry signal", available on every later
// bar.
//
// It is deliberately NOT Pine's `ta.valuewhen`, which searches backward for the
// Nth most recent occurrence. That operation is not incremental, so it cannot be
// computed in one forward pass, and the forward-fill is the form that composes.
//
// ValueWhen returns nil for an empty input, and panics if the lengths differ.
func ValueWhen(cond []uint8, xs []float64) []float64 {
	if len(cond) != len(xs) {
		panic("vec: ValueWhen length mismatch")
	}
	if len(cond) == 0 {
		return nil
	}
	out := make([]float64, len(cond))
	have := false
	var last float64
	for i, c := range cond {
		if c != 0 {
			last = xs[i]
			have = true
		}
		if have {
			out[i] = last
		} else {
			out[i] = math.NaN()
		}
	}
	return out
}

// HighestSince returns the running maximum of xs since the last index where cond
// was non-zero, inclusive of that index.
//
// The reset at a non-zero cond means the value is the maximum over the current
// "episode". Before the first non-zero cond the result is NaN, since no episode has
// begun.
//
// NaN wins within an episode: a NaN encountered after the last reset makes the
// result NaN until the next reset. An infinity is a value and persists.
//
// HighestSince returns nil for an empty input, and panics if the lengths differ.
func HighestSince(cond []uint8, xs []float64) []float64 {
	return extremeSince(cond, xs, true)
}

// LowestSince returns the running minimum of xs since the last index where cond was
// non-zero, inclusive. See [HighestSince] for the reset and NaN semantics.
func LowestSince(cond []uint8, xs []float64) []float64 {
	return extremeSince(cond, xs, false)
}

func extremeSince(cond []uint8, xs []float64, highest bool) []float64 {
	if len(cond) != len(xs) {
		panic("vec: HighestSince/LowestSince length mismatch")
	}
	if len(cond) == 0 {
		return nil
	}
	out := make([]float64, len(cond))
	active := false
	var m float64
	for i, c := range cond {
		if c != 0 {
			active = true
			m = xs[i]
		} else if active {
			v := xs[i]
			if highest {
				if v != v {
					m = math.NaN()
				} else if v > m {
					m = v
				}
			} else {
				if v != v {
					m = math.NaN()
				} else if v < m {
					m = v
				}
			}
		}
		if active {
			out[i] = m
		} else {
			out[i] = math.NaN()
		}
	}
	return out
}
