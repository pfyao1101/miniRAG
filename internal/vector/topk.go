package vector

import (
	"container/heap"
	"sort"
)

type Result struct {
	ID    string
	Score float32
}

type resultHeap []Result

func (h resultHeap) Len() int { return len(h) }
func (h resultHeap) Less(i, j int) bool {
	if h[i].Score == h[j].Score {
		return h[i].ID > h[j].ID
	}
	return h[i].Score < h[j].Score
}
func (h resultHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}
func (h *resultHeap) Push(r any) {
	*h = append(*h, r.(Result))
}
func (h *resultHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func resultBetter(a, b Result) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return a.ID < b.ID
}

func TopKBySort(results []Result, k int) ([]Result, error) {
	if k <= 0 {
		return nil, ErrInvalidK
	}

	sorted := make([]Result, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return resultBetter(sorted[i], sorted[j])
	})

	limit := min(k, len(results))
	return sorted[:limit], nil
}

func TopKByHeap(results []Result, k int) ([]Result, error) {
	if k <= 0 {
		return nil, ErrInvalidK
	}

	limit := min(len(results), k)
	outputs := make([]Result, limit)
	index := limit - 1
	h := make(resultHeap, 0, limit)

	for _, candidate := range results {
		if h.Len() < limit {
			heap.Push(&h, candidate)
			continue
		}

		if resultBetter(candidate, h[0]) {
			h[0] = candidate
			heap.Fix(&h, 0)
		}
	}

	for h.Len() > 0 {
		outputs[index] = heap.Pop(&h).(Result)
		index--
	}

	return outputs, nil
}
