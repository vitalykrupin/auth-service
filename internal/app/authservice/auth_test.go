package authservice

import (
	"context"
	"errors"
	"testing"

	"github.com/vitalykrupin/auth-service/internal/app/storage"
)

// fakeStorage is a minimal implementation of storage.Storage for construction tests
type fakeStorage struct{}

func (f *fakeStorage) Add(ctx context.Context, batch map[storage.Alias]storage.OriginalURL) error {
	return nil
}
func (f *fakeStorage) GetURL(ctx context.Context, alias storage.Alias) (storage.OriginalURL, error) {
	return "", errors.New("not implemented")
}
func (f *fakeStorage) GetAlias(ctx context.Context, url storage.OriginalURL) (storage.Alias, error) {
	return "", errors.New("not implemented")
}
func (f *fakeStorage) GetUserURLs(ctx context.Context, userID string) (storage.AliasKeysMap, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeStorage) DeleteUserURLs(ctx context.Context, userID string, urls []string) error {
	return nil
}
func (f *fakeStorage) GetUserByLogin(ctx context.Context, login string) (*storage.User, error) {
	return nil, errors.New("user not found")
}
func (f *fakeStorage) CreateUser(ctx context.Context, user *storage.User) error { return nil }
func (f *fakeStorage) CloseStorage(ctx context.Context) error                   { return nil }
func (f *fakeStorage) PingStorage(ctx context.Context) error                    { return nil }

// TestNewAuthService_Construct ensures the package compiles and constructs the service
func TestNewAuthService_Construct(t *testing.T) {
	ctx := context.Background()
	svc := NewAuthService(&fakeStorage{})
	if svc == nil {
		t.Fatal("expected non-nil auth service")
	}
	if _, err := svc.AuthenticateUser(ctx, "nouser", "nopass"); err == nil {
		t.Error("expected error for unknown user")
	}
}
