package vec

// # NaN policy
//
// [Min], [Max] and [MinMax] return NaN if any element of the input is NaN. This is
// the "NaN wins" policy, chosen deliberately over "ignore NaN".
//
// The reasoning: an array containing NaN is almost always the result of an
// upstream error (a 0/0, an inf-inf, a failed parse). Silently returning the
// extreme of the *remaining* values would hide that error and produce a
// plausible-looking number, which is worse than propagating it in a domain where
// the numbers feed decisions. Failing loudly is the safer default.
//
// The alternative is not merely a different opinion, it is worse in a specific
// way: a naive scan's behaviour is *positional*. Since `v < m` is false for NaN, a
// NaN at index 0 gets overwritten by later comparisons while a NaN at the last
// index gets returned. Neither policy justifies a result that depends on where the
// bad value sits, which is why the check exists rather than being left implicit.
//
// If you want NaN-skipping semantics, filter the input first with a predicate.
// That keeps the policy visible at the call site rather than buried in the
// reduction.
//
// # The check is fused into the scan loop
//
// The NaN test is part of the scan loop, not a separate pass. This matters more
// than it sounds.
//
// Measured on arm64 at n=4M:
//
//	naive one-accumulator loop      4.72 GB/s
//	scan, no NaN check at all       9.51 GB/s
//	scan, fused NaN check           9.43 GB/s
//	scan + separate NaN pass        4.68 GB/s
//
// A separate pass halves throughput and cancels the entire benefit of the tuned
// scan. Fusing costs about 1%.
//
// The reason fusion is nearly free: the NaN test accumulates an integer flag,
// which the CPU can evaluate on different execution ports than the
// floating-point compare. The scan is latency-bound on the *compare* chain, and
// the flag does not lengthen that chain. A second pass, by contrast, doubles the
// memory traffic and re-walks the latency-bound loop.
//
// # Why the obvious arithmetic trick does not work
//
// The natural branch-free NaN test is to accumulate a poison value:
//
//	poison += v - v     // 0 for finite v, NaN for NaN
//
// For finite values this adds 0. For NaN it adds NaN, which contaminates every
// later addition, so `poison != poison` at the end detects it -- no compare
// needed. It is branch-free, so it ought to pipeline like ordinary arithmetic.
//
// It does not work, for a reason worth recording so nobody retries it:
// **`Inf - Inf` is NaN too.** So is `Inf * 0`. Any array containing an infinity
// is therefore falsely poisoned, and there is no branch-free float expression that
// isolates NaN from infinity. The compare is unavoidable, and since a compare is
// unavoidable, folding it into the existing loop is the cheapest place to put it.
//
// This was verified by probing each candidate expression against finite, signed
// zero, NaN, +Inf and -Inf inputs rather than by reasoning about it.
//
// # Implementation note
//
// Each scan keeps one NaN flag *per accumulator* rather than a single shared flag.
// A shared flag would be written and read every iteration, making it a
// loop-carried dependency of its own and defeating the purpose of having
// independent accumulator chains. The flags are combined once, after the loop.
