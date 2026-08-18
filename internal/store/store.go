package store

import (
	"context"

	"github.com/pfyao1101/miniRAG/internal/vector"
)

// MemoryStore 不需要显式声明实现接口，只要方法集合匹配，就自动实现。
type VectorStore interface {
	Insert(ctx context.Context, record Record) error
	Get(ctx context.Context, id string) (Record, error)
	Delete(ctx context.Context, id string) error
	Search(
		ctx context.Context,
		query []float32,
		k int,
	) ([]vector.Result, error)
}
