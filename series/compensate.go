package series

import "github.com/rj-raghavjoshi/numa/internal/comp"

// ---------------------------------------------------------------------------
// Neumaier compensated addition, shared by every roller that accumulates a sum.
//
// The arithmetic itself lives in internal/comp, because the vec batch kernels need exactly
// the same routine and series deliberately does not import vec. See that package for the
// explanation of the algorithm and of what the compensation term means.
// ---------------------------------------------------------------------------

// neumaierAdd returns the new total and compensation for total+compensation+v.
func neumaierAdd(total, compensation, v float64) (newTotal, newComp float64) {
	return comp.NeumaierAdd(total, compensation, v)
}
