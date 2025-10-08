// Package storage provides in-memory data storage implementation
package storage

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// AliasKeysMap is a map of alias to original URL
type AliasKeysMap map[Alias]OriginalURL

// urlKeysMap is a map of original URL to alias
type urlKeysMap map[OriginalURL]Alias

// MemoryStorage represents in-memory storage
type MemoryStorage struct {
	AliasKeysMap AliasKeysMap
	URLKeysMap   urlKeysMap
	Users        map[string]*User // login -> user
}

// SyncMemoryStorage represents thread-safe in-memory storage
type SyncMemoryStorage struct {
	Mu            sync.Mutex
	MemoryStorage *MemoryStorage
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *SyncMemoryStorage {
	return &SyncMemoryStorage{
		Mu: sync.Mutex{},
		MemoryStorage: &MemoryStorage{
			AliasKeysMap: make(map[Alias]OriginalURL),
			URLKeysMap:   make(map[OriginalURL]Alias),
			Users:        make(map[string]*User),
		},
	}
}

// Add adds new URLs to the in-memory storage
// batch is the map of alias -> OriginalURL to add
// Returns an error if the addition failed
func (s *SyncMemoryStorage) Add(batch map[Alias]OriginalURL) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	for k, v := range batch {
		s.MemoryStorage.AliasKeysMap[k] = v
		s.MemoryStorage.URLKeysMap[v] = k
	}
	return nil
}

// GetURL retrieves the original URL by alias from in-memory storage
// alias is the short URL alias
// Returns the original URL and an error if retrieval failed
func (s *SyncMemoryStorage) GetURL(alias Alias) (url OriginalURL, err error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if url, ok := s.MemoryStorage.AliasKeysMap[alias]; !ok {
		log.Println("URL by alias " + alias + " is not exists")
		return "", fmt.Errorf("url by alias %s is not exists", alias)
	} else {
		return url, nil
	}
}

// GetAlias retrieves the alias for a given URL from in-memory storage
// url is the original URL
// Returns the alias and an error if retrieval failed
func (s *SyncMemoryStorage) GetAlias(url OriginalURL) (alias Alias, err error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if alias, ok := s.MemoryStorage.URLKeysMap[url]; !ok {
		log.Println("Alias by URL " + url + " is not exists")
		return "", fmt.Errorf("alias by URL %s is not exists", url)
	} else {
		return alias, nil
	}
}

// GetUserByLogin retrieves a user by login from in-memory storage
// ctx is the request context
// login is the user login
// Returns the user and an error if retrieval failed
func (s *SyncMemoryStorage) GetUserByLogin(ctx context.Context, login string) (user *User, err error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if user, exists := s.MemoryStorage.Users[login]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found for login: %s", login)
}

// CreateUser creates a new user in in-memory storage
// ctx is the request context
// user is the user to create
// Returns an error if creation failed
func (s *SyncMemoryStorage) CreateUser(ctx context.Context, user *User) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	// Check if user already exists
	if _, exists := s.MemoryStorage.Users[user.Login]; exists {
		return fmt.Errorf("user already exists with login: %s", user.Login)
	}
	// Add to memory map
	s.MemoryStorage.Users[user.Login] = user
	return nil
}
