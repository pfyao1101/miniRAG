package store

import "errors"

var (
	ErrInvalidDimension        = errors.New("dimension must be greater than zero")
	ErrEmptyID                 = errors.New("record ID must not be empty")
	ErrDuplicateID             = errors.New("record ID already exists")
	ErrRecordNotFound          = errors.New("record not found")
	ErrVectorDimensionMismatch = errors.New("vector dimension does not match store dimension")
)
