package auth

import (
	"context"

	internalAuth "github.com/vitalykrupin/auth-service/internal/app/authservice"
	internalStorage "github.com/vitalykrupin/auth-service/internal/app/storage"
)

// Storage is the public alias for the internal storage interface.
type Storage = internalStorage.Storage

// User is the public alias for the internal user type.
type User = internalStorage.User

// AuthService is the public alias for the internal authentication service.
type AuthService = internalAuth.AuthService

// NewAuthService constructs a new authentication service.
func NewAuthService(store Storage) *AuthService { return internalAuth.NewAuthService(store) }

// NewDB creates a new PostgreSQL storage by DSN.
func NewDB(dsn string) (*internalStorage.DB, error) { return internalStorage.NewDB(dsn) }

// NewFileStorage creates a file-based storage by path.
func NewFileStorage(path string) (*internalStorage.FileStorage, error) {
	return internalStorage.NewFileStorage(path)
}

// CloseStorage closes the storage implementation.
func CloseStorage(ctx context.Context, s Storage) error { return s.CloseStorage(ctx) }
