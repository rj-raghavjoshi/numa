package vec

import (
	"math"

	"github.com/rj-raghavjoshi/numa/internal/comp"
)

// ---------------------------------------------------------------------------
// Rolling-window batch kernels.
//
// Each of these takes a slice and a window and produces one output per input, where output i
// summarises the trailing window ending at i. The first n-1 outputs are NaN because the window
// is not yet full.
//
// # Two implementation shapes, chosen per operation
//
// **Running state (O(1) per element):** the sum, the mean and the extremes. A sum is updated by
// adding the arriving value and subtracting the one that left; an extreme is maintained with a
// monotonic deque. These are the same algorithms the series rollers use, and they are the ones
// that need an explicit NaN *counter*, because a NaN added to running state cannot be
// subtracted back out cleanly -- see the note on [RollingSumTo].
//
// **Window recomputation (O(window) per element):** variance, covariance, correlation, the
// weighted average and the order statistics. These need the window's own mean, or its sorted
// order, before they can produce anything, and the incremental forms that would avoid the
// second pass are exactly the numerically unstable ones this package rejects elsewhere. The
// cost is real and is measured in docs/benchmarks.md.
//
// Recomputation handles NaN for free -- a window containing a NaN sums to NaN and recovers when
// the NaN leaves -- which is why those functions carry no counter.
//
// # Allocation
//
// The allocating forms are the primary API. The `*To` forms exist for the operations whose state
// is a few registers; the recomputing ones need a scratch buffer per call and would need a
// second caller-supplied buffer to be allocation-free, which is a different signature shape.
// The series package remains the zero-allocation path for all of them.
// ---------------------------------------------------------------------------

// rollingWindowStart returns the first index at which a window of n is full.
func rollingWindowStart(n int) int { return n - 1 }

// RollingSum returns the trailing n-value sum at each position.
//
// The first n-1 values are NaN, and a window containing a NaN is NaN. The running sum is
// Neumaier-compensated, so it does not drift over a long series and does not lose small values
// when a large one leaves the window.
func RollingSum(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RollingSumTo(make([]float64, len(xs)), xs, n)
}

// RollingSumTo stores the trailing n-value sum into dst and returns dst.
//
// dst and xs must be the same length, and RollingSumTo panics otherwise, or if n < 1.
//
// **dst must not alias xs.** The loop reads xs[i-n] as the value leaving the window, and by then
// earlier iterations have written dst[i-n] over it. This is the same restriction the lagged
// operations in signal.go carry, and for the same cause; see the file header.
//
// # Why NaNs are counted rather than accumulated
//
// Adding a NaN to the compensated total poisons the compensation term permanently: its eviction
// is a subtraction of NaN, and no later addition clears it, so the sum would never recover even
// after every NaN had left the window. Counting them instead makes the poisoning last exactly
// as long as a NaN is inside the window.
func RollingSumTo(dst, xs []float64, n int) []float64 {
	checkRolling("RollingSumTo", dst, xs, n)
	size := len(dst)
	start := rollingWindowStart(n)
	fillNaN(dst, 0, start)

	var total, compensation float64
	nanCount := 0
	for i := 0; i < size; i++ {
		v := xs[i]
		if v != v {
			nanCount++
		} else {
			total, compensation = comp.NeumaierAdd(total, compensation, v)
		}
		if i >= n {
			old := xs[i-n]
			if old != old {
				nanCount--
			} else {
				total, compensation = comp.NeumaierAdd(total, compensation, -old)
			}
		}
		if i < start {
			continue
		}
		if nanCount > 0 {
			dst[i] = math.NaN()
			continue
		}
		dst[i] = total + compensation
	}
	return dst
}

// RollingMean returns the trailing n-value mean at each position.
//
// The first n-1 values are NaN, and a window containing a NaN is NaN.
func RollingMean(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RollingMeanTo(make([]float64, len(xs)), xs, n)
}

