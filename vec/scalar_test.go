package vec

import (
	"math"
	"testing"
)

func TestScalarKernels(t *testing.T) {
	xs := []float64{1, 2, 4}
	cases := []struct {
		name string
		got  []float64
		want []float64
	}{
		{"sub", SubScalar(xs, 1), []float64{0, 1, 3}},
		{"rsub", RSubScalar(xs, 1), []float64{0, -1, -3}},
		{"div", DivScalar(xs, 2), []float64{0.5, 1, 2}},
		{"rdiv", RDivScalar(xs, 8), []float64{8, 4, 2}},
	}
	for _, testCase := range cases {
		if len(testCase.got) != len(testCase.want) {
			t.Fatalf("%s: length = %d", testCase.name, len(testCase.got))
		}
		for i := range testCase.want {
			if testCase.got[i] != testCase.want[i] {
				t.Fatalf("%s[%d] = %v, want %v", testCase.name, i, testCase.got[i], testCase.want[i])
			}
		}
	}
}

func TestScalarKernelsEmpty(t *testing.T) {
	if SubScalar(nil, 1) != nil || RSubScalar(nil, 1) != nil ||
		DivScalar(nil, 1) != nil || RDivScalar(nil, 1) != nil {
		t.Fatal("empty input should return nil")
	}
}

func TestScalarKernelsAliasAndSource(t *testing.T) {
	src := []float64{1, 2, 3}
	dst := append([]float64(nil), src...)
	SubScalarTo(dst, dst, 1)
	if dst[0] != 0 || dst[2] != 2 {
		t.Fatalf("alias result = %v", dst)
	}
	if src[0] != 1 || src[2] != 3 {
		t.Fatalf("source mutated = %v", src)
	}
}

func TestScalarKernelsDivisionByZeroIsIEEE(t *testing.T) {
	if got := DivScalar([]float64{1}, 0); !math.IsInf(got[0], 1) {
		t.Fatalf("1/0 = %v, want +Inf", got[0])
	}
	if got := RDivScalar([]float64{0}, 1); !math.IsInf(got[0], 1) {
		t.Fatalf("1/0 = %v, want +Inf", got[0])
	}
}

func TestScalarKernelsPanicOnLengthMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a length panic")
		}
	}()
	SubScalarTo(make([]float64, 1), make([]float64, 2), 1)
}
