package main

import (
	"ariand/internal/ai"
	_ "ariand/internal/ai/gollm"
	api "ariand/internal/api"
	"ariand/internal/api/middleware"
	"ariand/internal/config"
	"ariand/internal/db"
	"ariand/internal/service"
	"ariand/internal/version"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	cfg := config.Load()

	// ----- logger -----------------
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("failed to create log file", "err", err)
	}
	defer logFile.Close()

	logger := log.NewWithOptions(
		io.MultiWriter(os.Stdout, logFile),
		log.Options{
			ReportTimestamp: true,
			Level:           cfg.LogLevel,
			Formatter:       log.JSONFormatter,
		})

	logger.Info("starting ariand", "version", version.Full())

	// ----- migrations -------------
	logger.Info("running database migrations")
	if err := db.RunMigrations(cfg.DatabaseURL, "internal/db/migrations"); err != nil {
		logger.Fatal("failed to run database migrations", "err", err)
	}
	logger.Info("database migrations completed successfully")

	// ----- database ---------------
	store, err := db.New(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database connection failed", "err", err)
	}
	defer store.Close()
	logger.Info("database connection established")

	// ----- ai manager -------------
	aiManager := ai.GetManager()

	// ----- services ---------------
	services, err := service.New(store, logger, &cfg, aiManager)
	if err != nil {
		logger.Fatal("failed to create services", "error", err)
	}
	logger.Info("services initialized")

	// ----- api layer --------
	srv := api.NewServer(services, logger.WithPrefix("api"))
	authConfig := &middleware.AuthConfig{
		InternalAPIKey: cfg.APIKey,
		BetterAuthURL:  cfg.BetterAuthURL,
	}

	handler := srv.GetHandler(authConfig)

	serverErrors := make(chan error, 1)

	go func() {
		server := &http.Server{
			Addr:    cfg.Address,
			Handler: h2c.NewHandler(handler, &http2.Server{}),
		}

		logger.Info("server is listening", "addr", cfg.Address)
		serverErrors <- server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Fatal("server error", "err", err)

	case <-quit:
		logger.Info("shutdown signal received")
		logger.Info("server stopping...")
	}

	logger.Info("server shutdown complete")
}
