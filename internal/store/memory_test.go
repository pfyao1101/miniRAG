package store

import (
	"context"
	"errors"
	"maps"
	"math"
	"slices"
	"testing"

	"github.com/pfyao1101/miniRAG/internal/vector"
)

func TestNewMemoryStore(t *testing.T) {
	tests := []struct {
		name      string
		dimension int
		wantErr   error
	}{
		{name: "valid dimension", dimension: 2},
		{name: "zero dimension", dimension: 0, wantErr: ErrInvalidDimension},
		{name: "negative dimension", dimension: -1, wantErr: ErrInvalidDimension},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMemoryStore(tt.dimension)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewMemoryStore() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got != nil {
					t.Errorf("NewMemoryStore() = %#v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("NewMemoryStore() returned a nil store")
			}
			if got.dimension != tt.dimension {
				t.Errorf("store dimension = %d, want %d", got.dimension, tt.dimension)
			}
			if got.records == nil {
				t.Error("store records map is nil")
			}
		})
	}
}

func TestMemoryStoreInsert(t *testing.T) {
	t.Run("insert and copy input", func(t *testing.T) {
		store := newTestMemoryStore(t, 2)
		record := testRecord("record-1")

		if err := store.Insert(context.Background(), record); err != nil {
			t.Fatalf("Insert() unexpected error: %v", err)
		}

		record.Vector[0] = 99
		record.Metadata["source"] = "changed"
		got, err := store.Get(context.Background(), record.ID)
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}
		want := testRecord("record-1")
		if !equalRecord(got, want) {
			t.Errorf("stored record = %#v, want %#v", got, want)
		}
	})

	tests := []struct {
		name    string
		record  Record
		wantErr error
	}{
		{name: "empty ID", record: Record{Vector: []float32{1, 0}}, wantErr: ErrEmptyID},
		{name: "short vector", record: Record{ID: "short", Vector: []float32{1}}, wantErr: ErrVectorDimensionMismatch},
		{name: "long vector", record: Record{ID: "long", Vector: []float32{1, 0, 0}}, wantErr: ErrVectorDimensionMismatch},
		{name: "zero vector", record: Record{ID: "zero", Vector: []float32{0, 0}}, wantErr: vector.ErrZeroVector},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestMemoryStore(t, 2)
			err := store.Insert(context.Background(), tt.record)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Insert() error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("duplicate ID does not overwrite", func(t *testing.T) {
		store := newTestMemoryStore(t, 2)
		original := testRecord("duplicate")
		if err := store.Insert(context.Background(), original); err != nil {
			t.Fatalf("first Insert() unexpected error: %v", err)
		}

		duplicate := Record{ID: original.ID, Vector: []float32{0, 1}, Text: "replacement"}
		err := store.Insert(context.Background(), duplicate)
		if !errors.Is(err, ErrDuplicateID) {
			t.Fatalf("second Insert() error = %v, want %v", err, ErrDuplicateID)
		}

		got, err := store.Get(context.Background(), original.ID)
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}
		if !equalRecord(got, original) {
			t.Errorf("record after duplicate Insert() = %#v, want %#v", got, original)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		store := newTestMemoryStore(t, 2)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := store.Insert(ctx, testRecord("canceled"))
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Insert() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestMemoryStoreGet(t *testing.T) {
	store := newTestMemoryStore(t, 2)
	want := testRecord("record-1")
	if err := store.Insert(context.Background(), want); err != nil {
		t.Fatalf("Insert() unexpected error: %v", err)
	}

	t.Run("get returns a copy", func(t *testing.T) {
		first, err := store.Get(context.Background(), want.ID)
		if err != nil {
			t.Fatalf("first Get() unexpected error: %v", err)
		}
		first.Vector[0] = 99
		first.Metadata["source"] = "changed"

		second, err := store.Get(context.Background(), want.ID)
		if err != nil {
			t.Fatalf("second Get() unexpected error: %v", err)
		}
		if !equalRecord(second, want) {
			t.Errorf("second Get() = %#v, want %#v", second, want)
		}
	})

	t.Run("record not found", func(t *testing.T) {
		got, err := store.Get(context.Background(), "missing")
		if !errors.Is(err, ErrRecordNotFound) {
			t.Errorf("Get() error = %v, want %v", err, ErrRecordNotFound)
		}
		if !equalRecord(got, Record{}) {
			t.Errorf("Get() record on error = %#v, want zero value", got)
		}
	})

	t.Run("empty ID", func(t *testing.T) {
		_, err := store.Get(context.Background(), "")
		if !errors.Is(err, ErrEmptyID) {
			t.Errorf("Get() error = %v, want %v", err, ErrEmptyID)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := store.Get(ctx, want.ID)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Get() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestMemoryStoreDelete(t *testing.T) {
	t.Run("delete existing record", func(t *testing.T) {
		store := newTestMemoryStore(t, 2)
		record := testRecord("record-1")
		if err := store.Insert(context.Background(), record); err != nil {
			t.Fatalf("Insert() unexpected error: %v", err)
		}

		if err := store.Delete(context.Background(), record.ID); err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}
		_, err := store.Get(context.Background(), record.ID)
		if !errors.Is(err, ErrRecordNotFound) {
			t.Errorf("Get() after Delete() error = %v, want %v", err, ErrRecordNotFound)
		}
	})

	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{name: "record not found", id: "missing", wantErr: ErrRecordNotFound},
		{name: "empty ID", id: "", wantErr: ErrEmptyID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestMemoryStore(t, 2)
			err := store.Delete(context.Background(), tt.id)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Delete() error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("canceled context does not delete", func(t *testing.T) {
		store := newTestMemoryStore(t, 2)
		record := testRecord("record-1")
		if err := store.Insert(context.Background(), record); err != nil {
			t.Fatalf("Insert() unexpected error: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := store.Delete(ctx, record.ID)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Delete() error = %v, want %v", err, context.Canceled)
		}
		if _, err := store.Get(context.Background(), record.ID); err != nil {
			t.Errorf("record was deleted after context cancellation: %v", err)
		}
	})
}

func TestMemoryStoreSearch(t *testing.T) {
	t.Run("returns top results", func(t *testing.T) {
		store := newTestMemoryStore(t, 2)
		for _, record := range []Record{
			{ID: "x", Vector: []float32{1, 0}},
			{ID: "diagonal", Vector: []float32{1, 1}},
			{ID: "y", Vector: []float32{0, 1}},
		} {
			if err := store.Insert(context.Background(), record); err != nil {
				t.Fatalf("Insert(%q) unexpected error: %v", record.ID, err)
			}
		}

		got, err := store.Search(context.Background(), []float32{1, 0}, 2)
		if err != nil {
			t.Fatalf("Search() unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("Search() returned %d results, want 2", len(got))
		}
		if got[0].ID != "x" || math.Abs(float64(got[0].Score)-1) > 1e-6 {
			t.Errorf("first Search() result = %#v, want ID x with score 1", got[0])
		}
		if got[1].ID != "diagonal" {
			t.Errorf("second Search() result = %#v, want ID diagonal", got[1])
		}
	})

	tests := []struct {
		name    string
		query   []float32
		k       int
		prepare func(*testing.T, *MemoryStore)
		ctx     func() context.Context
		wantErr error
	}{
		{name: "short query", query: []float32{1}, k: 1, wantErr: ErrVectorDimensionMismatch},
		{name: "long query", query: []float32{1, 0, 0}, k: 1, wantErr: ErrVectorDimensionMismatch},
		{name: "zero k", query: []float32{1, 0}, k: 0, wantErr: vector.ErrInvalidK},
		{name: "negative k", query: []float32{1, 0}, k: -1, wantErr: vector.ErrInvalidK},
		{name: "zero query on empty store", query: []float32{0, 0}, k: 1, wantErr: vector.ErrZeroVector},
		{
			name:  "canceled context",
			query: []float32{1, 0},
			k:     1,
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestMemoryStore(t, 2)
			if tt.prepare != nil {
				tt.prepare(t, store)
			}
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			got, err := store.Search(ctx, tt.query, tt.k)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Search() error = %v, want %v", err, tt.wantErr)
			}
			if got != nil {
				t.Errorf("Search() result on error = %v, want nil", got)
			}
		})
	}
}

func newTestMemoryStore(t *testing.T, dimension int) *MemoryStore {
	t.Helper()

	store, err := NewMemoryStore(dimension)
	if err != nil {
		t.Fatalf("NewMemoryStore(%d) unexpected error: %v", dimension, err)
	}
	return store
}

func testRecord(id string) Record {
	return Record{
		ID:     id,
		Vector: []float32{1, 0},
		Text:   "test record",
		Metadata: map[string]string{
			"source": "test",
		},
	}
}

func equalRecord(a, b Record) bool {
	return a.ID == b.ID &&
		a.Text == b.Text &&
		slices.Equal(a.Vector, b.Vector) &&
		maps.Equal(a.Metadata, b.Metadata)
}
