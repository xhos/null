package grpc

import (
	pb "ariand/internal/gen/arian/v1"
	"ariand/internal/service"

	"github.com/charmbracelet/log"
	"google.golang.org/grpc"
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

	services *service.Services
	log      *log.Logger
}

func NewServer(services *service.Services, logger *log.Logger) *Server {
	return &Server{
		services: services,
		log:      logger,
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
}
