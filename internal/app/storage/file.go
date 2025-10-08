// Package storage provides file-based data storage implementation
package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
)

// JSONFS represents the JSON structure for file storage
type JSONFS struct {
	UUID  string      `json:"id"`
	Alias Alias       `json:"alias"`
	URL   OriginalURL `json:"url"`
}

// JSONUserFS represents the JSON structure for user file storage
type JSONUserFS struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
	UserID   string `json:"user_id"`
}

// FileStorage implements file-based data storage
type FileStorage struct {
	SyncMemoryStorage *SyncMemoryStorage
	file              *os.File
	usersFile         *os.File
	users             map[string]*User // login -> user
}

// NewFileStorage creates a new file storage instance
// FileStoragePath is the path to the storage file
// Returns a pointer to FileStorage and an error if creation failed
func NewFileStorage(FileStoragePath string) (*FileStorage, error) {
	if FileStoragePath == "" {
		return nil, fmt.Errorf("no FileStoragePath provided")
	}
	syncMem := NewMemoryStorage()

	file, err := os.OpenFile(FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("can not open file: %w", err)
	}

	// Create users file
	usersFile, err := os.OpenFile(FileStoragePath+".users", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("can not open users file: %w", err)
	}

	fs := FileStorage{
		SyncMemoryStorage: syncMem,
		file:              file,
		usersFile:         usersFile,
		users:             make(map[string]*User),
	}

	if err := fs.LoadJSONfromFS(); err != nil && !errors.Is(err, bufio.ErrTooLong) {
		return nil, fmt.Errorf("can not load JSON from file: %w", err)
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

// LoadJSONfromFS loads JSON data from the file system
// Returns an error if loading failed
func (f *FileStorage) LoadJSONfromFS() error {
	if _, err := f.file.Seek(0, 0); err != nil {
		return err
	}
	scanner := bufio.NewScanner(f.file)
	data := make(map[Alias]OriginalURL)
	for scanner.Scan() {
		var urls JSONFS
		if err := json.Unmarshal(scanner.Bytes(), &urls); err != nil {
			return err
		}
		data[urls.Alias] = urls.URL
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := f.SyncMemoryStorage.Add(data); err != nil {
		return err
	}
	return nil
}

// Add adds new URLs to the file storage
// ctx is the request context
// batch is the map of alias -> OriginalURL to add
// Returns an error if the addition failed
func (f *FileStorage) Add(ctx context.Context, batch map[Alias]OriginalURL) error {
	if err := f.SyncMemoryStorage.Add(batch); err != nil {
		return err
	}
	if f.file == nil {
		return errors.New("file is not opened")
	}

	writter := bufio.NewWriter(f.file)
	for alias, url := range batch {
		entry := JSONFS{
			UUID:  strconv.Itoa(len(f.SyncMemoryStorage.MemoryStorage.AliasKeysMap)),
			Alias: alias,
			URL:   url,
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		if _, err := writter.Write(data); err != nil {
			return err
		}
		if err := writter.WriteByte('\n'); err != nil {
			return err
		}
	}
	if err := writter.Flush(); err != nil {
		return err
	}
	return nil
}

// GetURL retrieves the original URL by alias from file storage
// ctx is the request context
// alias is the short URL alias
// Returns the original URL and an error if retrieval failed
func (f *FileStorage) GetURL(ctx context.Context, alias Alias) (url OriginalURL, err error) {
	return f.SyncMemoryStorage.GetURL(alias)
}

// GetUserURLs retrieves all URLs for a user from file storage
// ctx is the request context
// userID is the user identifier
// Returns a map of alias -> OriginalURL and an error if retrieval failed
func (f *FileStorage) GetUserURLs(ctx context.Context, userID string) (aliasKeysMap AliasKeysMap, err error) {
	return nil, fmt.Errorf("can not get user urls from file storage")
}

// GetAlias retrieves the alias for a given URL from file storage
// ctx is the request context
// url is the original URL
// Returns the alias and an error if retrieval failed
func (f *FileStorage) GetAlias(ctx context.Context, url OriginalURL) (alias Alias, err error) {
	return f.SyncMemoryStorage.GetAlias(url)
}

// DeleteUserURLs deletes user URLs from file storage
// ctx is the request context
// userID is the user identifier
// urls is the list of URLs to delete
// Returns an error if deletion failed
func (f *FileStorage) DeleteUserURLs(ctx context.Context, userID string, urls []string) error {
	return fmt.Errorf("can not delete user urls from file storage")
}

// CloseStorage closes the file storage
// ctx is the request context
// Returns an error if closing failed
func (f *FileStorage) CloseStorage(ctx context.Context) error {
	var firstErr error
	if f.file != nil {
		if err := f.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if f.usersFile != nil {
		if err := f.usersFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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
