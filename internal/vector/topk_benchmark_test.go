package vector

import (
	"fmt"
	"testing"
)

func makeBenchmarkResults(n int) []Result {
	results := make([]Result, n)
	for i := range results {
		results[i] = Result{
			ID:    fmt.Sprintf("id-%06d", i),
			Score: float32((i*37)%1000) / 1000,
		}
	}
	return results
}

func BenchmarkTopK(b *testing.B) {
	cases := []struct {
		n int
		k int
	}{
		{n: 1_000, k: 10},
		{n: 10_000, k: 10},
		{n: 100_000, k: 10},
		{n: 100_000, k: 100},
		{n: 100_000, k: 50_000},
	}
	implementations := []struct {
		name string
		topK func(results []Result, k int) ([]Result, error)
	}{
		{name: "sort", topK: TopKBySort},
		{name: "heap", topK: TopKByHeap},
	}

	for _, tc := range cases {
		results := makeBenchmarkResults(tc.n)
		for _, implementation := range implementations {
			name := fmt.Sprintf(
				"%s/N=%d/K=%d",
				implementation.name,
				tc.n,
				tc.k,
			)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					got, err := implementation.topK(results, tc.k)
					if err != nil {
						b.Fatal(err)
					}
					if len(got) != min(tc.k, tc.n) {
						b.Fatalf(
							"got %d results, want %d",
							len(got),
							min(tc.k, tc.n),
						)
					}
				}
			})
		}
	}

}
