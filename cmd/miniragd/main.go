package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pfyao1101/miniRAG/internal/store"
	"github.com/pfyao1101/miniRAG/internal/transport/grpcserver"
)

type config struct {
	listenAddress   string
	dimension       int
	shutdownTimeout time.Duration
}

var errShutdownTimeout = errors.New("graceful shutdown timed out")

func run(
	ctx context.Context,
	listener net.Listener,
	dimension int,
	shutdownTimeout time.Duration,
) error {
	if shutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be greater than zero")
	}

	backend, err := store.NewMemoryStore(dimension)
	if err != nil {
		return fmt.Errorf("create memory store: %w", err)
	}

	server := grpcserver.NewServer(backend)

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("serve grpc: %w", err)
		}
		return nil

	case <-ctx.Done():
		// 进入关闭流程
		gracefulDone := make(chan struct{})

		go func() {
			server.GracefulStop()
			close(gracefulDone)
		}()

		timer := time.NewTimer(shutdownTimeout)
		defer timer.Stop()

		select {
		// graceful stop 完成
		case <-gracefulDone:
			return nil

		// graceful stop 超时
		case <-timer.C:
			server.Stop()
			<-gracefulDone
			return fmt.Errorf(
				"%w after %s",
				errShutdownTimeout,
				shutdownTimeout,
			)
		}
	}
}

func main() {
	cfg := config{}
	flag.StringVar(
		&cfg.listenAddress,
		"listen",
		"127.0.0.1:50051",
		"gRPC listen address",
	)
	flag.IntVar(&cfg.dimension, "dimension", 2, "vector dimension")

	flag.DurationVar(
		&cfg.shutdownTimeout,
		"shutdown-timeout",
		5*time.Second,
		"maximum time to wait for graceful shutdown",
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	listener, err := net.Listen("tcp", cfg.listenAddress)
	if err != nil {
		log.Fatalf("listen on %q: %v", cfg.listenAddress, err)
	}
	defer listener.Close()

	log.Printf(
		"miniragd listening on %s with dimension %d",
		listener.Addr(),
		cfg.dimension,
	)

	if err := run(ctx, listener, cfg.dimension, cfg.shutdownTimeout); err != nil {
		log.Fatalf("miniragd exited with error: %v", err)
	}

	log.Printf("miniragd stopped")
}
