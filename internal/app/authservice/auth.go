// Package authservice provides authentication service functionality
package authservice

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/vitalykrupin/auth-service/internal/app/storage"
	"golang.org/x/crypto/bcrypt"
)

// AuthService provides user authentication functionality
type AuthService struct {
	store storage.Storage
}

// NewAuthService is the constructor for AuthService
func NewAuthService(store storage.Storage) *AuthService {
	return &AuthService{
		store: store,
	}
}

// RegisterUser registers a new user with the given login and password
// Returns the user ID of the newly created user
func (s *AuthService) RegisterUser(ctx context.Context, login, password string) (string, error) {
	// Check if user already exists
	_, err := s.store.GetUserByLogin(ctx, login)
	if err == nil {
		// User already exists
		return "", errors.New("user already exists")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Generate user ID
	userID := uuid.New().String()

	// Create new user
	user := &storage.User{
		Login:    login,
		Password: string(hashedPassword),
		UserID:   userID,
	}

	err = s.store.CreateUser(ctx, user)
	if err != nil {
		return "", err
	}

	return userID, nil
}

// AuthenticateUser authenticates a user with the given login and password
// Returns the user ID if authentication is successful
func (s *AuthService) AuthenticateUser(ctx context.Context, login, password string) (string, error) {
	// Get user by login
	user, err := s.store.GetUserByLogin(ctx, login)
	if err != nil {
		return "", errors.New("invalid login or password")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid login or password")
	}

	return user.UserID, nil
}
