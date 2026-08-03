package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	miniragv1 "github.com/pfyao1101/miniRAG/api/minirag/v1"
	"github.com/pfyao1101/miniRAG/internal/store"
	"github.com/pfyao1101/miniRAG/internal/vector"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServiceInsert(t *testing.T) {
	backend := newTestBackend(t, 2)
	service := NewService(backend)
	request := validInsertRequest("record-1")

	response, err := service.Insert(context.Background(), request)
	if err != nil {
		t.Fatalf("Insert() unexpected error: %v", err)
	}
	if response == nil {
		t.Fatal("Insert() returned a nil response")
	}

	// Mutating the protobuf request after Insert must not mutate stored data.
	request.Record.Vector[0] = 99
	request.Record.Metadata["source"] = "changed"

	got, err := backend.Get(context.Background(), "record-1")
	if err != nil {
		t.Fatalf("backend.Get() unexpected error: %v", err)
	}
	want := store.Record{
		ID:     "record-1",
		Vector: []float32{1, 0},
		Text:   "test record",
		Metadata: map[string]string{
			"source": "test",
		},
	}
	if !equalStoreRecord(got, want) {
		t.Errorf("stored record = %#v, want %#v", got, want)
	}
}

func TestServiceInsertErrors(t *testing.T) {
	tests := []struct {
		name     string
		ctx      func() context.Context
		request  *miniragv1.InsertRequest
		prepare  func(*testing.T, *store.MemoryStore)
		wantCode codes.Code
	}{
		{
			name:     "nil request",
			request:  nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "missing record",
			request:  &miniragv1.InsertRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty ID",
			request: &miniragv1.InsertRequest{Record: &miniragv1.Record{
				Vector: []float32{1, 0},
			}},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "dimension mismatch",
			request: &miniragv1.InsertRequest{Record: &miniragv1.Record{
				Id:     "short",
				Vector: []float32{1},
			}},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "zero vector",
			request: &miniragv1.InsertRequest{Record: &miniragv1.Record{
				Id:     "zero",
				Vector: []float32{0, 0},
			}},
			wantCode: codes.InvalidArgument,
		},
		{
			name:    "duplicate ID",
			request: validInsertRequest("duplicate"),
			prepare: func(t *testing.T, backend *store.MemoryStore) {
				t.Helper()
				if err := backend.Insert(context.Background(), store.Record{
					ID:     "duplicate",
					Vector: []float32{1, 0},
				}); err != nil {
					t.Fatalf("prepare Insert() unexpected error: %v", err)
				}
			},
			wantCode: codes.AlreadyExists,
		},
		{
			name: "canceled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			request:  validInsertRequest("canceled"),
			wantCode: codes.Canceled,
		},
		{
			name: "expired deadline",
			ctx: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			request:  validInsertRequest("expired"),
			wantCode: codes.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := newTestBackend(t, 2)
			if tt.prepare != nil {
				tt.prepare(t, backend)
			}
			service := NewService(backend)
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			response, err := service.Insert(ctx, tt.request)
			if gotCode := status.Code(err); gotCode != tt.wantCode {
				t.Fatalf("Insert() code = %v, want %v; error = %v", gotCode, tt.wantCode, err)
			}
			if response != nil {
				t.Errorf("Insert() response on error = %#v, want nil", response)
			}
		})
	}
}

func TestServiceGet(t *testing.T) {
	backend := newTestBackend(t, 2)
	want := store.Record{
		ID:     "record-1",
		Vector: []float32{1, 0},
		Text:   "test record",
		Metadata: map[string]string{
			"source": "test",
		},
	}
	if err := backend.Insert(context.Background(), want); err != nil {
		t.Fatalf("prepare Insert() unexpected error: %v", err)
	}
	service := NewService(backend)

	response, err := service.Get(context.Background(), &miniragv1.GetRequest{Id: want.ID})
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if response.GetRecord() == nil {
		t.Fatal("Get() returned a nil record")
	}
	got := store.Record{
		ID:       response.GetRecord().GetId(),
		Vector:   response.GetRecord().GetVector(),
		Text:     response.GetRecord().GetText(),
		Metadata: response.GetRecord().GetMetadata(),
	}
	if !equalStoreRecord(got, want) {
		t.Errorf("Get() record = %#v, want %#v", got, want)
	}
}

