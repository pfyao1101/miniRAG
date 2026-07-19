package vector

import (
	"math"
)

func CosineSimilarity(a, b []float32) (float32, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, ErrEmptyVector
	}
	if len(a) != len(b) {
		return 0, ErrDimensionMismatch
	}
	var dot float64
	var normSquaredA float64
	var normSquaredB float64

	// 中间结果 float64 降低误差
	// 输出结果 float32 节省内存
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])

		dot += av * bv
		normSquaredA += av * av
		normSquaredB += bv * bv
	}

	if normSquaredA == 0 || normSquaredB == 0 {
		return 0, ErrZeroVector
	}

	similarity := dot / math.Sqrt(normSquaredA*normSquaredB)
	return float32(similarity), nil
}
