// Package config provides functionality for working with application configuration
package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v10"
)

const (
	// defaultServerAddress is the default server address
	defaultServerAddress = "localhost:8080"

	// defaultResponseAddress is the default base URL for responses
	defaultResponseAddress = "http://localhost:8080"

	// defaultDBDSN is the default database connection string
	defaultDBDSN = ""
)

// Config structure for storing application configuration
type Config struct {
	// ServerAddress is the server address
	ServerAddress string `env:"SERVER_ADDRESS"`

	// ResponseAddress is the base URL for responses
	ResponseAddress string `env:"BASE_URL"`

	// FileStorePath is the path to the storage file
	FileStorePath string `env:"FILE_STORAGE_PATH"`

	// DBDSN is the database connection string
	DBDSN string `env:"DATABASE_DSN"`
}

// NewConfig creates a new configuration instance with default values
// Returns a pointer to Config
func NewConfig() *Config {
	return &Config{
		ServerAddress:   defaultServerAddress,
		ResponseAddress: defaultResponseAddress,
		FileStorePath:   filepath.Join(os.TempDir(), "short-url-db.json"),
		DBDSN:           defaultDBDSN,
	}
}

// ParseFlags parses command line flags and environment variables
// Returns an error if parsing failed or configuration is invalid
func (c *Config) ParseFlags() error {
	// Register command line flags
	flag.Func("a", "example: '-a localhost:8080'", func(addr string) error {
		c.ServerAddress = addr
		return nil
	})
	flag.Func("b", "example: '-b http://localhost:8000'", func(addr string) error {
		c.ResponseAddress = addr
		return nil
	})
	flag.Func("f", "example: '-f /tmp/testfile.json'", func(path string) error {
		c.FileStorePath = path
		return nil
	})
	flag.Func("d", "example: '-d postgres://postgres:pwd@localhost:5432/postgres?sslmode=disable'", func(dbAddr string) error {
		c.DBDSN = dbAddr
		return nil
	})
	flag.Parse()

	// Parse environment variables
	err := env.Parse(c)
	if err != nil {
		return fmt.Errorf("failed to parse environment variables: %w", err)
	}

	// Validate configuration
	if err := c.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// Validate checks the correctness of the configuration
// Returns an error if the configuration is invalid
func (c *Config) Validate() error {
	// Check server address
	if c.ServerAddress == "" {
		return fmt.Errorf("server address is required")
	}

	// Check base URL
	if c.ResponseAddress == "" {
		return fmt.Errorf("response address is required")
	}

	// Check URL format
	if _, err := url.ParseRequestURI(c.ResponseAddress); err != nil {
		return fmt.Errorf("invalid response address format: %w", err)
	}

	// Check storage file path (if file storage is used)
	if c.DBDSN == "" && c.FileStorePath == "" {
		return fmt.Errorf("either database DSN or file storage path must be provided")
	}

	return nil
}
