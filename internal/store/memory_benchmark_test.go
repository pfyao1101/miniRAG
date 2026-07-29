package store

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkMemoryStoreGet(b *testing.B) {
	store := getMemoryStore(b, 1, 2)

	ctx := context.Background()

	b.ReportAllocs()

	for b.Loop() {
		got, err := store.Get(ctx, "record-0")
		if err != nil {
			b.Fatal(err)
		}
		if got.ID != "record-0" {
			b.Fatalf("Get() ID = %q, want record-0", got.ID)
		}
	}
}

func BenchmarkMemoryStoreSearch(b *testing.B) {
	ctx := context.Background()

	cases := []struct {
		records   int
		dimension int
		k         int
	}{
		{records: 1_000, dimension: 384, k: 10},
		{records: 10_000, dimension: 384, k: 10},
		{records: 10_000, dimension: 768, k: 10},
		{records: 100_000, dimension: 384, k: 10},
		{records: 10_000, dimension: 1536, k: 10},
	}

	for _, tc := range cases {
		store := getMemoryStore(b, tc.records, tc.dimension)
		name := fmt.Sprintf(
			"N=%d/D=%d/K=%d",
			tc.records,
			tc.dimension,
			tc.k,
		)
		query := makeBenchmarkVector(tc.dimension, 1)
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				results, err := store.Search(ctx, query, tc.k)
				if err != nil {
					b.Fatal(err)
				}
				if len(results) != min(tc.k, tc.records) {
					b.Fatalf(
						"got %d results, want %d",
						len(results),
						min(tc.k, tc.records),
					)
				}
			}
		})
	}
}

func BenchmarkMemoryStoreInsertBatch(b *testing.B) {
	dimensions := []int{384, 768, 1536}
	const batchSize = 1_000
	ctx := context.Background()

	for _, d := range dimensions {
		records := make([]Record, batchSize)
		for i := range batchSize {
			records[i] = makeBenchmarkRecord(fmt.Sprintf("record-%d", i), d, i)
		}
		b.Run(fmt.Sprintf("N=%d/D=%d", batchSize, d), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				store, err := NewMemoryStore(d)
				if err != nil {
					b.Fatal(err)
				}

				for _, record := range records {
					err := store.Insert(ctx, record)
					if err != nil {
						b.Fatalf("Insert(%q) unexpected error: %v", record.ID, err)
					}
				}
			}
			b.ReportMetric(
				float64(b.Elapsed().Nanoseconds())/
					float64(b.N*batchSize),
				"ns/record",
			)

			b.ReportMetric(
				float64(b.N*batchSize)/
					b.Elapsed().Seconds(),
				"records/s",
			)
		})
	}
}

func makeBenchmarkVector(dimension, seed int) []float32 {
	values := make([]float32, dimension)
	for i := range values {
		values[i] = float32((i+seed)%17 + 1)
	}
	return values
}

func makeBenchmarkRecord(id string, dimension int, seed int) Record {
	return Record{
		ID:     id,
		Vector: makeBenchmarkVector(dimension, seed),
		Text:   "test record",
		Metadata: map[string]string{
			"source": "test",
		},
	}
}

func getMemoryStore(b *testing.B, n int, d int) *MemoryStore {
	b.Helper()
	store, err := NewMemoryStore(d)
	if err != nil {
		b.Fatalf("NewMemoryStore(%d) unexpected error: %v", d, err)
	}
	for i := range n {
		record := Record{
			ID:       "record-" + strconv.Itoa(i),
			Vector:   makeBenchmarkVector(d, i),
			Text:     "test",
			Metadata: map[string]string{"resource": "test"},
		}

		if err := store.Insert(context.Background(), record); err != nil {
			b.Fatalf("Insert(%q) unexpected error: %v", record.ID, err)
		}
	}
	return store
}