// RollingMeanTo stores the trailing n-value mean into dst and returns dst.
//
// dst and xs must be the same length, and RollingMeanTo panics otherwise, or if n < 1.
//
// **dst must not alias xs**; see [RollingSumTo].
func RollingMeanTo(dst, xs []float64, n int) []float64 {
	checkRolling("RollingMeanTo", dst, xs, n)
	RollingSumTo(dst, xs, n)
	inv := 1 / float64(n)
	for i := rollingWindowStart(n); i < len(dst); i++ {
		dst[i] *= inv
	}
	return dst
}

// RollingMax returns the trailing n-value maximum at each position.
//
// The first n-1 values are NaN, and a window containing a NaN is NaN. The window is maintained
// with a monotonic deque, so the cost is O(1) amortised per element rather than O(n).
func RollingMax(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RollingMaxTo(make([]float64, len(xs)), xs, n)
}

// RollingMaxTo stores the trailing n-value maximum into dst and returns dst.
//
// dst and xs must be the same length, and RollingMaxTo panics otherwise, or if n < 1.
//
// **dst must not alias xs**; see [RollingSumTo].
func RollingMaxTo(dst, xs []float64, n int) []float64 {
	checkRolling("RollingMaxTo", dst, xs, n)
	return rollingExtreme(dst, xs, n, true)
}

// RollingMin returns the trailing n-value minimum at each position.
//
// See [RollingMax] for the warm-up and NaN behaviour; the implementation differs only in the
// comparison direction.
func RollingMin(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RollingMinTo(make([]float64, len(xs)), xs, n)
}

// RollingMinTo stores the trailing n-value minimum into dst and returns dst.
//
// dst and xs must be the same length, and RollingMinTo panics otherwise, or if n < 1.
//
// **dst must not alias xs**; see [RollingSumTo].
func RollingMinTo(dst, xs []float64, n int) []float64 {
	checkRolling("RollingMinTo", dst, xs, n)
	return rollingExtreme(dst, xs, n, false)
}

// rollingExtreme maintains a monotonic deque of indices and writes the window extreme.
//
// For a maximum the deque holds values in decreasing order, so its front is the window's
// maximum; for a minimum the order is reversed. An index is pushed once and popped at most once,
// so the whole scan is linear.
//
// NaN is counted rather than compared: every comparison against NaN is false, so a NaN pushed
// onto the deque would block the back-popping of everything behind it and leave the deque
// non-monotonic. Counting makes the output NaN for exactly the windows that contain a NaN and
// lets the deque run only on data where its ordering guarantees hold.
func rollingExtreme(dst, xs []float64, n int, high bool) []float64 {
	size := len(dst)
	start := rollingWindowStart(n)
	fillNaN(dst, 0, start)

	dq := make([]int, 0, n+1)
	head := 0
	nanCount := 0

	for i := 0; i < size; i++ {
		v := xs[i]
		if v != v {
			nanCount++
		}
		if i >= n && xs[i-n] != xs[i-n] {
			nanCount--
		}

		if high {
			for len(dq) > head && xs[dq[len(dq)-1]] <= v {
				dq = dq[:len(dq)-1]
			}
		} else {
			for len(dq) > head && xs[dq[len(dq)-1]] >= v {
				dq = dq[:len(dq)-1]
			}
		}
		dq = append(dq, i)

		// Expire indices that have left the window. The element just appended is at index i,
		// and i <= i-n is false for n >= 1, so this cannot run off the end.
		for dq[head] <= i-n {
			head++
		}
		// Compact when more than half the backing array is dead, keeping the cost linear.
		if head > 0 && head*2 >= len(dq) {
			m := copy(dq, dq[head:])
			dq = dq[:m]
			head = 0
		}

		if i < start {
			continue
		}
		if nanCount > 0 {
			dst[i] = math.NaN()
			continue
		}
		dst[i] = xs[dq[head]]
	}
	return dst
}

