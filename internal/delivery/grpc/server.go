package delivery_grpc

import (
	"fmt"
	"log/slog"
	"net"
	post_grpc "pinstack-post-service/internal/delivery/grpc/post"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/middleware"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/post/v1"
	"google.golang.org/grpc"
)

type Server struct {
	postGRPCService *post_grpc.PostGRPCService
	server          *grpc.Server
	address         string
	port            int
	log             *logger.Logger
}

func NewServer(grpcServer *post_grpc.PostGRPCService, address string, port int, log *logger.Logger) *Server {
	return &Server{
		postGRPCService: grpcServer,
		address:         address,
		port:            port,
		log:             log,
	}
}

func (s *Server) Run() error {
	address := fmt.Sprintf("%s:%d", s.address, s.port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s.server = grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			middleware.UnaryLoggerInterceptor(s.log),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	pb.RegisterPostServiceServer(s.server, s.postGRPCService)

	s.log.Info("Starting gRPC server", slog.Int("port", s.port))
	return s.server.Serve(lis)
}

func (s *Server) Shutdown() error {
	if s.server != nil {
		s.server.GracefulStop()
	}
	return nil
}
