package main

import (
	"ariand/internal/ai"
	_ "ariand/internal/ai/gollm"
	"ariand/internal/config"
	"ariand/internal/db"
	grpcServer "ariand/internal/grpc"
	"ariand/internal/grpc/interceptors"
	"ariand/internal/service"
	"ariand/internal/version"
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// --- configuration ---
	cfg := config.Load()

	// --- logger ---
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = log.InfoLevel
	}
	logger := log.NewWithOptions(os.Stdout, log.Options{
		Prefix: "ariand",
		Level:  level,
	})

	logger.Info("starting ariand", "version", version.FullVersion())

	// --- run database migrations ---
	logger.Info("running database migrations")
	if err := db.RunMigrations(cfg.DatabaseURL, "migrations"); err != nil {
		logger.Fatal("failed to run database migrations", "err", err)
	}
	logger.Info("database migrations completed successfully")

	// --- database ---
	store, err := db.New(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database connection failed", "err", err)
	}
	defer store.Close()
	logger.Info("database connection established")

	// --- AI manager ---
	aiManager := ai.GetManager()

	// --- services ---
	services, err := service.New(store, logger, &cfg, aiManager)
	if err != nil {
		logger.Fatal("Failed to create services", "error", err)
	}
	logger.Info("services initialized")

	// start gRPC server
	serverErrors := make(chan error, 1)

	go func() {
		lis, err := net.Listen("tcp", cfg.Port)
		if err != nil {
			serverErrors <- err
			return
		}

		// setup interceptors
		unaryInterceptor, streamInterceptor := interceptors.SetupInterceptors(interceptors.InterceptorConfig{
			Logger:            logger.WithPrefix("grpc"),
			APIKey:            cfg.APIKey,
			EnableAuth:        true,
			EnableRateLimit:   true,
			RateLimitRPS:      1.0,                                      // 1 request per second for unauthenticated requests
			RateLimitCapacity: 5,                                        // burst capacity
			PublicMethods:     []string{"/grpc.health.v1.Health/Check"}, // health check doesn't need auth
		})

		s := grpc.NewServer(
			grpc.UnaryInterceptor(unaryInterceptor),
			grpc.StreamInterceptor(streamInterceptor),
		)
		grpcSrv := grpcServer.NewServer(services, logger.WithPrefix("grpc"))
		grpcSrv.RegisterServices(s)
		reflection.Register(s)

		grpcSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

		logger.Info("gRPC server is listening", "addr", lis.Addr().String())
		serverErrors <- s.Serve(lis)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Fatal("gRPC server error", "err", err)

	case <-quit:
		logger.Info("shutdown signal received")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			logger.Info("gRPC server stopping...")
			close(done)
		}()

		select {
		case <-done:
			logger.Info("gRPC server stopped gracefully")
		case <-ctx.Done():
			logger.Warn("gRPC server shutdown timed out")
		}
	}

	logger.Info("server shutdown complete")
}
