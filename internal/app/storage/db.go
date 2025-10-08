// Package storage provides PostgreSQL data storage implementation
package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is the PostgreSQL data storage implementation
type DB struct {
	// pool is the database connection pool
	pool *pgxpool.Pool
}

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

	// Create profiles table if it doesn't exist
	_, err = conn.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS profiles (
            user_id VARCHAR(255) PRIMARY KEY,
            email VARCHAR(255) UNIQUE,
            created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
        );`)
	if err != nil {
		log.Println("Can not create profiles table")
		return nil, err
	}

	// Create refresh_tokens table if it doesn't exist
	_, err = conn.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS refresh_tokens (
            token TEXT PRIMARY KEY,
            user_id VARCHAR(255) NOT NULL,
            expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
            revoked BOOLEAN NOT NULL DEFAULT FALSE
        );`)
	if err != nil {
		log.Println("Can not create refresh_tokens table")
		return nil, err
	}

	return &DB{conn}, nil
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

// SetUserProfile upserts user's profile
func (d *DB) SetUserProfile(ctx context.Context, userID, email string) error {
	_, err := d.pool.Exec(ctx, `
        INSERT INTO profiles (user_id, email)
        VALUES ($1, $2)
        ON CONFLICT (user_id) DO UPDATE SET email = EXCLUDED.email;`, userID, email)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// GetUserProfile returns email if present
func (d *DB) GetUserProfile(ctx context.Context, userID string) (string, error) {
	var email string
	err := d.pool.QueryRow(ctx, `SELECT email FROM profiles WHERE user_id = $1;`, userID).Scan(&email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("profile not found for user: %s", userID)
		}
		return "", fmt.Errorf("database error: %w", err)
	}
	return email, nil
}

// CreateRefreshToken stores a refresh token
func (d *DB) CreateRefreshToken(ctx context.Context, token, userID string, expiresAt time.Time) error {
	_, err := d.pool.Exec(ctx, `
        INSERT INTO refresh_tokens (token, user_id, expires_at)
        VALUES ($1, $2, $3);`, token, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// GetRefreshToken fetches refresh token info
func (d *DB) GetRefreshToken(ctx context.Context, token string) (userID string, expiresAt time.Time, revoked bool, err error) {
	err = d.pool.QueryRow(ctx, `
        SELECT user_id, expires_at, revoked FROM refresh_tokens WHERE token = $1;`, token).Scan(&userID, &expiresAt, &revoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", time.Time{}, false, fmt.Errorf("refresh token not found")
		}
		return "", time.Time{}, false, fmt.Errorf("database error: %w", err)
	}
	return
}

// RevokeRefreshToken marks a token as revoked
func (d *DB) RevokeRefreshToken(ctx context.Context, token string) error {
	_, err := d.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked = TRUE WHERE token = $1;`, token)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// DeleteExpiredRefreshTokens removes expired tokens
func (d *DB) DeleteExpiredRefreshTokens(ctx context.Context) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < NOW();`)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}