// RollingRange returns the trailing n-value high-low range at each position.
//
// It is the difference between [RollingMax] and [RollingMin] and costs two deque passes; a
// caller who needs both extremes as well should note that there is no combined form, so the
// work is not shared.
func RollingRange(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RollingRangeTo(make([]float64, len(xs)), xs, n)
}

// RollingRangeTo stores the trailing n-value range into dst and returns dst.
//
// dst and xs must be the same length, and RollingRangeTo panics otherwise, or if n < 1.
//
// **dst must not alias xs**; see [RollingSumTo].
func RollingRangeTo(dst, xs []float64, n int) []float64 {
	checkRolling("RollingRangeTo", dst, xs, n)
	size := len(dst)
	// Compute the extremes into scratch, then difference them, because dst may alias xs and
	// both passes have to read the original data.
	hi := make([]float64, size)
	lo := make([]float64, size)
	rollingExtreme(hi, xs, n, true)
	rollingExtreme(lo, xs, n, false)
	for i := 0; i < size; i++ {
		dst[i] = hi[i] - lo[i]
	}
	return dst
}

// RollingWMA returns the trailing n-value weighted moving average at each position, with weights
// 1..n from the oldest value to the newest.
//
// The first n-1 values are NaN. It is O(n) per element, unlike [RollingSum] and [RollingMax];
// see the file header for why the incremental form is not used.
func RollingWMA(xs []float64, n int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	return RollingWMATo(make([]float64, len(xs)), xs, n)
}

// RollingWMATo stores the trailing n-value weighted average into dst and returns dst.
//
// dst and xs must be the same length, and RollingWMATo panics otherwise, or if n < 1.
//
// **dst must not alias xs**; see [RollingSumTo].
func RollingWMATo(dst, xs []float64, n int) []float64 {
	checkRolling("RollingWMATo", dst, xs, n)
	size := len(dst)
	start := rollingWindowStart(n)
	fillNaN(dst, 0, start)

	denom := float64(n) * float64(n+1) / 2
	for i := start; i < size; i++ {
		var num float64
		base := i - n + 1
		for j := 0; j < n; j++ {
			num += float64(j+1) * xs[base+j]
		}
		dst[i] = num / denom
	}
	return dst
}

// RollingVariance returns the trailing n-value population variance at each position.
//
// The first n-1 values are NaN. Each window is centred on its own mean before the deviations
// are accumulated, which is what makes the result stable; see the file header.
func RollingVariance(xs []float64, n int) []float64 {
	return rollingMoment1(xs, n, 1)
}

// RollingVarianceSample returns the trailing n-value sample variance, dividing by n-1.
//
// It returns NaN for every position when n < 2, since the sample variance needs a degree of
// freedom.
func RollingVarianceSample(xs []float64, n int) []float64 {
	return rollingMoment1(xs, n, -1)
}

// RollingStdDev returns the trailing n-value population standard deviation.
func RollingStdDev(xs []float64, n int) []float64 {
	out := rollingMoment1(xs, n, 1)
	for i := range out {
		out[i] = math.Sqrt(out[i])
	}
	return out
}

// RollingStdDevSample returns the trailing n-value sample standard deviation.
func RollingStdDevSample(xs []float64, n int) []float64 {
	out := rollingMoment1(xs, n, -1)
	for i := range out {
		out[i] = math.Sqrt(out[i])
	}
	return out
}

// rollingMoment1 computes a one-series second central moment. divisor selects population (1,
// dividing by n) or sample (-1, dividing by n-1).
func rollingMoment1(xs []float64, n int, divisor int) []float64 {
	if n < 1 {
		panic("vec: rolling period must be >= 1")
	}
	if len(xs) == 0 {
		return nil
	}
	size := len(xs)
	out := make([]float64, size)
	fillNaN(out, 0, size)
	if divisor < 0 && n < 2 {
		// The sample variance needs a degree of freedom, so no window is defined.
		return out
	}
	fillNaN(out, 0, rollingWindowStart(n))

	d := float64(n)
	if divisor < 0 {
		d = float64(n - 1)
	}
	for i := rollingWindowStart(n); i < size; i++ {
		base := i - n + 1
		var sum float64
		for j := 0; j < n; j++ {
			sum += xs[base+j]
		}
		mean := sum / float64(n)
		var ss float64
		for j := 0; j < n; j++ {
			dev := xs[base+j] - mean
			ss += dev * dev
		}
		out[i] = ss / d
	}
	return out
}

