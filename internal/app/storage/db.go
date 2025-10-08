// Package storage provides PostgreSQL data storage implementation
package storage

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vitalykrupin/auth-service/internal/app/auth/middleware"
)

// DB is the PostgreSQL data storage implementation
type DB struct {
	// pool is the database connection pool
	pool *pgxpool.Pool
}

// ErrDeleted is an error that occurs when trying to get a deleted URL
var ErrDeleted = errors.New(`url deleted`)

// NewDB creates a new connection to the PostgreSQL database
// DBDSN is the database connection string
// Returns a pointer to DB and an error if the connection failed
func NewDB(DBDSN string) (*DB, error) {
	ctx := context.Background()

	// Create connection pool
	conn, err := pgxpool.New(ctx, DBDSN)
	if err != nil {
		log.Println("Can not connect to database")
		return nil, err
	}

	// Create urls table if it doesn't exist
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS urls (
			id serial PRIMARY KEY,
			alias TEXT NOT NULL UNIQUE,
			url TEXT NOT NULL,
			user_id TEXT,
			deleted_flag BOOLEAN NOT NULL DEFAULT FALSE
		);`)
	if err != nil {
		log.Println("Can not create table")
		return nil, err
	}

	// Create users table if it doesn't exist
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			login VARCHAR(255) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL UNIQUE
		);`)
	if err != nil {
		log.Println("Can not create users table")
		return nil, err
	}

	return &DB{conn}, nil
}

// Add adds new URLs to the database
// ctx is the request context
// batch is the map of alias -> OriginalURL to add
// Returns an error if the addition failed
func (d *DB) Add(ctx context.Context, batch map[Alias]OriginalURL) error {
	if len(batch) == 0 {
		return nil
	}

	var query = `INSERT INTO urls (alias, url, user_id) VALUES (@alias, @url, @user_id)`
	b := &pgx.Batch{}
	for alias, url := range batch {
		b.Queue(query, pgx.NamedArgs{
			"alias":   alias,
			"url":     url,
			"user_id": ctx.Value(middleware.UserIDKey),
		})
	}

	results := d.pool.SendBatch(ctx, b)
	defer func() {
		if err := results.Close(); err != nil {
			log.Printf("Failed to close batch results: %v", err)
		}
	}()

	for i := 0; i < len(batch); i++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("failed to execute batch query #%d: %w", i, err)
		}
	}

	return nil
}

// GetAlias gets the alias for a given URL
// ctx is the request context
// url is the original URL
// Returns the alias and an error if retrieval failed
func (d *DB) GetAlias(ctx context.Context, url OriginalURL) (Alias, error) {
	userID := ctx.Value(middleware.UserIDKey)
	if userID == nil || userID == "" {
		return "", fmt.Errorf("user ID not found in context")
	}

	var alias Alias
	err := d.pool.QueryRow(ctx, `SELECT alias FROM urls WHERE url = $1 AND user_id = $2 AND deleted_flag = false;`, url, userID).Scan(&alias)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("alias not found for URL: %s", url)
		}
		log.Printf("Failed to get alias from database: %v", err)
		return "", fmt.Errorf("database error: %w", err)
	}
	return alias, nil
}

// GetURL gets the original URL by alias
// ctx is the request context
// alias is the short URL alias
// Returns the original URL and an error if retrieval failed
func (d *DB) GetURL(ctx context.Context, alias Alias) (OriginalURL, error) {
	var url OriginalURL
	var deletedFlag bool
	row := d.pool.QueryRow(ctx, `SELECT url, deleted_flag FROM urls WHERE alias = $1;`, alias)
	err := row.Scan(&url, &deletedFlag)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("URL not found for alias: %s", alias)
		}
		log.Printf("Failed to get URL from database: %v", err)
		return "", fmt.Errorf("database error: %w", err)
	}
	if deletedFlag {
		return "", ErrDeleted
	}
	return url, nil
}

// GetUserURLs gets all URLs for a user
// ctx is the request context
// userID is the user identifier
// Returns a map of alias -> OriginalURL and an error if retrieval failed
func (d *DB) GetUserURLs(ctx context.Context, userID string) (AliasKeysMap, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	result := make(AliasKeysMap)
	rows, err := d.pool.Query(ctx, `SELECT alias, url FROM urls WHERE user_id = $1;`, userID)
	if err != nil {
		log.Printf("Failed to query user URLs from database: %v", err)
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer func() {
		rows.Close()
	}()

	for rows.Next() {
		var alias, originalURL string
		if err := rows.Scan(&alias, &originalURL); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result[Alias(alias)] = OriginalURL(originalURL)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// DeleteUserURLs marks user URLs as deleted
// ctx is the request context
// userID is the user identifier
// aliases is the list of aliases to delete
// Returns an error if deletion failed
func (d *DB) DeleteUserURLs(ctx context.Context, userID string, aliases []string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	if len(aliases) == 0 {
		return nil
	}

	_, err := d.pool.Exec(ctx, `UPDATE urls SET deleted_flag = TRUE WHERE user_id = $1 AND alias = ANY($2);`, userID, aliases)
	if err != nil {
		log.Printf("Failed to delete URLs from database: %v", err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// CloseStorage closes the database connection
// ctx is the request context
// Returns an error if closing failed
func (d *DB) CloseStorage(ctx context.Context) error {
	if d.pool != nil {
		d.pool.Close()
	}
	return nil
}

// PingStorage checks the database connection
// ctx is the request context
// Returns an error if connection failed
func (d *DB) PingStorage(ctx context.Context) error {
	if d.pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	if err := d.pool.Ping(ctx); err != nil {
		log.Printf("Failed to ping database: %v", err)
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}

// GetUserByLogin retrieves a user by login
// ctx is the request context
// login is the user login
// Returns the user and an error if retrieval failed
func (d *DB) GetUserByLogin(ctx context.Context, login string) (user *User, err error) {
	user = &User{}
	err = d.pool.QueryRow(ctx, `SELECT id, login, password, user_id FROM users WHERE login = $1;`, login).Scan(&user.ID, &user.Login, &user.Password, &user.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found for login: %s", login)
		}
		log.Printf("Failed to get user from database: %v", err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	return user, nil
}

// CreateUser creates a new user
// ctx is the request context
// user is the user to create
// Returns an error if creation failed
func (d *DB) CreateUser(ctx context.Context, user *User) error {
	_, err := d.pool.Exec(ctx, `INSERT INTO users (login, password, user_id) VALUES ($1, $2, $3);`, user.Login, user.Password, user.UserID)
	if err != nil {
		log.Printf("Failed to create user in database: %v", err)
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}
