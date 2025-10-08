package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	authpkg "github.com/vitalykrupin/auth-service/internal/app/auth"
	"github.com/vitalykrupin/auth-service/internal/app/auth/middleware"
	"github.com/vitalykrupin/auth-service/internal/app/authservice"
	"github.com/vitalykrupin/auth-service/internal/app/storage"
)

// registerRequest represents the JSON request structure for registration
type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// registerResponse represents the JSON response structure for registration
type registerResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// RegisterHandler handles POST requests for user registration
type RegisterHandler struct {
	*authpkg.BaseHandler
	storage     storage.Storage
	authService *authservice.AuthService
}

// NewRegisterHandler is the constructor for RegisterHandler
func NewRegisterHandler(store storage.Storage, authService *authservice.AuthService) *RegisterHandler {
	return &RegisterHandler{
		BaseHandler: authpkg.NewBaseHandler(),
		storage:     store,
		authService: authService,
	}
}

// ServeHTTP handles the HTTP request for user registration
func (handler *RegisterHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	if req.Method != http.MethodPost {
		log.Println("Only POST requests are allowed!")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	regReq := new(registerRequest)
	if err := json.NewDecoder(req.Body).Decode(regReq); err != nil {
		log.Println("Can not parse request body", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if login and password are provided
	if regReq.Login == "" || regReq.Password == "" {
		log.Println("Login and password are required")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Register user
	userID, err := handler.authService.RegisterUser(ctx, regReq.Login, regReq.Password)
	if err != nil {
		log.Println("Failed to register user", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(userID)
	if err != nil {
		log.Println("Failed to generate token", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set token as cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		Path:     "/",
		MaxAge:   86400, // 24 hours
	})

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(registerResponse{
		UserID: userID,
		Token:  token,
	}); err != nil {
		log.Println("Can not encode response", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
