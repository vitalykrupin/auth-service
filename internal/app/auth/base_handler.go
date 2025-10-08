// Package auth provides HTTP request handlers for authentication
package auth

// BaseHandler provides base functionality for auth handlers
type BaseHandler struct {
	// Currently empty, but can be extended with common functionality for auth handlers
}

// NewBaseHandler creates a new BaseHandler instance
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}
