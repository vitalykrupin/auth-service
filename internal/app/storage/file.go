// Package storage provides file-based data storage implementation
package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// JSONUserFS represents the JSON structure for user file storage
type JSONUserFS struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
	UserID   string `json:"user_id"`
}

// FileStorage implements file-based data storage
type FileStorage struct {
	usersFile *os.File
	users     map[string]*User  // login -> user
	profiles  map[string]string // userID -> email
	refresh   map[string]struct {
		UserID    string
		ExpiresAt time.Time
		Revoked   bool
	}
}

// NewFileStorage creates a new file storage instance
// FileStoragePath is the path to the storage file
// Returns a pointer to FileStorage and an error if creation failed
func NewFileStorage(FileStoragePath string) (*FileStorage, error) {
	if FileStoragePath == "" {
		return nil, fmt.Errorf("no FileStoragePath provided")
	}

	// Create users file
	usersFile, err := os.OpenFile(FileStoragePath+".users", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("can not open users file: %w", err)
	}

	fs := FileStorage{
		usersFile: usersFile,
		users:     make(map[string]*User),
		profiles:  make(map[string]string),
		refresh: make(map[string]struct {
			UserID    string
			ExpiresAt time.Time
			Revoked   bool
		}),
	}

	if err := fs.loadUsersFromFile(); err != nil {
		return nil, fmt.Errorf("can not load users from file: %w", err)
	}

	return &fs, nil
}

// loadUsersFromFile loads users from the file system
// Returns an error if loading failed
func (f *FileStorage) loadUsersFromFile() error {
	if _, err := f.usersFile.Seek(0, 0); err != nil {
		return err
	}
	scanner := bufio.NewScanner(f.usersFile)
	for scanner.Scan() {
		var user JSONUserFS
		if err := json.Unmarshal(scanner.Bytes(), &user); err != nil {
			return err
		}
		f.users[user.Login] = &User{
			ID:       user.ID,
			Login:    user.Login,
			Password: user.Password,
			UserID:   user.UserID,
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// CloseStorage closes the file storage
func (f *FileStorage) CloseStorage(ctx context.Context) error {
	if f.usersFile != nil {
		return f.usersFile.Close()
	}
	return nil
}

// PingStorage checks the file storage connection
// ctx is the request context
// Returns an error if connection check failed
func (f *FileStorage) PingStorage(ctx context.Context) error { return nil }

// GetUserByLogin retrieves a user by login from file storage
// ctx is the request context
// login is the user login
// Returns the user and an error if retrieval failed
func (f *FileStorage) GetUserByLogin(ctx context.Context, login string) (user *User, err error) {
	if user, exists := f.users[login]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found for login: %s", login)
}

// CreateUser creates a new user in file storage
// ctx is the request context
// user is the user to create
// Returns an error if creation failed
func (f *FileStorage) CreateUser(ctx context.Context, user *User) error {
	// Check if user already exists
	if _, exists := f.users[user.Login]; exists {
		return fmt.Errorf("user already exists with login: %s", user.Login)
	}

	// Add to memory map
	f.users[user.Login] = user

	// Write to file
	if f.usersFile == nil {
		return errors.New("users file is not opened")
	}

	entry := JSONUserFS{
		ID:       user.ID,
		Login:    user.Login,
		Password: user.Password,
		UserID:   user.UserID,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(f.usersFile)
	if _, err := writer.Write(data); err != nil {
		return err
	}
	if err := writer.WriteByte('\n'); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	return nil
}

// SetUserProfile stores user's email in memory (file-backed persistence not implemented for simplicity)
func (f *FileStorage) SetUserProfile(ctx context.Context, userID, email string) error {
	f.profiles[userID] = email
	return nil
}

// GetUserProfile returns user's email if set
func (f *FileStorage) GetUserProfile(ctx context.Context, userID string) (string, error) {
	if email, ok := f.profiles[userID]; ok {
		return email, nil
	}
	return "", fmt.Errorf("profile not found for user: %s", userID)
}

// CreateRefreshToken stores refresh token in memory
func (f *FileStorage) CreateRefreshToken(ctx context.Context, token, userID string, expiresAt time.Time) error {
	f.refresh[token] = struct {
		UserID    string
		ExpiresAt time.Time
		Revoked   bool
	}{UserID: userID, ExpiresAt: expiresAt, Revoked: false}
	return nil
}

// GetRefreshToken returns token payload
func (f *FileStorage) GetRefreshToken(ctx context.Context, token string) (string, time.Time, bool, error) {
	if r, ok := f.refresh[token]; ok {
		return r.UserID, r.ExpiresAt, r.Revoked, nil
	}
	return "", time.Time{}, false, fmt.Errorf("refresh token not found")
}

// RevokeRefreshToken marks token as revoked
func (f *FileStorage) RevokeRefreshToken(ctx context.Context, token string) error {
	if r, ok := f.refresh[token]; ok {
		r.Revoked = true
		f.refresh[token] = r
		return nil
	}
	return fmt.Errorf("refresh token not found")
}

// DeleteExpiredRefreshTokens cleans memory map
func (f *FileStorage) DeleteExpiredRefreshTokens(ctx context.Context) error {
	now := time.Now()
	for k, v := range f.refresh {
		if now.After(v.ExpiresAt) {
			delete(f.refresh, k)
		}
	}
	return nil
}
