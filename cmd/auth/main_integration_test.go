package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vitalykrupin/auth-service/cmd/auth/config"
	apiauth "github.com/vitalykrupin/auth-service/internal/app/auth"
	"github.com/vitalykrupin/auth-service/internal/app/auth/middleware"
	"github.com/vitalykrupin/auth-service/internal/app/authservice"
	"github.com/vitalykrupin/auth-service/internal/app/storage"
)

func buildTestMux(t *testing.T) (*http.ServeMux, storage.Storage) {
	t.Helper()
	cfg := config.NewConfig()
	cfg.FileStorePath = filepath.Join(t.TempDir(), "test.json")
	st, err := storage.NewStorage(cfg)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	authSvc := authservice.NewAuthService(st)
	mux := http.NewServeMux()
	mux.Handle("/api/auth/register", apiauth.NewRegisterHandler(st, authSvc))
	mux.Handle("/api/auth/login", apiauth.NewLoginHandler(st, authSvc))
	mux.Handle("/api/auth/profile", middleware.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey)
		if userID == nil {
			http.Error(w, "User not found in context", http.StatusInternalServerError)
			return
		}
		email, _ := st.GetUserProfile(r.Context(), userID.(string))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"user_id\":\"" + userID.(string) + "\",\"email\":\"" + email + "\"}"))
	})))
	// refresh
	refreshTTL := 2 * time.Hour
	mux.Handle("/api/auth/token/refresh", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		var resp struct {
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		userID, expiresAt, revoked, err := st.GetRefreshToken(r.Context(), req.RefreshToken)
		if err != nil || revoked || time.Now().After(expiresAt) {
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}
		_ = st.RevokeRefreshToken(r.Context(), req.RefreshToken)
		newRT := uuid.New().String()
		if err := st.CreateRefreshToken(r.Context(), newRT, userID, time.Now().Add(refreshTTL)); err != nil {
			http.Error(w, "Failed to create refresh token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		token, err := middleware.GenerateToken(userID)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp.Token, resp.RefreshToken = token, newRT
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	// logout
	mux.Handle("/api/auth/logout", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		_ = st.RevokeRefreshToken(r.Context(), req.RefreshToken)
		w.WriteHeader(http.StatusNoContent)
	}))

	return mux, st
}

func TestProfileAndRefreshFlow(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	mux, st := buildTestMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Prepare user + profile + refresh token
	userID := uuid.New().String()
	_ = st.CreateUser(nil, &storage.User{Login: "u", Password: "p", UserID: userID})
	_ = st.SetUserProfile(nil, userID, "u@example.com")
	rt := uuid.New().String()
	_ = st.CreateRefreshToken(nil, rt, userID, time.Now().Add(1*time.Hour))

	// Refresh
	resp, err := http.Post(srv.URL+"/api/auth/token/refresh", "application/json", strings.NewReader(`{"refresh_token":"`+rt+`"}`))
	if err != nil {
		t.Fatalf("refresh post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var rr struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	_ = resp.Body.Close()
	if rr.Token == "" || rr.RefreshToken == "" {
		t.Fatalf("empty tokens returned: token=%q refresh=%q", rr.Token, rr.RefreshToken)
	}

	// Profile
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+rr.Token)
	pr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("profile get: %v", err)
	}
	if pr.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(pr.Body)
		t.Fatalf("expected 200, got %d: %s", pr.StatusCode, string(body))
	}
	_ = pr.Body.Close()
}
