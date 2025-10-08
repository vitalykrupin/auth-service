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
		// Don't call CloseStorage as it might panic with nil pool
	}
}

func TestNewStorage_FileStorageWithEmptyPath(t *testing.T) {
	conf := &config.Config{
		FileStorePath: "",
		DBDSN:         "",
	}

	_, err := NewStorage(conf)
	if err == nil {
		t.Error("Expected error for empty file path")
	}

	// Don't call CloseStorage on failed storage creation
}

func TestNewStorage_FileStorageWithInvalidPath(t *testing.T) {
	// Use a path that doesn't exist and can't be created
	conf := &config.Config{
		FileStorePath: "/invalid/path/that/does/not/exist/test.json",
		DBDSN:         "",
	}

	_, err := NewStorage(conf)
	if err == nil {
		t.Error("Expected error for invalid file path")
	}

	// Don't call CloseStorage on failed storage creation
}

func TestStorage_InterfaceCompliance(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// Close before TempDir cleanup on Windows to release file handles
	defer func() {
		_ = store.CloseStorage(context.Background())
	}()

	// Test that the returned storage implements the Storage interface
	var _ Storage = store
}

func TestStorage_FileStorageOperations(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer func() {
		_ = store.CloseStorage(context.Background())
	}()

	ctx := context.Background()

	// Test Add operation
	batch := map[Alias]OriginalURL{
		"test123": "https://example.com",
		"test456": "https://google.com",
	}

	err = store.Add(ctx, batch)
	if err != nil {
		t.Fatalf("Expected no error on Add, got %v", err)
	}

	// Test GetURL operation
	url, err := store.GetURL(ctx, "test123")
	if err != nil {
		t.Fatalf("Expected no error on GetURL, got %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", url)
	}

	// Test GetAlias operation
	alias, err := store.GetAlias(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Expected no error on GetAlias, got %v", err)
	}
	if alias != "test123" {
		t.Errorf("Expected alias 'test123', got '%s'", alias)
	}

	// Test GetURL for non-existent alias
	_, err = store.GetURL(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent alias")
	}

	// Test GetAlias for non-existent URL
	_, err = store.GetAlias(ctx, "https://nonexistent.com")
	if err == nil {
		t.Error("Expected error for non-existent URL")
	}
}

func TestStorage_FileStorageUserOperations(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer func() {
		_ = store.CloseStorage(context.Background())
	}()

	ctx := context.Background()

	// Test GetUserURLs - should return error for file storage
	_, err = store.GetUserURLs(ctx, "user123")
	if err == nil {
		t.Error("Expected error for GetUserURLs in file storage")
	}

	// Test DeleteUserURLs - should return error for file storage
	err = store.DeleteUserURLs(ctx, "user123", []string{"url1", "url2"})
	if err == nil {
		t.Error("Expected error for DeleteUserURLs in file storage")
	}
}

func TestStorage_FileStoragePing(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer func() {
		_ = store.CloseStorage(context.Background())
	}()

	ctx := context.Background()

	// Test PingStorage - should not return error for file storage
	err = store.PingStorage(ctx)
	if err != nil {
		t.Errorf("Expected no error on PingStorage, got %v", err)
	}
}

func TestStorage_FileStoragePersistence(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.json")

	// First, create storage and add some data
	conf1 := &config.Config{
		FileStorePath: filePath,
		DBDSN:         "",
	}

	store1, err := NewStorage(conf1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ctx := context.Background()
	batch := map[Alias]OriginalURL{
		"persist123": "https://persistent.com",
	}

	err = store1.Add(ctx, batch)
	if err != nil {
		t.Fatalf("Expected no error on Add, got %v", err)
	}

	_ = store1.CloseStorage(ctx)

	// Now create a new storage instance and check if data persists
	conf2 := &config.Config{
		FileStorePath: filePath,
		DBDSN:         "",
	}

	store2, err := NewStorage(conf2)
	if err != nil {
		t.Fatalf("Expected no error on second creation, got %v", err)
	}
	defer func() {
		_ = store2.CloseStorage(ctx)
	}()

	// Check if the data persisted
	url, err := store2.GetURL(ctx, "persist123")
	if err != nil {
		t.Fatalf("Expected no error on GetURL after persistence, got %v", err)
	}
	if url != "https://persistent.com" {
		t.Errorf("Expected URL 'https://persistent.com', got '%s'", url)
	}
}

func TestStorage_FileStorageClose(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ctx := context.Background()

	// Test CloseStorage - should not return error
	err = store.CloseStorage(ctx)
	if err != nil {
		t.Errorf("Expected no error on CloseStorage, got %v", err)
	}

	// Test CloseStorage again - might return error for already closed file
	// This is expected behavior
	_ = store.CloseStorage(ctx)
	// We don't check for specific error as it depends on implementation
}

func TestStorage_FileStorageEmptyBatch(t *testing.T) {
	conf := &config.Config{
		FileStorePath: filepath.Join(t.TempDir(), "test.json"),
		DBDSN:         "",
	}

	store, err := NewStorage(conf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer func() {
		_ = store.CloseStorage(context.Background())
	}()

	ctx := context.Background()

	// Test Add with empty batch
	batch := map[Alias]OriginalURL{}
	err = store.Add(ctx, batch)
	if err != nil {
		t.Errorf("Expected no error on Add with empty batch, got %v", err)
	}
}
