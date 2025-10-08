package auth

import (
	"net/http"

	internalJWT "github.com/vitalykrupin/auth-service/internal/app/auth/middleware"
)

// Claims is the public alias for JWT claims.
type Claims = internalJWT.Claims

// ContextKey is the alias for JWT context key type.
type ContextKey = internalJWT.ContextKey

// UserIDKey is the exported context key for user id.
const UserIDKey = internalJWT.UserIDKey

// GenerateToken re-exports JWT token generator.
func GenerateToken(userID string) (string, error) { return internalJWT.GenerateToken(userID) }

// JWTMiddleware re-exports the HTTP middleware.
func JWTMiddleware(next http.Handler) http.Handler { return internalJWT.JWTMiddleware(next) }
