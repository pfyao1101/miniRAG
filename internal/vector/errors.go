package vector

import "errors"

var (
	ErrEmptyVector       = errors.New("vector must not be empty")
	ErrDimensionMismatch = errors.New("vector dimensions do not match")
	ErrZeroVector        = errors.New("cosine similarity is undefined for zero vector")
)
