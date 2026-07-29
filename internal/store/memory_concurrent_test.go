package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestMemoryStoreConcurrentInsertAndGet(t *testing.T) {
	store := newTestMemoryStore(t, 2)

	const workerCount = 100

	t.Run("ConcurrentInsert", func(t *testing.T) {
		var wg sync.WaitGroup
		start := make(chan struct{})
		errCh := make(chan error, workerCount)
		for i := 0; i < workerCount; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				<-start // 等待统一放行

				// 执行并发操作
				err := store.Insert(context.Background(), testRecord(strconv.Itoa(i)))

				errCh <- err
			}(i)
		}

		close(start)
		wg.Wait()
		close(errCh)

		for err := range errCh {
			if err != nil {
				t.Errorf("Insert() unexpected error: %v", err)
			}
		}

		for i := 0; i < workerCount; i++ {
			id := strconv.Itoa(i)
			got, err := store.Get(context.Background(), id)
			if err != nil {
				t.Errorf("Get(%q) unexpected error: %v", id, err)
				continue
			}
			if want := testRecord(id); !equalRecord(got, want) {
				t.Errorf("Get(%q) = %#v, want %#v", id, got, want)
			}
		}
	})

	t.Run("ConcurrentGet", func(t *testing.T) {
		var wg sync.WaitGroup
		start := make(chan struct{})
		errCh := make(chan error, workerCount)
		for i := 0; i < workerCount; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				<-start

				id := strconv.Itoa(i)
				want := testRecord(id)
				for j := 0; j < 100; j++ {
					got, err := store.Get(context.Background(), id)
					if err != nil {
						errCh <- fmt.Errorf("Get(%q): %w", id, err)
						return
					}
					if !equalRecord(got, want) {
						errCh <- fmt.Errorf("Get(%q) = %#v, want %#v", id, got, want)
						return
					}
				}

				errCh <- nil
			}(i)
		}

		close(start)
		wg.Wait()
		close(errCh)

		for err := range errCh {
			if err != nil {
				t.Error(err)
			}
		}
	})
}

func TestMemoryStoreConcurrentDuplicateInsert(t *testing.T) {
	store := newTestMemoryStore(t, 2)

	const workerCount = 100

	var wg sync.WaitGroup
	start := make(chan struct{})
	errCh := make(chan error, workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			<-start // 等待统一放行

			// 执行并发操作
			err := store.Insert(context.Background(), Record{
				ID:     "1",
				Vector: []float32{1, 0},
				Text:   "test record",
				Metadata: map[string]string{
					"source": "test" + strconv.Itoa(i),
				},
			})

			errCh <- err
		}(i)
	}

	close(start)
	wg.Wait()
	close(errCh)

	successCount := 0
	duplicateCount := 0
	for err := range errCh {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, ErrDuplicateID):
			duplicateCount++
		default:
			t.Errorf("Insert() unexpected error: %v", err)
		}
	}

	if successCount != 1 {
		t.Errorf("successful Insert() count = %d, want 1", successCount)
	}
	if duplicateCount != workerCount-1 {
		t.Errorf("duplicate Insert() count = %d, want %d", duplicateCount, workerCount-1)
	}
}

func TestMemoryStoreConcurrentInsertAndSearch(t *testing.T) {
	store := newTestMemoryStore(t, 2)
	if err := store.Insert(context.Background(), testRecord("seed")); err != nil {
		t.Fatalf("Insert(seed) unexpected error: %v", err)
	}

	const (
		writerCount       = 100
		searcherCount     = 20
		searchesPerWorker = 50
		k                 = 10
	)

	var wg sync.WaitGroup
	start := make(chan struct{})
	errCh := make(chan error, writerCount+searcherCount)

	for i := 0; i < writerCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			id := fmt.Sprintf("record-%03d", i)
			errCh <- store.Insert(context.Background(), testRecord(id))
		}(i)
	}

	for i := 0; i < searcherCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < searchesPerWorker; j++ {
				results, err := store.Search(context.Background(), []float32{1, 0}, k)
				if err != nil {
					errCh <- fmt.Errorf("Search(): %w", err)
					return
				}
				if len(results) == 0 || len(results) > k {
					errCh <- fmt.Errorf("Search() returned %d results, want between 1 and %d", len(results), k)
					return
				}
			}

			errCh <- nil
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Error(err)
		}
	}

	for i := 0; i < writerCount; i++ {
		id := fmt.Sprintf("record-%03d", i)
		if _, err := store.Get(context.Background(), id); err != nil {
			t.Errorf("Get(%q) after concurrent insert: %v", id, err)
		}
	}
}

func TestMemoryStoreConcurrentDeleteAndGet(t *testing.T) {
	const recordCount = 100

	store := newTestMemoryStore(t, 2)
	for i := 0; i < recordCount; i++ {
		id := strconv.Itoa(i)
		if err := store.Insert(context.Background(), testRecord(id)); err != nil {
			t.Fatalf("Insert(%q) unexpected error: %v", id, err)
		}
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	errCh := make(chan error, recordCount*2)

	for i := 0; i < recordCount; i++ {
		id := strconv.Itoa(i)

		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			<-start

			got, err := store.Get(context.Background(), id)
			switch {
			case err == nil:
				if want := testRecord(id); !equalRecord(got, want) {
					errCh <- fmt.Errorf("Get(%q) = %#v, want %#v", id, got, want)
					return
				}
				errCh <- nil
			case errors.Is(err, ErrRecordNotFound):
				errCh <- nil
			default:
				errCh <- fmt.Errorf("Get(%q) unexpected error: %w", id, err)
			}
		}(id)

		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			<-start

			errCh <- store.Delete(context.Background(), id)
		}(id)
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Error(err)
		}
	}

	for i := 0; i < recordCount; i++ {
		id := strconv.Itoa(i)
		_, err := store.Get(context.Background(), id)
		if !errors.Is(err, ErrRecordNotFound) {
			t.Errorf("Get(%q) after Delete() error = %v, want %v", id, err, ErrRecordNotFound)
		}
	}
}
