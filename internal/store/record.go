package store

import "github.com/mohae/deepcopy"

type Record struct {
	ID       string
	Vector   []float32
	Text     string
	Metadata map[string]string
}

func cloneRecord(record Record) Record {
	dst := deepcopy.Copy(record)
	cloned := dst.(Record)
	return cloned
}
