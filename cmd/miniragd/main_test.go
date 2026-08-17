package main

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	miniragv1 "github.com/pfyao1101/miniRAG/api/minirag/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	listenAddress   = "127.0.0.1:0"
	dimension       = 2
	shutdownTimeout = 2 * time.Second
	rpcTimeout      = 5 * time.Second
)

func TestRunServesGRPCOverTCP(t *testing.T) {
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

	listener, err := net.Listen("tcp", listenAddress)

	if err != nil {
		t.Fatalf("listen on %q: %v", listenAddress, err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	serverCtx, cancelServer := context.WithCancel(
		context.Background(),
	)
	t.Cleanup(cancelServer)

	runErr := make(chan error, 1)

	go func() {
		runErr <- run(serverCtx, listener, dimension, shutdownTimeout)
	}()

	rpcCtx, rpcCancel := context.WithTimeout(
		context.Background(),
		rpcTimeout,
	)
	defer rpcCancel()

	conn := newTestGRPCConn(t, listener)

	healthClient := healthpb.NewHealthClient(conn)

	response, err := healthClient.Check(
		rpcCtx,
		&healthpb.HealthCheckRequest{
			Service: miniragv1.VectorStoreService_ServiceDesc.ServiceName,
		},
	)
	if err != nil {
		t.Fatalf("Health.Check(): %v", err)
	}

	if response.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Errorf(
			"Health status = %v, want %v",
			response.GetStatus(),
			healthpb.HealthCheckResponse_SERVING,
		)
	}

	client := miniragv1.NewVectorStoreServiceClient(conn)

	for _, record := range records {
		_, err := client.Insert(rpcCtx, &miniragv1.InsertRequest{
			Record: record,
		})
		if err != nil {
			t.Fatalf("Insert(%q): %v", record.GetId(), err)
		}
	}

	searchResponse, err := client.Search(
		rpcCtx,
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

	cancelServer()

	select {
	case err = <-runErr:
		if err != nil {
			t.Fatalf("server cancel() want nil got %v", err)
		}
	case <-time.After(shutdownTimeout + time.Second):
		t.Fatal("run() did not return after server context cancellation")
	}
}

func TestRunGracefulShutdownPublishesNotServing(t *testing.T) {
	listener, err := net.Listen("tcp", listenAddress)

	if err != nil {
		t.Fatalf("listen on %q: %v", listenAddress, err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	serverCtx, cancelServer := context.WithCancel(
		context.Background(),
	)
	t.Cleanup(cancelServer)

	runErr := make(chan error, 1)

	go func() {
		runErr <- run(serverCtx, listener, dimension, shutdownTimeout)
	}()

	watchCtx, cancelWatch := context.WithTimeout(
		context.Background(),
		rpcTimeout,
	)
	defer cancelWatch()

	conn := newTestGRPCConn(t, listener)

	healthClient := healthpb.NewHealthClient(conn)

	stream, err := healthClient.Watch(
		watchCtx,
		&healthpb.HealthCheckRequest{
			Service: miniragv1.VectorStoreService_ServiceDesc.ServiceName,
		},
	)
	if err != nil {
		t.Fatalf("Health.Watch(): %v", err)
	}

	initial, err := stream.Recv()
	if err != nil {
		t.Fatalf("receive initial health status: %v", err)
	}
	if initial.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Errorf(
			"Health status = %v, want %v",
			initial.GetStatus(),
			healthpb.HealthCheckResponse_SERVING,
		)
	}

	cancelServer()
	// cancelServer() 之后 run 接收到消息进入关闭流程
	// 进行 gracefulStop()
	// 先 shutdown() healthServer
	// 然后停止接受新的连接和 RPC，并等待已经处于 active 状态的 RPC 完成。

	update, err := stream.Recv()
	if err != nil {
		t.Fatalf("receive update health status: %v", err)
	}
	if update.GetStatus() != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Errorf(
			"Health status = %v, want %v",
			update.GetStatus(),
			healthpb.HealthCheckResponse_NOT_SERVING,
		)
	}

	cancelWatch()

	select {
	case err = <-runErr:
		if err != nil {
			t.Fatalf("run(): %v", err)
		}
	case <-time.After(shutdownTimeout + time.Second):
		t.Fatal("run() did not return")
	}
}

func TestRunForcesStopAfterShutdownTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", listenAddress)

	if err != nil {
		t.Fatalf("listen on %q: %v", listenAddress, err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	serverCtx, cancelServer := context.WithCancel(
		context.Background(),
	)
	t.Cleanup(cancelServer)

	runErr := make(chan error, 1)

	go func() {
		runErr <- run(serverCtx, listener, dimension, shutdownTimeout)
	}()

	watchCtx, cancelWatch := context.WithTimeout(
		context.Background(),
		rpcTimeout,
	)
	defer cancelWatch()

	conn := newTestGRPCConn(t, listener)

	healthClient := healthpb.NewHealthClient(conn)

	stream, err := healthClient.Watch(
		watchCtx,
		&healthpb.HealthCheckRequest{
			Service: miniragv1.VectorStoreService_ServiceDesc.ServiceName,
		},
	)
	if err != nil {
		t.Fatalf("Health.Watch(): %v", err)
	}

	initial, err := stream.Recv()
	if err != nil {
		t.Fatalf("receive initial health status: %v", err)
	}
	if initial.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Errorf(
			"Health status = %v, want %v",
			initial.GetStatus(),
			healthpb.HealthCheckResponse_SERVING,
		)
	}

	cancelServer()

	// Keep the Watch RPC active so GracefulStop reaches its timeout.
	// cancelWatch()

	select {
	case err = <-runErr:
		if !errors.Is(err, errShutdownTimeout) {
			t.Fatalf("run(): want %v, got %v", errShutdownTimeout, err)
		}
	case <-time.After(shutdownTimeout + time.Second):
		t.Fatal("run() did not return")
	}
}

func newTestGRPCConn(t *testing.T, listener net.Listener) *grpc.ClientConn {
	t.Helper()

	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient(): %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close gRPC client connection: %v", err)
		}
	})

	return conn
}
