package ta

import (
	"math"

	"github.com/rj-raghavjoshi/numa/series"
	"github.com/rj-raghavjoshi/numa/vec"
)

// ---------------------------------------------------------------------------
// Shared helpers.
//
// These encode the conventions the package documentation states, so that each
// indicator file contains the mathematics rather than a re-derivation of the same
// length checks and NaN padding.
// ---------------------------------------------------------------------------

// FirstValid returns the index of the first element of xs that is not NaN, or -1 if
// xs is empty or entirely NaN.
//
// It is how a caller discovers an indicator's warm-up length at runtime, rather than
// keeping a table that has to stay in step with the implementations:
//
//	out := ta.SMA(close, 20)
//	first := ta.FirstValid(out) // 19
func FirstValid(xs []float64) int {
	for i, v := range xs {
		if v == v {
			return i
		}
	}
	return -1
}

// fillNaN sets out[lo:hi] to NaN, clamped to the slice bounds.
func fillNaN(out []float64, lo, hi int) {
	if lo < 0 {
		lo = 0
	}
	if hi > len(out) {
		hi = len(out)
	}
	for i := lo; i < hi; i++ {
		out[i] = math.NaN()
	}
}

// allNaN returns a slice of length n filled with NaN.
func allNaN(n int) []float64 {
	out := make([]float64, n)
	fillNaN(out, 0, n)
	return out
}

// checkPeriod panics unless n >= 1.
//
// A non-positive period is a programming error and is reported as one. A period
// larger than the input is not an error: the caller gets an all-NaN result, which is
// the honest answer and is checked by the boundary tests rather than by a panic.
func checkPeriod(name string, n int) {
	if n < 1 {
		panic("ta: " + name + " period must be >= 1")
	}
}

// requireSameLen panics unless every slice has the same length as the first, and
// returns that length.
//
// A length mismatch between open, high, low and close is always a bug, and silently
// using min(len) -- as the vec elementwise functions do, because they are defined on
// sequences -- would hide it in a way that produces plausible wrong indicators.
func requireSameLen(name string, slices ...[]float64) int {
	if len(slices) == 0 {
		return 0
	}
	n := len(slices[0])
	for _, s := range slices[1:] {
		if len(s) != n {
			panic("ta: " + name + " length mismatch")
		}
	}
	return n
}

// applyFrom runs r over xs[from:], returning a slice the length of xs with NaN before
// from.
//
// It exists for composing indicators. A seeded smoother consumes the leading NaNs of
// its input as seed values, so feeding it the raw output of another indicator would
// poison every result; running it from the first valid index instead gives the
// composition the behaviour a caller expects. DEMA and TEMA are the immediate users.
func applyFrom(xs []float64, from int, r series.Roller) []float64 {
	out := make([]float64, len(xs))
	if from < 0 {
		from = 0
	}
	fillNaN(out, 0, from)
	if from < len(xs) {
		series.ApplyTo(out[from:], xs[from:], r)
	}
	return out
}

// applyEMA runs the batch exponential smoother over xs[from:], returning a slice the length of xs
// with NaN before from.
//
// It is the batch counterpart of applyFrom with a series EMA roller, and it exists so the composite
// indicators do not pay the streaming roller's per-element cost. The two must agree exactly -- they
// share the seeding convention and the compensation -- which is what the equivalence tests in the
// root package check.
func applyEMA(xs []float64, from, n int) []float64 {
	return applyBatchSmoother(xs, from, n, vec.RollingEMA)
}

// applyRMA is applyEMA for Wilder's smoothing.
func applyRMA(xs []float64, from, n int) []float64 {
	return applyBatchSmoother(xs, from, n, vec.RollingRMA)
}

// applyBatchSmoother splices a whole-slice smoother over xs[from:].
func applyBatchSmoother(xs []float64, from, n int, kernel func([]float64, int) []float64) []float64 {
	out := make([]float64, len(xs))
	if from < 0 {
		from = 0
	}
	// The warm-up region must be filled explicitly. `make` zeroes it, and a zero is a *value*: a
	// caller reading out[0] would see an average of 0 rather than "not yet defined", and anything
	// composed on top would treat it as data. That is not hypothetical -- it made Klinger's
	// oscillator report a defined value at index 0, which then seeded the signal line one step
	// earlier and made two different signal periods produce the same series.
	fillNaN(out, 0, min(from, len(out)))
	if from < len(xs) {
		copy(out[from:], kernel(xs[from:], n))
	}
	return out
}

// smaRef computes the mean of xs[lo:hi], used by the warm-up seeding of the composite
// averages.
func smaRef(xs []float64, lo, hi int) float64 {
	var s float64
	for i := lo; i < hi; i++ {
		s += xs[i]
	}
	return s / float64(hi-lo)
}

// populationStdevWindow returns the population standard deviation of xs[base:base+n]
// about the given mean, taking the deviations from the mean directly rather than from a
// sum of squares.
//
// Sharing it matters because the *choice* is the important part: computing a window's
// spread from `mean(x^2) - mean(x)^2` is the unstable form that vec/stats.go rejects, and
// a second implementation elsewhere could quietly reintroduce it. Every windowed spread in
// this package goes through here.
func populationStdevWindow(xs []float64, base, n int, mean float64) float64 {
	var ss float64
	for j := 0; j < n; j++ {
		d := xs[base+j] - mean
		ss += d * d
	}
	return math.Sqrt(ss / float64(n))
}

// logReturns returns the natural logarithm of the one-bar ratio, with NaN at index 0 and
// wherever the previous value is zero.
//
// A negative ratio produces NaN through the logarithm itself rather than through a check,
// which is the honest propagation: a negative price ratio is not a return.
func logReturns(xs []float64) []float64 {
	out := make([]float64, len(xs))
	if len(xs) == 0 {
		return out
	}
	out[0] = math.NaN()
	for i := 1; i < len(xs); i++ {
		if xs[i-1] == 0 {
			out[i] = math.NaN()
			continue
		}
		out[i] = math.Log(xs[i] / xs[i-1])
	}
	return out
}
