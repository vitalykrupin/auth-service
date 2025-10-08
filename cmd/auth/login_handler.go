// Package auth provides HTTP request handlers for authentication
package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/vitalykrupin/auth-service/internal/app/auth/middleware"
	"github.com/vitalykrupin/auth-service/internal/app/authservice"
	"github.com/vitalykrupin/auth-service/internal/app/storage"
)

// loginRequest represents the JSON request structure for login
type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// loginResponse represents the JSON response structure for login
type loginResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// LoginHandler handles POST requests for user login
type LoginHandler struct {
	*BaseHandler
	storage     storage.Storage
	authService *authservice.AuthService
}

// NewLoginHandler is the constructor for LoginHandler
func NewLoginHandler(store storage.Storage, authService *authservice.AuthService) *LoginHandler {
	return &LoginHandler{
		BaseHandler: NewBaseHandler(),
		storage:     store,
		authService: authService,
	}
}

// ServeHTTP handles the HTTP request for user login
func (handler *LoginHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	if req.Method != http.MethodPost {
		log.Println("Only POST requests are allowed!")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	loginReq := new(loginRequest)
	if err := json.NewDecoder(req.Body).Decode(loginReq); err != nil {
		log.Println("Can not parse request body", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if login and password are provided
	if loginReq.Login == "" || loginReq.Password == "" {
		log.Println("Login and password are required")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Authenticate user
	userID, err := handler.authService.AuthenticateUser(ctx, loginReq.Login, loginReq.Password)
	if err != nil {
		log.Println("Failed to authenticate user", err)
		w.WriteHeader(http.StatusUnauthorized)
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
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loginResponse{
		UserID: userID,
		Token:  token,
	}); err != nil {
		log.Println("Can not encode response", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
