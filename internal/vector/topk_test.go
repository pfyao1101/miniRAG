package vector

import (
	"errors"
	"slices"
	"testing"
)

func TestTopKBySort(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := slices.Clone(tt.results)

			got, err := TopKBySort(tt.results, tt.k)

			if !slices.Equal(tt.results, original) {
				t.Error("TopKBySort() modified its input")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("TopKBySort() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got != nil {
					t.Errorf("TopKBySort() result on error = %v, want nil", got)
				}
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("TopKBySort() = %v, want %v", got, tt.want)
			}
		})
	}
}
