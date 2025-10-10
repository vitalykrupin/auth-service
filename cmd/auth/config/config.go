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
	defaultServerAddress = "0.0.0.0:8081"

	// defaultResponseAddress is the default base URL for responses
	defaultResponseAddress = "http://localhost:8081"

	// defaultDBDSN is the default database connection string
	defaultDBDSN = ""

	// defaultMigrationsPath is default path to SQL migrations
	defaultMigrationsPath = "./migrations"
)

// Config structure for storing application configuration
type Config struct {
	// ServerAddress is the server address (alias env: HTTP_ADDR)
	ServerAddress string `env:"HTTP_ADDR"`

	// ResponseAddress is the base URL for responses
	ResponseAddress string `env:"BASE_URL"`

	// FileStorePath is the path to the storage file
	FileStorePath string `env:"FILE_STORAGE_PATH"`

	// DBDSN is the database connection string
	DBDSN string `env:"DB_DSN"`

	// JWTSecret is the secret key for signing JWTs
	JWTSecret string `env:"JWT_SECRET"`

	// RunMigrations toggles running migrations on startup
	RunMigrations bool `env:"RUN_MIGRATIONS"`

	// MigrationsPath is the path to migration files
	MigrationsPath string `env:"MIGRATIONS_PATH"`
}

// NewConfig creates a new configuration instance with default values
// Returns a pointer to Config
func NewConfig() *Config {
	return &Config{
		ServerAddress:   defaultServerAddress,
		ResponseAddress: defaultResponseAddress,
		FileStorePath:   filepath.Join(os.TempDir(), "short-url-db.json"),
		DBDSN:           defaultDBDSN,
		MigrationsPath:  defaultMigrationsPath,
	}
}

// ParseFlags parses command line flags and environment variables
// Returns an error if parsing failed or configuration is invalid
func (c *Config) ParseFlags() error {
	// Register command line flags
	flag.Func("a", "example: '-a 0.0.0.0:8081'", func(addr string) error {
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
	flag.Func("migrations", "example: '-migrations ./migrations'", func(p string) error {
		c.MigrationsPath = p
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