func TestServiceGetErrors(t *testing.T) {
	tests := []struct {
		name     string
		ctx      func() context.Context
		request  *miniragv1.GetRequest
		wantCode codes.Code
	}{
		{
			name:     "nil request",
			request:  nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "empty ID",
			request:  &miniragv1.GetRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "record not found",
			request:  &miniragv1.GetRequest{Id: "missing"},
			wantCode: codes.NotFound,
		},
		{
			name: "canceled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			request:  &miniragv1.GetRequest{Id: "record-1"},
			wantCode: codes.Canceled,
		},
		{
			name: "expired deadline",
			ctx: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			request:  &miniragv1.GetRequest{Id: "record-1"},
			wantCode: codes.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(newTestBackend(t, 2))
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			response, err := service.Get(ctx, tt.request)
			if gotCode := status.Code(err); gotCode != tt.wantCode {
				t.Fatalf("Get() code = %v, want %v; error = %v", gotCode, tt.wantCode, err)
			}
			if response != nil {
				t.Errorf("Get() response on error = %#v, want nil", response)
			}
		})
	}
}

func TestServiceDelete(t *testing.T) {
	backend := newTestBackend(t, 2)
	request := validInsertRequest("record-1")
	if _, err := NewService(backend).Insert(context.Background(), request); err != nil {
		t.Fatalf("prepare Insert() unexpected error: %v", err)
	}
	service := NewService(backend)

	response, err := service.Delete(context.Background(), &miniragv1.DeleteRequest{Id: "record-1"})
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}
	if response == nil {
		t.Fatal("Delete() returned a nil response")
	}
	if _, err := backend.Get(context.Background(), "record-1"); !errors.Is(err, store.ErrRecordNotFound) {
		t.Errorf("backend.Get() after Delete() error = %v, want %v", err, store.ErrRecordNotFound)
	}
}

func TestServiceDeleteErrors(t *testing.T) {
	tests := []struct {
		name     string
		ctx      func() context.Context
		request  *miniragv1.DeleteRequest
		wantCode codes.Code
	}{
		{
			name:     "nil request",
			request:  nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "empty ID",
			request:  &miniragv1.DeleteRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "record not found",
			request:  &miniragv1.DeleteRequest{Id: "missing"},
			wantCode: codes.NotFound,
		},
		{
			name: "canceled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			request:  &miniragv1.DeleteRequest{Id: "record-1"},
			wantCode: codes.Canceled,
		},
		{
			name: "expired deadline",
			ctx: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			request:  &miniragv1.DeleteRequest{Id: "record-1"},
			wantCode: codes.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(newTestBackend(t, 2))
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			response, err := service.Delete(ctx, tt.request)
			if gotCode := status.Code(err); gotCode != tt.wantCode {
				t.Fatalf("Delete() code = %v, want %v; error = %v", gotCode, tt.wantCode, err)
			}
			if response != nil {
				t.Errorf("Delete() response on error = %#v, want nil", response)
			}
		})
	}
}

func TestServiceSearch(t *testing.T) {
	backend := newTestBackend(t, 2)
	for _, record := range []store.Record{
		{ID: "x", Vector: []float32{1, 0}},
		{ID: "y", Vector: []float32{0, 1}},
	} {
		if err := backend.Insert(context.Background(), record); err != nil {
			t.Fatalf("prepare Insert(%q) unexpected error: %v", record.ID, err)
		}
	}
	service := NewService(backend)

	response, err := service.Search(context.Background(), &miniragv1.SearchRequest{
		Query: []float32{1, 0},
		K:     2,
	})
	if err != nil {
		t.Fatalf("Search() unexpected error: %v", err)
	}
	if len(response.GetResults()) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(response.GetResults()))
	}
	if got := response.GetResults()[0]; got.GetId() != "x" || got.GetScore() != 1 {
		t.Errorf("first Search() result = %#v, want ID x with score 1", got)
	}
	if got := response.GetResults()[1]; got.GetId() != "y" || got.GetScore() != 0 {
		t.Errorf("second Search() result = %#v, want ID y with score 0", got)
	}
}

