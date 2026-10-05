package ta

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Benchmarks for the indicator layer.
//
// The most informative comparison here is Highest: the implementation uses a
// monotonic deque, O(1) amortised per element, and the naive alternative rescans the
// window, O(window). That is a genuine algorithmic difference rather than a tuning
// difference, so the gap should grow with the window and not just with n.
//
// The moving averages are measured without a naive baseline in most cases, because the
// implementation *is* the clearest form: SMA and EMA delegate to the series rollers,
// which have their own baselines and their own measured crossover in
// docs/benchmarks.md. WMA is the exception and gets a note, since it is deliberately
// O(window).
// ---------------------------------------------------------------------------

var sink float64

var benchSizes = []int{1 << 10, 1 << 15, 1 << 20}

func benchClose(n int) []float64 {
	xs := make([]float64, n)
	p := 100.0
	for i := range xs {
		p += float64((i%13)-6) * 0.05
		xs[i] = p
	}
	return xs
}

func benchOHLC(n int) (high, low, close []float64) {
	close = benchClose(n)
	high = make([]float64, n)
	low = make([]float64, n)
	for i := range close {
		high[i] = close[i] + 0.5
		low[i] = close[i] - 0.5
	}
	return
}

func BenchmarkSMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = SMA(close, 20)[n-1]
			}
		})
	}
}

func BenchmarkEMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = EMA(close, 20)[n-1]
			}
		})
	}
}

func BenchmarkWMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		// WMA is deliberately O(window) per element; two windows show that.
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = WMA(close, w)[n-1]
				}
			})
		}
	}
}

// naiveHighest rescans the window, which is what the deque exists to avoid.
func naiveHighest(xs []float64, n int) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		if i+1 < n {
			out[i] = 0
			continue
		}
		m := xs[i-n+1]
		for j := i - n + 2; j <= i; j++ {
			if xs[j] > m {
				m = xs[j]
			}
		}
		out[i] = m
	}
	return out
}

func BenchmarkHighest(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = Highest(close, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkNaiveHighest(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = naiveHighest(close, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkBollingerBands(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				_, _, lower := BollingerBands(close, 20, 2)
				sink = lower[n-1]
			}
		})
	}
}

func BenchmarkATR(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				sink = ATR(high, low, close, 14)[n-1]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Momentum indicators.
//
// Two of these have an O(window) inner loop that is worth seeing explicitly: CCI
// recomputes the mean deviation over the window at every bar, and CMO sums the gains
// and losses over the window at every bar. Both are kept because the centred
// formulation is the accurate one, and the numbers below are the price of that choice
// rather than an accident to be optimized away later without noticing the trade.
// ---------------------------------------------------------------------------

func BenchmarkRSI(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = RSI(close, 14)[n-1]
			}
		})
	}
}

func BenchmarkMACD(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				_, _, hist := MACD(close, 12, 26, 9)
				sink = hist[n-1]
			}
		})
	}
}

func BenchmarkStochastic(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				_, d := Stochastic(high, low, close, 14, 3, 3)
				sink = d[n-1]
			}
		})
	}
}

func BenchmarkCCI(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				sink = CCI(high, low, close, 20)[n-1]
			}
		})
	}
}

func BenchmarkCMO(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		// Two periods, because CMO's rolling sums are O(1) while the rescan it
		// replaced was O(window): the comparison is only meaningful across window
		// sizes.
		for _, period := range []int{14, 100} {
			b.Run(fmt.Sprintf("%d/period%d", n, period), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = CMO(close, period)[n-1]
				}
			})
		}
	}
}

func BenchmarkROC(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = ROC(close, 10)[n-1]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Volume indicators.
//
// The cumulative ones (VWAP, OBV, A/D) are single passes with a running total. The
// windowed ones (VWMA, CMF, MFI) use rolling sums and should be flat in their period,
// which is the point of building them that way rather than resumming.
// ---------------------------------------------------------------------------

func benchVolume(n int) []float64 {
	vs := make([]float64, n)
	for i := range vs {
		vs[i] = 1000 + float64(i%97)
	}
	return vs
}

func BenchmarkVWAP(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		volume := benchVolume(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				sink = VWAP(high, low, close, volume)[n-1]
			}
		})
	}
}

func BenchmarkVWMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		volume := benchVolume(n)
		for _, p := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/period%d", n, p), func(b *testing.B) {
				b.SetBytes(int64(16 * n))
				for i := 0; i < b.N; i++ {
					sink = VWMA(close, volume, p)[n-1]
				}
			})
		}
	}
}

