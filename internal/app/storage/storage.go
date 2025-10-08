// Package storage provides data storage implementation
package storage

import (
	"context"

	"github.com/vitalykrupin/auth-service/cmd/auth/config"
)

// Alias represents a short URL alias
type Alias string

// OriginalURL represents an original URL
type OriginalURL string

// User represents a user in the system
type User struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
	UserID   string `json:"user_id"`
}

// Storage interface for data storage operations
type Storage interface {
	// Add adds new URLs to the storage
	Add(ctx context.Context, batch map[Alias]OriginalURL) error

	// GetURL retrieves the original URL by alias
	GetURL(ctx context.Context, alias Alias) (url OriginalURL, err error)

	// GetAlias retrieves the alias for a given URL
	GetAlias(ctx context.Context, url OriginalURL) (alias Alias, err error)

	// GetUserURLs retrieves all URLs for a user
	GetUserURLs(ctx context.Context, userID string) (aliasKeysMap AliasKeysMap, err error)

	// DeleteUserURLs marks user URLs as deleted
	DeleteUserURLs(ctx context.Context, userID string, urls []string) error

	// User methods
	// GetUserByLogin retrieves a user by login
	GetUserByLogin(ctx context.Context, login string) (user *User, err error)

	// CreateUser creates a new user
	CreateUser(ctx context.Context, user *User) error

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