func TestServiceSearchErrors(t *testing.T) {
	tests := []struct {
		name     string
		ctx      func() context.Context
		request  *miniragv1.SearchRequest
		wantCode codes.Code
	}{
		{
			name:     "nil request",
			request:  nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "missing query",
			request:  &miniragv1.SearchRequest{K: 1},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "dimension mismatch",
			request: &miniragv1.SearchRequest{
				Query: []float32{1},
				K:     1,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "zero vector",
			request: &miniragv1.SearchRequest{
				Query: []float32{0, 0},
				K:     1,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "zero k",
			request: &miniragv1.SearchRequest{
				Query: []float32{1, 0},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "negative k",
			request: &miniragv1.SearchRequest{
				Query: []float32{1, 0},
				K:     -1,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "canceled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			request: &miniragv1.SearchRequest{
				Query: []float32{1, 0},
				K:     1,
			},
			wantCode: codes.Canceled,
		},
		{
			name: "expired deadline",
			ctx: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			request: &miniragv1.SearchRequest{
				Query: []float32{1, 0},
				K:     1,
			},
			wantCode: codes.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(newTestBackend(t, 2))
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			response, err := service.Search(ctx, tt.request)
			if gotCode := status.Code(err); gotCode != tt.wantCode {
				t.Fatalf("Search() code = %v, want %v; error = %v", gotCode, tt.wantCode, err)
			}
			if response != nil {
				t.Errorf("Search() response on error = %#v, want nil", response)
			}
		})
	}
}

func TestToStatusError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{name: "nil", wantCode: codes.OK},
		{name: "canceled", err: context.Canceled, wantCode: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, wantCode: codes.DeadlineExceeded},
		{name: "empty ID", err: store.ErrEmptyID, wantCode: codes.InvalidArgument},
		{name: "dimension mismatch", err: store.ErrVectorDimensionMismatch, wantCode: codes.InvalidArgument},
		{name: "empty vector", err: vector.ErrEmptyVector, wantCode: codes.InvalidArgument},
		{name: "zero vector", err: vector.ErrZeroVector, wantCode: codes.InvalidArgument},
		{name: "invalid k", err: vector.ErrInvalidK, wantCode: codes.InvalidArgument},
		{name: "duplicate", err: store.ErrDuplicateID, wantCode: codes.AlreadyExists},
		{name: "not found", err: store.ErrRecordNotFound, wantCode: codes.NotFound},
		{
			name:     "wrapped known error",
			err:      fmt.Errorf("insert: %w", store.ErrDuplicateID),
			wantCode: codes.AlreadyExists,
		},
		{name: "unknown", err: errors.New("sensitive internal detail"), wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStatusError(tt.err)
			if gotCode := status.Code(got); gotCode != tt.wantCode {
				t.Errorf("toStatusError() code = %v, want %v; error = %v", gotCode, tt.wantCode, got)
			}
			if tt.wantCode == codes.Internal && status.Convert(got).Message() != "internal server error" {
				t.Errorf("internal error message = %q, want %q", status.Convert(got).Message(), "internal server error")
			}
		})
	}
}

func newTestBackend(t *testing.T, dimension int) *store.MemoryStore {
	t.Helper()

	backend, err := store.NewMemoryStore(dimension)
	if err != nil {
		t.Fatalf("NewMemoryStore(%d) unexpected error: %v", dimension, err)
	}
	return backend
}

func validInsertRequest(id string) *miniragv1.InsertRequest {
	return &miniragv1.InsertRequest{Record: &miniragv1.Record{
		Id:     id,
		Vector: []float32{1, 0},
		Text:   "test record",
		Metadata: map[string]string{
			"source": "test",
		},
	}}
}

func equalStoreRecord(a, b store.Record) bool {
	return a.ID == b.ID &&
		a.Text == b.Text &&
		slices.Equal(a.Vector, b.Vector) &&
		maps.Equal(a.Metadata, b.Metadata)
}