func BenchmarkOBV(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		volume := benchVolume(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = OBV(close, volume)[n-1]
			}
		})
	}
}

func BenchmarkAccumulationDistribution(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		volume := benchVolume(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				sink = AccumulationDistribution(high, low, close, volume)[n-1]
			}
		})
	}
}

func BenchmarkChaikinMoneyFlow(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		volume := benchVolume(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				sink = ChaikinMoneyFlow(high, low, close, volume, 20)[n-1]
			}
		})
	}
}

func BenchmarkMoneyFlowIndex(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		volume := benchVolume(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				sink = MoneyFlowIndex(high, low, close, volume, 14)[n-1]
			}
		})
	}
}

func BenchmarkPriceVolumeTrend(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		volume := benchVolume(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = PriceVolumeTrend(close, volume)[n-1]
			}
		})
	}
}

func BenchmarkForceIndex(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		volume := benchVolume(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = ForceIndex(close, volume, 13)[n-1]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Trend indicators.
//
// Aroon is the interesting one here: it scans its window directly, O(window) per bar,
// where an index-tracking monotonic deque would make it O(1). The two-period benchmark
// measures exactly that gap so the follow-up has a number attached to it rather than an
// adjective.
//
// The recursive indicators (SuperTrend, PSAR) are inherently one multiply-add per bar
// plus their state updates, so their cost should be flat in every parameter -- which is
// worth confirming, since a hidden window loop would show up immediately.
// ---------------------------------------------------------------------------

func BenchmarkDirectionalMovement(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				adx, _, _ := DirectionalMovement(high, low, close, 14)
				sink = adx[n-1]
			}
		})
	}
}

func BenchmarkAroon(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		for _, p := range []int{14, 100} {
			b.Run(fmt.Sprintf("%d/period%d", n, p), func(b *testing.B) {
				b.SetBytes(int64(16 * n))
				for i := 0; i < b.N; i++ {
					up, _, _ := Aroon(high, low, p)
					sink = up[n-1]
				}
			})
		}
	}
}

func BenchmarkVortex(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				vp, _ := Vortex(high, low, close, 14)
				sink = vp[n-1]
			}
		})
	}
}

func BenchmarkChoppiness(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				sink = Choppiness(high, low, close, 14)[n-1]
			}
		})
	}
}

func BenchmarkKeltnerChannels(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				_, _, lower := KeltnerChannels(high, low, close, 20, 10, 2)
				sink = lower[n-1]
			}
		})
	}
}

func BenchmarkSuperTrend(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				line, _ := SuperTrend(high, low, close, 10, 3)
				sink = line[n-1]
			}
		})
	}
}

func BenchmarkParabolicSAR(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sar, _ := ParabolicSAR(high, low, 0.02, 0.02, 0.2)
				sink = sar[n-1]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Regression and composed oscillators.
//
// The regression family is O(window) per bar by design (see regression.go), so it is
// measured at two periods. The composed oscillators are measured at their defaults; the
// interesting question for them is only their absolute cost, since each is a small fixed
// number of passes over the data.
// ---------------------------------------------------------------------------

func BenchmarkLinearRegression(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		for _, p := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/period%d", n, p), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = LinearRegression(close, p)[n-1]
				}
			})
		}
	}
}

func BenchmarkStandardErrorBands(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				_, _, lower := StandardErrorBands(close, 20, 2)
				sink = lower[n-1]
			}
		})
	}
}

func BenchmarkAwesomeOscillator(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = AwesomeOscillator(high, low, 5, 34)[n-1]
			}
		})
	}
}

func BenchmarkTRIX(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = TRIX(close, 15)[n-1]
			}
		})
	}
}

func BenchmarkUltimateOscillator(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				sink = UltimateOscillator(high, low, close, 7, 14, 28)[n-1]
			}
		})
	}
}

func BenchmarkFisherTransform(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = FisherTransform(high, low, 9)[n-1]
			}
		})
	}
}

func BenchmarkMassIndex(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = MassIndex(high, low, 9, 25)[n-1]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Correlation, realized volatility, and the long-period oscillators.
//
// RankCorrelation is the one to watch here: it ranks each window independently, so it is
// O(n log n) per bar rather than O(n). The gap between it and the Pearson version on the same
// data is the measured cost of the ranks.
// ---------------------------------------------------------------------------

func BenchmarkCorrelationCoefficient(b *testing.B) {
	for _, n := range benchSizes {
		xs, ys := benchClose(n), benchClose(0)
		ys = benchClose(n)
		for i := range ys {
			ys[i] += float64(i%11) * 0.1
		}
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = CorrelationCoefficient(xs, ys, 20)[n-1]
			}
		})
	}
}

