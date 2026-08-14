package grpcserver

import (
	"context"
	"math"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	miniragv1 "github.com/pfyao1101/miniRAG/api/minirag/v1"
	"github.com/pfyao1101/miniRAG/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

func TestGRPCWorkflow(t *testing.T) {
	client := newTestGRPCClient(t, 2)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	records := []*miniragv1.Record{
		{
			Id:     "b",
			Vector: []float32{1, 0},
			Text:   "record b",
			Metadata: map[string]string{
				"source": "test",
			},
		},
		{
			Id:     "a",
			Vector: []float32{1, 0},
			Text:   "record a",
			Metadata: map[string]string{
				"source": "test",
			},
		},
		{
			Id:     "c",
			Vector: []float32{0, 1},
			Text:   "record c",
			Metadata: map[string]string{
				"source": "test",
			},
		},
	}

	// Insert
	for _, record := range records {
		_, err := client.Insert(
			ctx,
			&miniragv1.InsertRequest{Record: record},
		)
		if err != nil {
			t.Fatalf("Insert(%q): %v", record.GetId(), err)
		}
	}
	// Get
	for _, record := range records {
		getResponse, err := client.Get(
			ctx,
			&miniragv1.GetRequest{Id: record.Id},
		)
		if err != nil {
			t.Fatalf("Get(%v): %v", record.Id, err)
		}

		if !proto.Equal(getResponse.GetRecord(), record) {
			t.Errorf(
				"Get(%v) record = %v, want %v",
				record.Id,
				getResponse.GetRecord(),
				record,
			)
		}
	}
	// Search
	searchResponse, err := client.Search(
		ctx,
		&miniragv1.SearchRequest{
			Query: []float32{1, 0},
			K:     2,
		},
	)
	if err != nil {
		t.Fatalf("Search(): %v", err)
	}
	results := searchResponse.GetResults()

	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}

	if results[0].GetId() != "a" || results[1].GetId() != "b" {
		t.Errorf(
			"Search() IDs = [%q, %q], want [a, b]",
			results[0].GetId(),
			results[1].GetId(),
		)
	}

	if math.Abs(float64(results[0].GetScore()-1)) > 1e-6 {
		t.Errorf(
			"Search() first score = %v, want 1",
			results[0].GetScore(),
		)
	}

	if math.Abs(float64(results[1].GetScore()-1)) > 1e-6 {
		t.Errorf(
			"Search() second score = %v, want 1",
			results[1].GetScore(),
		)
	}
	// Delete
	_, err = client.Delete(
		ctx,
		&miniragv1.DeleteRequest{Id: "a"},
	)
	if err != nil {
		t.Fatalf("Delete(a): %v", err)
	}

	_, err = client.Get(
		ctx,
		&miniragv1.GetRequest{Id: "a"},
	)
	if status.Code(err) != codes.NotFound {
		t.Errorf(
			"Get(a) after Delete code = %v, want %v; error = %v",
			status.Code(err),
			codes.NotFound,
			err,
		)
	}
}

func TestGRPCConcurrentRPCs(t *testing.T) {
	const workerCount = 16
	client := newTestGRPCClient(t, 2)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	t.Run("ConcurrentInsert", func(t *testing.T) {
		var wg sync.WaitGroup
		start := make(chan struct{})
		errCh := make(chan error, workerCount)

		for i := range workerCount {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start

				_, err := client.Insert(ctx, &miniragv1.InsertRequest{Record: testRecord(strconv.Itoa(i))})
				errCh <- err
			}()
		}

		close(start)
		wg.Wait()
		close(errCh)

		for err := range errCh {
			if err != nil {
				t.Errorf("Insert() unexpected error: %v", err)
			}
		}
	})

	t.Run("records check", func(t *testing.T) {
		response, err := client.Search(
			ctx,
			&miniragv1.SearchRequest{
				Query: []float32{1, 0},
				K:     workerCount,
			},
		)

		if err != nil {
			t.Fatalf("Search() unexpected error: %v", err)
		}

		if len(response.GetResults()) != workerCount {
			t.Fatalf("records num get %v want %v", len(response.GetResults()), workerCount)
		}
	})

}

// 辅助函数
// 创建 cient
func newTestGRPCClient(
	t *testing.T,
	dimension int,
) miniragv1.VectorStoreServiceClient {
	t.Helper()

	// 1. 创建 MemoryStore
	backend, err := store.NewMemoryStore(dimension)
	if err != nil {
		t.Fatalf("NewMemoryStore(%d): %v", dimension, err)
	}

	// 2. 创建 bufconn.Listener
	listener := bufconn.Listen(1024 * 1024)

	// 3. 创建 grpc.Server
	server := grpc.NewServer()
	// 4. 注册 VectorStoreService
	// 把生成的 ServiceDesc 和 实现的 Service 注册到 server
	miniragv1.RegisterVectorStoreServiceServer(
		server,
		NewService(backend),
	)

	// 5. 在 goroutine 中启动 Serve
	serveErr := make(chan error, 1) // 有缓冲允许 goroutine 自行退出

	go func() {
		serveErr <- server.Serve(listener)
	}()

	// 6. 通过自定义 dialer 创建 ClientConn
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer( // tcp dialer 换成了 内存 dialer
			func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			},
		),
		// 不配置 TLS
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		server.Stop()
		t.Fatalf("grpc.NewClient(): %v", err)
	}

	// 7. 使用 t.Cleanup 回收资源
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close client connection: %v", err)
		}

		server.Stop()

		select {
		case err := <-serveErr:
			if err != nil {
				t.Errorf("grpc.Server.Serve(): %v", err)
			}
		case <-time.After(time.Second):
			t.Error("grpc.Server.Serve() did not return")
		}
	})

	// 8. 返回生成的 VectorStoreServiceClient
	return miniragv1.NewVectorStoreServiceClient(conn)
}

func testRecord(id string) *miniragv1.Record {
	return &miniragv1.Record{
		Id:     id,
		Vector: []float32{1, 0},
		Text:   "record b",
		Metadata: map[string]string{
			"source": "test",
		},
	}
}
