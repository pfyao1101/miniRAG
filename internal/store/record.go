package store

import (
	"maps"
	"slices"
)

type Record struct {
	ID       string
	Vector   []float32
	Text     string
	Metadata map[string]string
}

func cloneRecord(record Record) Record {
	cloned := record
	cloned.Vector = slices.Clone(record.Vector)
	cloned.Metadata = maps.Clone(record.Metadata)
	return cloned
}
