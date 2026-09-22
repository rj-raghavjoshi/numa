# numa

> **numa** is a high-throughput, hardware-aware numerical and statistical computing engine written in Go. 

Engineered for quantitative modeling, time-series analysis, and algorithmic trading systems where allocation overhead and floating-point stalls are unacceptable.

### Highlights

* **Architecture-Aware Execution:** Tailored loop pipelining and instruction-level parallelism (ILP) targeting ARM64 Neon (`LDP`) and x86-64 AVX pipelines.
* **Mechanical Sympathy:** Designed around 64-byte cache line alignment and register saturation to minimize CPU pipeline bubbles.
* **Numerically Stable:** Employs two-pass shifted variances, pairwise tree summations, and online algorithms (Welford's) to prevent catastrophic cancellation.
* **Zero Dependencies:** Pure Go with standard library testing—no external runtime bloat or CGo requirements.