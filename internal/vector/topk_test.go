package vector

import (
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

func TestTopK(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		k       int
		want    []Result
		wantErr error
	}{
		{
			name: "select top two from unsorted results",
			results: []Result{
				{ID: "c", Score: 0.7},
				{ID: "a", Score: 0.9},
				{ID: "b", Score: 0.8},
			},
			k: 2,
			want: []Result{
				{ID: "a", Score: 0.9},
				{ID: "b", Score: 0.8},
			},
		},
		{
			name: "select one result",
			results: []Result{
				{ID: "a", Score: 0.2},
				{ID: "b", Score: 0.8},
			},
			k: 1,
			want: []Result{
				{ID: "b", Score: 0.8},
			},
		},
		{
			name: "k equals result count",
			results: []Result{
				{ID: "b", Score: 0.4},
				{ID: "a", Score: 0.6},
			},
			k: 2,
			want: []Result{
				{ID: "a", Score: 0.6},
				{ID: "b", Score: 0.4},
			},
		},
		{
			name: "k greater than result count",
			results: []Result{
				{ID: "b", Score: 0.4},
				{ID: "a", Score: 0.6},
			},
			k: 5,
			want: []Result{
				{ID: "a", Score: 0.6},
				{ID: "b", Score: 0.4},
			},
		},
		{
			name: "equal scores use ascending IDs",
			results: []Result{
				{ID: "c", Score: 0.9},
				{ID: "a", Score: 0.9},
				{ID: "b", Score: 0.9},
			},
			k: 3,
			want: []Result{
				{ID: "a", Score: 0.9},
				{ID: "b", Score: 0.9},
				{ID: "c", Score: 0.9},
			},
		},
		{
			name: "top k cuts through equal scores",
			results: []Result{
				{ID: "d", Score: 0.8},
				{ID: "c", Score: 0.9},
				{ID: "a", Score: 0.9},
				{ID: "b", Score: 0.9},
			},
			k: 2,
			want: []Result{
				{ID: "a", Score: 0.9},
				{ID: "b", Score: 0.9},
			},
		},
		{
			name:    "empty results",
			results: []Result{},
			k:       1,
			want:    []Result{},
		},
		{
			name: "zero k",
			results: []Result{
				{ID: "a", Score: 0.9},
			},
			k:       0,
			wantErr: ErrInvalidK,
		},
		{
			name: "negative k",
			results: []Result{
				{ID: "a", Score: 0.9},
			},
			k:       -1,
			wantErr: ErrInvalidK,
		},
	}

	implementations := []struct {
		name string
		topK func([]Result, int) ([]Result, error)
	}{
		{name: "sort", topK: TopKBySort},
		{name: "heap", topK: TopKByHeap},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, implementation := range implementations {
				t.Run(implementation.name, func(t *testing.T) {
					input := slices.Clone(tt.results)

					got, err := implementation.topK(input, tt.k)

					if !slices.Equal(input, tt.results) {
						t.Errorf("input was modified: got %v, want %v", input, tt.results)
					}
					if !errors.Is(err, tt.wantErr) {
						t.Fatalf("error = %v, want %v", err, tt.wantErr)
					}
					if tt.wantErr != nil {
						if got != nil {
							t.Errorf("result on error = %v, want nil", got)
						}
						return
					}
					if !slices.Equal(got, tt.want) {
						t.Errorf("result = %v, want %v", got, tt.want)
					}
				})
			}
		})
	}
}

func TestTopKByHeapMatchesSort(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for _, size := range []int{0, 1, 2, 10, 101, 1000} {
		results := make([]Result, size)
		for i := range results {
			results[i] = Result{
				ID:    fmt.Sprintf("id-%04d", i),
				Score: float32(rng.Intn(21)-10) / 10,
			}
		}

		ks := []int{1, size/2 + 1, size, size + 3}
		seenK := make(map[int]struct{}, len(ks))
		for _, k := range ks {
			if k <= 0 {
				continue
			}
			if _, exists := seenK[k]; exists {
				continue
			}
			seenK[k] = struct{}{}

			t.Run(fmt.Sprintf("size=%d/k=%d", size, k), func(t *testing.T) {
				want, err := TopKBySort(results, k)
				if err != nil {
					t.Fatalf("TopKBySort() unexpected error: %v", err)
				}

				got, err := TopKByHeap(results, k)
				if err != nil {
					t.Fatalf("TopKByHeap() unexpected error: %v", err)
				}
				if !slices.Equal(got, want) {
					t.Errorf("TopKByHeap() = %v, want %v", got, want)
				}
			})
		}
	}
}
