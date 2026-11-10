package vec

// ---------------------------------------------------------------------------
// Scalar comparisons that reduce to a count.
//
// GreaterScalar followed by CountTrue materialises a mask for a question that
// only needs a number. CountGreaterScalar keeps the comparison and the count in
// one pass and allocates nothing.
// ---------------------------------------------------------------------------

// CountGreaterScalar returns the number of elements of xs strictly greater
// than k. NaN elements compare false, matching [GreaterScalar].
func CountGreaterScalar(xs []float64, k float64) int {
	count := 0
	for _, v := range xs {
		if v > k {
			count++
		}
	}
	return count
}

// CountGreaterEqualScalar returns the number of elements of xs greater than or
// equal to k. NaN elements compare false, matching [GreaterEqualScalar].
func CountGreaterEqualScalar(xs []float64, k float64) int {
	count := 0
	for _, v := range xs {
		if v >= k {
			count++
		}
	}
	return count
}
