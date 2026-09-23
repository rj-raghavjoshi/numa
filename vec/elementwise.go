package vec

// ---------------------------------------------------------------------------
// Elementwise operations: walk two slices, write a third.
//
// These have no loop-carried dependency at all -- dst[i] depends only on xs[i] and
// ys[i] -- so the compiler was already free to overlap the work, and there was
// never a stall to eliminate. The tuned versions gain only ~1.3x, because the loops
// were never the bottleneck; memory bandwidth was. The tuning here is therefore
// about keeping the load/store units saturated rather than about shortening a
// dependency chain.
//
// That contrast is the single most useful thing to understand about this package:
// the identical tuning strategy buys ~4x on the reductions and ~1.3x here, purely
// because of what each loop is waiting on.
//
// Each operation comes in two forms:
//
//   - Allocating (Add, Sub, Mul, Scale, AddScalar, Abs): returns a new slice.
//   - In place (AddTo, SubTo, MulTo, ScaleTo, AddScalarTo, AbsTo): writes into a
//     caller-supplied destination, avoiding allocation in hot loops.
//
// The *To forms require equal lengths and panic otherwise, and permit dst to alias
// the inputs.
// ---------------------------------------------------------------------------

// checkSameLen panics unless dst, xs and ys all have the same length.
func checkSameLen(op string, dst, xs, ys []float64) {
	if len(dst) != len(xs) || len(dst) != len(ys) {
		panic("vec: " + op + " length mismatch")
	}
}

// checkSameLen1 panics unless dst and xs have the same length.
func checkSameLen1(op string, dst, xs []float64) {
	if len(dst) != len(xs) {
		panic("vec: " + op + " length mismatch")
	}
}
