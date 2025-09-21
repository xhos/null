package api

import (
	"ariand/internal/api/middleware"
	"ariand/internal/gen/arian/v1/arianv1connect"
	"ariand/internal/service"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/charmbracelet/log"
)

type Server struct {
	services    *service.Services
	log         *log.Logger
	healthCheck grpchealth.Checker
}

func NewServer(services *service.Services, logger *log.Logger) *Server {
	healthCheck := grpchealth.NewStaticChecker(
		"arian.v1.UserService",
		"arian.v1.AccountService",
		"arian.v1.TransactionService",
		"arian.v1.CategoryService",
		"arian.v1.RuleService",
		"arian.v1.DashboardService",
		"arian.v1.ReceiptService",
	)

	return &Server{
		services:    services,
		log:         logger,
		healthCheck: healthCheck,
	}
}

func (s *Server) SetServingStatus(service string, healthy bool) {
	if checker, ok := s.healthCheck.(*grpchealth.StaticChecker); ok {
		if healthy {
			checker.SetStatus(service, grpchealth.StatusServing)
		} else {
			checker.SetStatus(service, grpchealth.StatusNotServing)
		}
	}
}

func (s *Server) GetHandler(authConfig *middleware.AuthConfig) http.Handler {
	if authConfig == nil {
		s.log.Fatal("auth configuration is required")
	}

	mux := http.NewServeMux()
	s.registerServices(mux)

	stack := middleware.CreateStack(
		middleware.CORS(),
		middleware.Auth(authConfig, s.log),
		middleware.UserContext(),
	)

	return stack(mux)
}

func (s *Server) registerServices(mux *http.ServeMux) {
	healthPath, healthHandler := grpchealth.NewHandler(s.healthCheck)
	mux.Handle(healthPath, healthHandler)

	reflector := grpcreflect.NewStaticReflector(
		"arian.v1.UserService",
		"arian.v1.AccountService",
		"arian.v1.TransactionService",
		"arian.v1.CategoryService",
		"arian.v1.RuleService",
		"arian.v1.DashboardService",
		"arian.v1.ReceiptService",
	)
	reflectPath, reflectHandler := grpcreflect.NewHandlerV1(reflector)
	mux.Handle(reflectPath, reflectHandler)
	reflectPathAlpha, reflectHandlerAlpha := grpcreflect.NewHandlerV1Alpha(reflector)
	mux.Handle(reflectPathAlpha, reflectHandlerAlpha)

	interceptors := connect.WithInterceptors(
		middleware.ConnectLoggingInterceptor(s.log),
		middleware.UserIDExtractor(),
	)

	path, handler := arianv1connect.NewUserServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	path, handler = arianv1connect.NewAccountServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	path, handler = arianv1connect.NewTransactionServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	path, handler = arianv1connect.NewCategoryServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	path, handler = arianv1connect.NewRuleServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	path, handler = arianv1connect.NewDashboardServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	path, handler = arianv1connect.NewReceiptServiceHandler(s, interceptors)
	mux.Handle(path, handler)

	s.log.Info("all connect-go services registered",
		"health_endpoint", healthPath,
	)
}
