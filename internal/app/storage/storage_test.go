package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vitalykrupin/auth-service/cmd/auth/config"
)

func TestNewStorage_FileStorage(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "", // Empty DSN means file storage
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if store == nil {
		t.Fatal("Expected storage to be non-nil")
	}

	// Test that it's a FileStorage
	_, ok := store.(*FileStorage)
	if !ok {
		t.Error("Expected FileStorage when DBDSN is empty")
	}

	// Clean up
	ctx := context.Background()
	_ = store.CloseStorage(ctx)
}

func TestNewStorage_DatabaseStorage(t *testing.T) {
	// Test with a database DSN - this will fail to connect but should return DB type
	conf := &config.Config{
		FileStorePath: "",
		DBDSN:         "postgres://invalid:invalid@localhost:5432/invalid",
	}

	store, err := NewStorage(conf)
	// We expect an error because the database connection will fail
	if err == nil {
		t.Error("Expected error for invalid database connection")
	}

	// Store might be returned even with error, but it should be a DB type
	if store != nil {
		_, ok := store.(*DB)
		if !ok {
			t.Error("Expected DB when DBDSN is provided")
		}
	}
}

func TestFileStorage_UserOperations(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ctx := context.Background()
	defer store.CloseStorage(ctx)

	// Test CreateUser
	user := &User{
		ID:       1,
		Login:    "testuser",
		Password: "hashedpassword",
		UserID:   "user123",
	}

	err = store.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Expected no error creating user, got %v", err)
	}

	// Test GetUserByLogin
	retrievedUser, err := store.GetUserByLogin(ctx, "testuser")
	if err != nil {
		t.Fatalf("Expected no error getting user, got %v", err)
	}

	if retrievedUser.Login != "testuser" {
		t.Errorf("Expected login 'testuser', got %s", retrievedUser.Login)
	}

	if retrievedUser.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got %s", retrievedUser.UserID)
	}
}

func TestFileStorage_ProfileOperations(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ctx := context.Background()
	defer store.CloseStorage(ctx)

	// Test SetUserProfile
	err = store.SetUserProfile(ctx, "user123", "test@example.com")
	if err != nil {
		t.Fatalf("Expected no error setting profile, got %v", err)
	}

	// Test GetUserProfile
	email, err := store.GetUserProfile(ctx, "user123")
	if err != nil {
		t.Fatalf("Expected no error getting profile, got %v", err)
	}

	if email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", email)
	}
}

func TestFileStorage_RefreshTokenOperations(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ctx := context.Background()
	defer store.CloseStorage(ctx)

	// Test CreateRefreshToken
	token := "refresh_token_123"
	userID := "user123"
	expiresAt := store.(*FileStorage).refresh["test"].ExpiresAt // This will be zero time

	err = store.CreateRefreshToken(ctx, token, userID, expiresAt)
	if err != nil {
		t.Fatalf("Expected no error creating refresh token, got %v", err)
	}

	// Test GetRefreshToken
	retrievedUserID, _, revoked, err := store.GetRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("Expected no error getting refresh token, got %v", err)
	}

	if retrievedUserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, retrievedUserID)
	}

	if revoked {
		t.Error("Expected token to not be revoked")
	}

	// Test RevokeRefreshToken
	err = store.RevokeRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("Expected no error revoking refresh token, got %v", err)
	}
}
