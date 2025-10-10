// Package main implements the authentication service
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/vitalykrupin/auth-service/cmd/auth/config"
	"github.com/vitalykrupin/auth-service/internal/app/auth"
	"github.com/vitalykrupin/auth-service/internal/app/auth/middleware"
	"github.com/vitalykrupin/auth-service/internal/app/authservice"
	"github.com/vitalykrupin/auth-service/internal/app/storage"
	"go.uber.org/zap"
)

//go:generate echo "migrate hooks are not generated"

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
	// Optional: run migrations on startup (build-tagged implementation)
	if err := applyMigrations(conf, logger); err != nil {
		logger.Errorw("migrations failed", "error", err)
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

	// Refresh token TTL (hours) from env, default 720h
	refreshTTL := 720 * time.Hour
	if ttlStr := os.Getenv("REFRESH_TOKEN_TTL"); ttlStr != "" {
		if ttlParsed, err := time.ParseDuration(ttlStr + "h"); err == nil {
			refreshTTL = ttlParsed
		}
	}

	// Create mux router
	mux := http.NewServeMux()

	// Healthz endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Register routes
	mux.Handle("/api/auth/register", auth.NewRegisterHandler(store, authSvc))
	mux.Handle("/api/auth/login", auth.NewLoginHandler(store, authSvc))

	// Protected profile endpoint (returns JSON)
	mux.Handle("/api/auth/profile", middleware.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey)
		if userID == nil {
			http.Error(w, "User not found in context", http.StatusInternalServerError)
			return
		}
		email, _ := store.GetUserProfile(r.Context(), userID.(string))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"user_id\":\"" + userID.(string) + "\",\"email\":\"" + email + "\"}"))
	})))

	// Token refresh endpoint
	mux.Handle("/api/auth/token/refresh", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
			return
		}
		type refreshReq struct {
			RefreshToken string `json:"refresh_token"`
		}
		type refreshResp struct {
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}
		var req refreshReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		userID, expiresAt, revoked, err := store.GetRefreshToken(r.Context(), req.RefreshToken)
		if err != nil || revoked || time.Now().After(expiresAt) {
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}
		_ = store.RevokeRefreshToken(r.Context(), req.RefreshToken)
		newRT := uuid.New().String()
		_ = store.CreateRefreshToken(r.Context(), newRT, userID, time.Now().Add(refreshTTL))
		token, err := middleware.GenerateToken(userID)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(refreshResp{Token: token, RefreshToken: newRT})
	}))

	// Logout (revoke refresh token)
	mux.Handle("/api/auth/logout", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		_ = store.RevokeRefreshToken(r.Context(), req.RefreshToken)
		w.WriteHeader(http.StatusNoContent)
	}))

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
