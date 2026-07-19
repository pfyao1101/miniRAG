package vector

import (
	"errors"
	"math"
	"slices"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	const tolerance = 1e-6

	tests := []struct {
		name    string
		a       []float32
		b       []float32
		want    float64
		wantErr error
	}{
		{
			name: "identical unit vectors",
			a:    []float32{0, 1},
			b:    []float32{0, 1},
			want: 1,
		},
		{
			name: "same direction non-unit vectors",
			a:    []float32{1, 2},
			b:    []float32{2, 4},
			want: 1,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{0, 1},
			b:    []float32{1, 0},
			want: 0,
		},
		{
			name: "opposite vectors",
			a:    []float32{0, 1},
			b:    []float32{0, -1},
			want: -1,
		},
		{
			name: "general vectors",
			a:    []float32{3, 4},
			b:    []float32{5, 12},
			want: 63.0 / 65.0,
		},
		{
			name:    "both vectors empty",
			a:       []float32{},
			b:       []float32{},
			wantErr: ErrEmptyVector,
		},
		{
			name:    "first vector empty",
			a:       []float32{},
			b:       []float32{1},
			wantErr: ErrEmptyVector,
		},
		{
			name:    "second vector empty",
			a:       []float32{1},
			b:       []float32{},
			wantErr: ErrEmptyVector,
		},
		{
			name:    "first vector zero",
			a:       []float32{0, 0},
			b:       []float32{1, 0},
			wantErr: ErrZeroVector,
		},
		{
			name:    "second vector zero",
			a:       []float32{1, 0},
			b:       []float32{0, 0},
			wantErr: ErrZeroVector,
		},
		{
			name:    "dimension mismatch",
			a:       []float32{0, 0, 0},
			b:       []float32{1, 0},
			wantErr: ErrDimensionMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalA := slices.Clone(tt.a)
			originalB := slices.Clone(tt.b)

			got, err := CosineSimilarity(tt.a, tt.b)

			if !slices.Equal(tt.a, originalA) || !slices.Equal(tt.b, originalB) {
				t.Error("CosineSimilarity() modified its input")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CosineSimilarity() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got != 0 {
					t.Errorf("CosineSimilarity() result on error = %v, want zero value", got)
				}
				return
			}
			if difference := math.Abs(float64(got) - tt.want); difference > tolerance {
				t.Errorf("CosineSimilarity() = %v, want %v (difference %v)", got, tt.want, difference)
			}
		})
	}
}
