package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/pfyao1101/miniRAG/internal/vector"
)

type MemoryStore struct {
	dimension int
	records   map[string]Record
	mu        sync.RWMutex
}

func NewMemoryStore(dimension int) (*MemoryStore, error) {
	if dimension <= 0 {
		return nil, ErrInvalidDimension
	}
	return &MemoryStore{
		dimension: dimension,
		records:   make(map[string]Record),
	}, nil
}

func (memoryStore *MemoryStore) Insert(ctx context.Context, record Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if record.ID == "" {
		return ErrEmptyID
	}
	if len(record.Vector) != memoryStore.dimension {
		return ErrVectorDimensionMismatch
	}
	var normSquaredA float64
	for i := range record.Vector {
		av := float64(record.Vector[i])
		normSquaredA += av * av
	}
	if normSquaredA == 0 {
		return vector.ErrZeroVector
	}
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()
	_, ok := memoryStore.records[record.ID]
	if ok {
		return ErrDuplicateID
	}
	memoryStore.records[record.ID] = cloneRecord(record)
	return nil
}

func (memoryStore *MemoryStore) Get(ctx context.Context, id string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if id == "" {
		return Record{}, ErrEmptyID
	}
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()
	record, ok := memoryStore.records[id]
	if !ok {
		return Record{}, ErrRecordNotFound
	}
	return cloneRecord(record), nil
}

func (memoryStore *MemoryStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return ErrEmptyID
	}
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()
	_, ok := memoryStore.records[id]
	if !ok {
		return ErrRecordNotFound
	}
	delete(memoryStore.records, id)
	return nil
}

func (memoryStore *MemoryStore) Search(ctx context.Context, query []float32, k int) ([]vector.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(query) != memoryStore.dimension {
		return nil, ErrVectorDimensionMismatch
	}
	if err := vector.Validate(query); err != nil {
		return nil, err
	}
	if k <= 0 {
		return nil, vector.ErrInvalidK
	}
	var results []vector.Result
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()
	for id, record := range memoryStore.records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		score, err := vector.CosineSimilarity(query, record.Vector)
		if err != nil {
			return nil, fmt.Errorf("score record %q: %w", id, err)
		}
		results = append(results, vector.Result{
			ID:    id,
			Score: score,
		})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return vector.TopKByHeap(results, k)
}
