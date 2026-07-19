package vector

import (
	"sort"
)

type Result struct {
	ID    string
	Score float32
}

func TopKBySort(results []Result, k int) ([]Result, error) {
	if k <= 0 {
		return nil, ErrInvalidK
	}

	sorted := make([]Result, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Score == sorted[j].Score {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].Score > sorted[j].Score
	})

	limit := min(k, len(results))
	return sorted[:limit], nil
}