func BenchmarkRankCorrelation(b *testing.B) {
	for _, n := range benchSizes {
		xs := benchClose(n)
		ys := benchClose(n)
		for i := range ys {
			ys[i] += float64(i%11) * 0.1
		}
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = RankCorrelation(xs, ys, 20)[n-1]
			}
		})
	}
}

func BenchmarkHistoricalVolatility(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = HistoricalVolatility(close, 20, 252)[n-1]
			}
		})
	}
}

func BenchmarkVolatilityOHLC(b *testing.B) {
	for _, n := range benchSizes {
		open, high, low, close, _ := synthOHLCV(n, 1)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				sink = VolatilityOHLC(open, high, low, close, 20, 252)[n-1]
			}
		})
	}
}

func BenchmarkTrueStrengthIndex(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = TrueStrengthIndex(close, 25, 13)[n-1]
			}
		})
	}
}

func BenchmarkWilliamsAlligator(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				jaw, _, _ := WilliamsAlligator(high, low, 13, 8, 8, 5, 5, 3)
				sink = jaw[n-1]
			}
		})
	}
}

func BenchmarkWilliamsFractal(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				up, _ := WilliamsFractal(high, low, 2)
				sink = float64(up[n-1])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Ichimoku, pivots, GMMA and ZigZag.
//
// GMMA is the one to note: it is twelve independent EMA passes, so its cost is simply twelve
// times a single EMA. That is not a defect -- the indicator *is* twelve lines -- but it is a
// number a caller should know before asking for it on every bar. ZigZag, by contrast, is a
// single pass with O(1) state, so it should be among the cheapest indicators in the package.
// ---------------------------------------------------------------------------

func BenchmarkIchimoku(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				_, _, _, spanB, _ := Ichimoku(high, low, close, 9, 26, 52, 26)
				sink = spanB[n-1]
			}
		})
	}
}

func BenchmarkPivotPoints(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				pivot, _, _, _, _, _, _ := PivotPoints(high, low, close, PivotClassic)
				sink = pivot[n-1]
			}
		})
	}
}

func BenchmarkGMMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				short, _ := GMMA(close)
				sink = short[0][n-1]
			}
		})
	}
}

func BenchmarkZigZag(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				pivot, _ := ZigZag(high, low, 0.02)
				sink = pivot[n-1]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Extended moving averages, envelopes and the price oscillator.
//
// The adaptive averages are the interesting ones here: KAMA and VIDYA are recursive and their cost
// is a small constant per bar regardless of period, while ALMA and SWMA rescan a fixed window. The
// ALMA pair is measured at two periods to show it scaling, and the recursive trio to show it not.
// ---------------------------------------------------------------------------

func BenchmarkALMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		for _, period := range []int{9, 100} {
			b.Run(fmt.Sprintf("%d/period%d", n, period), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = ALMA(close, period, 0.85, 6)[n-1]
				}
			})
		}
	}
}

func BenchmarkSWMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = SWMA(close)[n-1]
			}
		})
	}
}

func BenchmarkKAMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = KAMA(close, 10, 2, 30)[n-1]
			}
		})
	}
}

func BenchmarkVIDYA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = VIDYA(close, 14, 9)[n-1]
			}
		})
	}
}

func BenchmarkZLEMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = ZLEMA(close, 20)[n-1]
			}
		})
	}
}

func BenchmarkT3(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = T3(close, 5, 0.7)[n-1]
			}
		})
	}
}

func BenchmarkMcGinleyDynamic(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = McGinleyDynamic(close, 14)[n-1]
			}
		})
	}
}

func BenchmarkMovingAverageEnvelope(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				_, _, lower := MovingAverageEnvelope(close, 20, 0.05)
				sink = lower[n-1]
			}
		})
	}
}

func BenchmarkMovingAverageChannel(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				upper, _ := MovingAverageChannel(high, low, 20)
				sink = upper[n-1]
			}
		})
	}
}

func BenchmarkPriceOscillator(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = PriceOscillator(close, 10, 30)[n-1]
			}
		})
	}
}

