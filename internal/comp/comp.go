// Package comp provides the compensated floating-point accumulation that both the batch
// kernels in vec and the streaming rollers in series depend on.
//
// It exists as an internal package rather than as a function in either of those packages
// because of a deliberate layering rule: series does not import vec, so that a rolling
// computation is never forced through a batch primitive it does not need. Rather than
// duplicate the arithmetic in both, the six lines live here and both import them -- an
// internal package that neither layer has to know about.
package comp

import "math"

// NeumaierAdd returns the new total and compensation for the sum total+comp+v.
//
// Kahan's original algorithm assumes the running total is larger in magnitude than the value
// being added; Neumaier's variant tests which is larger and chooses the loss-free formula
// accordingly, which makes it correct for the mixed-magnitude sequences a price or volume
// series produces.
//
// The returned compensation is not an error estimate to be discarded: it holds the low-order
// bits the total could not represent, and the sum of the two is the meaningful value. Callers
// report total+comp and keep accumulating into total.
func NeumaierAdd(total, comp, v float64) (newTotal, newComp float64) {
	t := total + v
	if math.Abs(total) >= math.Abs(v) {
		comp += (total - t) + v
	} else {
		comp += (v - t) + total
	}
	return t, comp
}
