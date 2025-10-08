// Package main implements the authentication service
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vitalykrupin/auth-service/cmd/auth/config"
	"github.com/vitalykrupin/auth-service/internal/app/auth"
	"github.com/vitalykrupin/auth-service/internal/app/auth/middleware"
	"github.com/vitalykrupin/auth-service/internal/app/authservice"
	"github.com/vitalykrupin/auth-service/internal/app/storage"
	"go.uber.org/zap"
)

const (
	// ShutdownTimeout is the timeout for graceful server shutdown
	ShutdownTimeout = 10 * time.Second

	// ServerTimeout is the timeout for reading and writing HTTP requests
	ServerTimeout = 10 * time.Second
)

// main is the entry point of the authentication service
func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	if err := run(logger.Sugar()); err != nil {
		logger.Fatal("Application failed", zap.Error(err))
	}
}

// run is the main application startup function
// logger is the logger for writing application logs
// Returns an error if the application failed to start
func run(logger *zap.SugaredLogger) error {
	// Create and parse configuration
	conf := config.NewConfig()
	if err := conf.ParseFlags(); err != nil {
		logger.Errorw("Failed to parse flags", "error", err)
		return err
	}

	// Create storage
	store, err := storage.NewStorage(conf)
	if err != nil {
		logger.Errorw("Failed to create storage", "error", err)
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()
		if err := store.CloseStorage(ctx); err != nil {
			logger.Errorw("Failed to close storage", "error", err)
		}
	}()

	// Create auth service
	authSvc := authservice.NewAuthService(store)

	// Create mux router
	mux := http.NewServeMux()

	// Register routes
	mux.Handle("/api/auth/register", auth.NewRegisterHandler(store, authSvc))
	mux.Handle("/api/auth/login", auth.NewLoginHandler(store, authSvc))

	// Add a protected route to demonstrate JWT middleware
	mux.Handle("/api/auth/profile", middleware.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID := r.Context().Value(middleware.UserIDKey)
		if userID == nil {
			http.Error(w, "User not found in context", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("User ID: " + userID.(string)))
	})))

	// Create HTTP server
	srv := &http.Server{
		Addr:         conf.ServerAddress, // Use server address from config
		Handler:      mux,
		ReadTimeout:  ServerTimeout,
		WriteTimeout: ServerTimeout,
	}

	// Channels for handling errors and signals
	errCh := make(chan error, 1)
	go func() {
		logger.Infow("Starting auth service", "address", conf.ServerAddress)
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for completion
	select {
	case sig := <-sigCh:
		logger.Infow("Received signal, shutting down...", "signal", sig)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Errorw("Server error", "error", err)
			return err
		}
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorw("Server shutdown failed", "error", err)
		return err
	}

	logger.Info("Server shutdown completed")
	return nil
}
