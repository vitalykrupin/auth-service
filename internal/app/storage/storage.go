// Package storage provides data storage implementation
package storage

import (
	"context"
	"time"

	"github.com/vitalykrupin/auth-service/cmd/auth/config"
)

// User represents a user in the system
type User struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
	UserID   string `json:"user_id"`
}

// Storage interface for authentication data storage operations
type Storage interface {
	// User methods
	// GetUserByLogin retrieves a user by login
	GetUserByLogin(ctx context.Context, login string) (user *User, err error)

	// CreateUser creates a new user
	CreateUser(ctx context.Context, user *User) error

	// Profile methods
	SetUserProfile(ctx context.Context, userID, email string) error
	GetUserProfile(ctx context.Context, userID string) (email string, err error)

	// Refresh tokens
	CreateRefreshToken(ctx context.Context, token, userID string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (userID string, expiresAt time.Time, revoked bool, err error)
	RevokeRefreshToken(ctx context.Context, token string) error
	DeleteExpiredRefreshTokens(ctx context.Context) error

	// CloseStorage closes the storage connection
	CloseStorage(ctx context.Context) error

	// PingStorage checks the storage connection
	PingStorage(ctx context.Context) error
}

// NewStorage creates a new storage instance based on configuration
func NewStorage(conf *config.Config) (Storage, error) {
	if conf.DBDSN != "" {
		return NewDB(conf.DBDSN)
	} else {
		return NewFileStorage(conf.FileStorePath)
	}
}
