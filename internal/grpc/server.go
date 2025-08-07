package grpc

import (
	pb "ariand/internal/gen/arian/v1"
	"ariand/internal/service"

	"github.com/charmbracelet/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Server implements all gRPC services in one type
type Server struct {
	pb.UnimplementedAccountServiceServer
	pb.UnimplementedAccountCollaborationServiceServer
	pb.UnimplementedUserServiceServer
	pb.UnimplementedCredentialServiceServer
	pb.UnimplementedTransactionServiceServer
	pb.UnimplementedCategoryServiceServer
	pb.UnimplementedDashboardServiceServer
	pb.UnimplementedReceiptServiceServer

	services     *service.Services
	log          *log.Logger
	healthServer *health.Server
}

func NewServer(services *service.Services, logger *log.Logger) *Server {
	healthServer := health.NewServer()
	return &Server{
		services:     services,
		log:          logger,
		healthServer: healthServer,
	}
}

func (s *Server) RegisterServices(grpcServer *grpc.Server) {
	pb.RegisterAccountServiceServer(grpcServer, s)
	pb.RegisterAccountCollaborationServiceServer(grpcServer, s)
	pb.RegisterUserServiceServer(grpcServer, s)
	pb.RegisterCredentialServiceServer(grpcServer, s)
	pb.RegisterTransactionServiceServer(grpcServer, s)
	pb.RegisterCategoryServiceServer(grpcServer, s)
	pb.RegisterDashboardServiceServer(grpcServer, s)
	pb.RegisterReceiptServiceServer(grpcServer, s)

	grpc_health_v1.RegisterHealthServer(grpcServer, s.healthServer)
}

func (s *Server) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.healthServer.SetServingStatus(service, status)
}