func BenchmarkWeightedClose(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				sink = WeightedClose(high, low, close)[n-1]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Second-tier momentum, volume, stop and range indicators.
//
// Most of these sit in the 20-50 ns/element band because they are compositions of two or three
// smoother passes, which is the pattern this file has now seen repeatedly: the indicator's own
// arithmetic is negligible next to the cost of the windows it drives. PercentRank is the exception
// measured here, because it rescans its window rather than accumulating one.
// ---------------------------------------------------------------------------

func BenchmarkPercentRank(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		for _, w := range []int{20, 200} {
			b.Run(fmt.Sprintf("%d/window%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = PercentRank(close, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkStochasticMomentumIndex(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				smi, _ := StochasticMomentumIndex(high, low, close, 10, 3, 3)
				sink = smi[n-1]
			}
		})
	}
}

func BenchmarkRelativeVolatilityIndex(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = RelativeVolatilityIndex(close, 14)[n-1]
			}
		})
	}
}

func BenchmarkConnorsRSI(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = ConnorsRSI(close, 3, 2, 100)[n-1]
			}
		})
	}
}

func BenchmarkElderRay(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				bull, _ := ElderRay(high, low, close, 13)
				sink = bull[n-1]
			}
		})
	}
}

func BenchmarkBalanceOfPower(b *testing.B) {
	for _, n := range benchSizes {
		open, high, low, close, _ := synthOHLCV(n, 1)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				sink = BalanceOfPower(open, high, low, close, 14)[n-1]
			}
		})
	}
}

func BenchmarkEaseOfMovement(b *testing.B) {
	for _, n := range benchSizes {
		_, high, low, _, volume := synthOHLCV(n, 1)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				sink = EaseOfMovement(high, low, volume, 14)[n-1]
			}
		})
	}
}

func BenchmarkKlingerOscillator(b *testing.B) {
	for _, n := range benchSizes {
		_, high, low, close, volume := synthOHLCV(n, 1)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				kvo, _ := KlingerOscillator(high, low, close, volume, 34, 55, 13)
				sink = kvo[n-1]
			}
		})
	}
}

func BenchmarkChaikinVolatility(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = ChaikinVolatility(high, low, 10)[n-1]
			}
		})
	}
}

func BenchmarkChandeKrollStop(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				long, _ := ChandeKrollStop(high, low, close, 10, 1, 9)
				sink = long[n-1]
			}
		})
	}
}

func BenchmarkFiftyTwoWeekHighLow(b *testing.B) {
	for _, n := range benchSizes {
		high, low, _ := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				h, _ := FiftyTwoWeekHighLow(high, low, 252)
				sink = h[n-1]
			}
		})
	}
}

func BenchmarkAccumulativeSwingIndex(b *testing.B) {
	for _, n := range benchSizes {
		open, high, low, close, _ := synthOHLCV(n, 1)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				sink = AccumulativeSwingIndex(open, high, low, close, 3)[n-1]
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Catalogue-completion indicators.
//
// Simple ratios and differences are elementwise and should be at the memory-bandwidth floor;
// HammingMA is O(window) with a fixed weight profile, the same shape as ALMA; the cross indicators
// are two average passes plus a sign comparison.
// ---------------------------------------------------------------------------

func BenchmarkStandardDeviation(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				sink = StandardDeviation(close, 20)[n-1]
			}
		})
	}
}

func BenchmarkRatio(b *testing.B) {
	for _, n := range benchSizes {
		a, c := benchClose(n), benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(16 * n))
			for i := 0; i < b.N; i++ {
				sink = Ratio(a, c)[n-1]
			}
		})
	}
}

func BenchmarkChopZone(b *testing.B) {
	for _, n := range benchSizes {
		high, low, close := benchOHLC(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(24 * n))
			for i := 0; i < b.N; i++ {
				sink = ChopZone(high, low, close, 14)[n-1]
			}
		})
	}
}

func BenchmarkHammingMA(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		for _, w := range []int{9, 100} {
			b.Run(fmt.Sprintf("%d/period%d", n, w), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				for i := 0; i < b.N; i++ {
					sink = HammingMA(close, w)[n-1]
				}
			})
		}
	}
}

func BenchmarkRelativeVigorIndex(b *testing.B) {
	for _, n := range benchSizes {
		open, high, low, close, _ := synthOHLCV(n, 1)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(32 * n))
			for i := 0; i < b.N; i++ {
				rvi, _ := RelativeVigorIndex(open, high, low, close, 10, 4)
				sink = rvi[n-1]
			}
		})
	}
}

func BenchmarkEMACross(b *testing.B) {
	for _, n := range benchSizes {
		close := benchClose(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(8 * n))
			for i := 0; i < b.N; i++ {
				_, _, cross := EMACross(close, 12, 26)
				sink = float64(cross[n-1])
			}
		})
	}
}