// RollingCovariance returns the trailing n-value population covariance of xs and ys at each
// position, over min(len(xs), len(ys)) elements.
//
// The first n-1 values are NaN.
func RollingCovariance(xs, ys []float64, n int) []float64 {
	return rollingMoment2(xs, ys, n, 1)
}

// RollingCovarianceSample returns the trailing n-value sample covariance, dividing by n-1.
func RollingCovarianceSample(xs, ys []float64, n int) []float64 {
	return rollingMoment2(xs, ys, n, -1)
}

// rollingMoment2 computes a two-series co-moment. divisor selects population (1) or sample (-1).
func rollingMoment2(xs, ys []float64, n int, divisor int) []float64 {
	checkPeriodRolling("rolling covariance", n)
	size := min(len(xs), len(ys))
	if size == 0 {
		return nil
	}
	if n < 2 && divisor < 0 {
		out := make([]float64, size)
		fillNaN(out, 0, size)
		return out
	}

	out := make([]float64, size)
	fillNaN(out, 0, rollingWindowStart(n))
	d := float64(n)
	if divisor < 0 {
		d = float64(n - 1)
	}

	for i := rollingWindowStart(n); i < size; i++ {
		base := i - n + 1
		var sx, sy float64
		for j := 0; j < n; j++ {
			sx += xs[base+j]
			sy += ys[base+j]
		}
		mx, my := sx/float64(n), sy/float64(n)
		var sxy float64
		for j := 0; j < n; j++ {
			sxy += (xs[base+j] - mx) * (ys[base+j] - my)
		}
		out[i] = sxy / d
	}
	return out
}

// RollingCorrelation returns the trailing n-value Pearson correlation of xs and ys at each
// position.
//
// The result lies in [-1, 1] where it is defined. A window in which either series has zero
// variance yields NaN, matching [Correlation], rather than 0: "uncorrelated" and "undefined" are
// different claims. The first n-1 values are NaN.
func RollingCorrelation(xs, ys []float64, n int) []float64 {
	checkPeriodRolling("RollingCorrelation", n)
	size := min(len(xs), len(ys))
	if size == 0 {
		return nil
	}
	out := make([]float64, size)
	// A zero-variance window is left as NaN, so the whole output starts NaN.
	fillNaN(out, 0, size)

	for i := rollingWindowStart(n); i < size; i++ {
		base := i - n + 1
		var sx, sy float64
		for j := 0; j < n; j++ {
			sx += xs[base+j]
			sy += ys[base+j]
		}
		mx, my := sx/float64(n), sy/float64(n)
		var sxy, sxx, syy float64
		for j := 0; j < n; j++ {
			dx := xs[base+j] - mx
			dy := ys[base+j] - my
			sxy += dx * dy
			sxx += dx * dx
			syy += dy * dy
		}
		if sxx == 0 || syy == 0 {
			continue
		}
		out[i] = sxy / math.Sqrt(sxx*syy)
	}
	return out
}

// checkRolling validates the shared preconditions of the rolling kernels: equal lengths and a
// positive window.
func checkRolling(name string, dst, xs []float64, n int) {
	if len(dst) != len(xs) {
		panic("vec: " + name + " length mismatch")
	}
	if n < 1 {
		panic("vec: " + name + " period must be >= 1")
	}
}

// checkPeriodRolling validates only the window.
func checkPeriodRolling(name string, n int) {
	if n < 1 {
		panic("vec: " + name + " period must be >= 1")
	}
}
