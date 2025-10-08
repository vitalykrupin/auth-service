package authservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vitalykrupin/auth-service/internal/app/storage"
)

// fakeStorage is a minimal implementation of storage.Storage for construction tests
type fakeStorage struct{}

func (f *fakeStorage) GetUserByLogin(ctx context.Context, login string) (*storage.User, error) {
	return nil, errors.New("user not found")
}
func (f *fakeStorage) CreateUser(ctx context.Context, user *storage.User) error { return nil }
func (f *fakeStorage) CloseStorage(ctx context.Context) error                   { return nil }
func (f *fakeStorage) PingStorage(ctx context.Context) error                    { return nil }

// New interface methods for profiles and refresh tokens
func (f *fakeStorage) GetUserProfile(ctx context.Context, userID string) (string, error) {
	return "", errors.New("not implemented")
}
func (f *fakeStorage) SetUserProfile(ctx context.Context, userID, email string) error {
	return nil
}
func (f *fakeStorage) CreateRefreshToken(ctx context.Context, token, userID string, expiresAt time.Time) error {
	return nil
}
func (f *fakeStorage) GetRefreshToken(ctx context.Context, token string) (string, time.Time, bool, error) {
	return "", time.Time{}, false, errors.New("not implemented")
}
func (f *fakeStorage) RevokeRefreshToken(ctx context.Context, token string) error {
	return nil
}
func (f *fakeStorage) DeleteExpiredRefreshTokens(ctx context.Context) error {
	return nil
}

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
