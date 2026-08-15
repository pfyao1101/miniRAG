package grpcserver

import (
	"net"

	miniragv1 "github.com/pfyao1101/miniRAG/api/minirag/v1"
	"github.com/pfyao1101/miniRAG/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer   *grpc.Server
	healthServer *health.Server
}

func NewServer(
	backend store.VectorStore,
	opts ...grpc.ServerOption,
) *Server {
	server := grpc.NewServer(opts...)

	miniragv1.RegisterVectorStoreServiceServer(server, NewService(backend))

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)

	// 让 grpcurl 动态查询有哪些服务、方法和消息类型
	reflection.Register(server)

	healthServer.SetServingStatus(
		miniragv1.VectorStoreService_ServiceDesc.ServiceName,
		healthpb.HealthCheckResponse_SERVING,
	)

	return &Server{
		grpcServer:   server,
		healthServer: healthServer,
	}
}

func (s *Server) Serve(listener net.Listener) error {
	return s.grpcServer.Serve(listener)
}
func (s *Server) GracefulStop() {
	s.healthServer.Shutdown()
	s.grpcServer.GracefulStop()
}

func (s *Server) Stop() {
	s.healthServer.Shutdown()
	s.grpcServer.Stop()
}
