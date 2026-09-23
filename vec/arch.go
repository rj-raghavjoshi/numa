package vec

// ---------------------------------------------------------------------------
// Architecture dispatch.
//
// Every reduction and elementwise operation in this package has three
// implementations, selected at compile time by build tags:
//
//	arch_arm64.go   //go:build arm64
//	arch_amd64.go   //go:build amd64
//	arch_generic.go //go:build !arm64 && !amd64
//
// They share this one file's worth of names. Each implementation is tuned for a
// different number of available floating-point registers, so the number of
// independent accumulators differs between them.
//
// See ../docs/design.md for why the loops are shaped the way they are, and
// ../docs/benchmarks.md for the measured payoff.
// ---------------------------------------------------------------------------
