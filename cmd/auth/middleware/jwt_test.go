// Package middleware provides HTTP middleware functions for authentication service
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJWTMiddleware(t *testing.T) {
	// Create a test handler that expects a user ID in context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey)
		if userID == nil {
			http.Error(w, "User ID not found in context", http.StatusInternalServerError)
			return
		}
		
		if userID.(string) != "test-user-id" {
			http.Error(w, "Incorrect user ID in context", http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})
	
	// Wrap the test handler with JWT middleware
	protectedHandler := JWTMiddleware(testHandler)
	
	// Generate a token for testing
	token, err := GenerateToken("test-user-id")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	// Create a test request with the token in Authorization header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	// Create a test response recorder
	rr := httptest.NewRecorder()
	
	// Call the protected handler
	protectedHandler.ServeHTTP(rr, req)
	
	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	
	// Check the response body
	expected := "Success"
	if rr.Body.String() != expected {
		t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestGenerateToken(t *testing.T) {
	// Generate a token
	token, err := GenerateToken("test-user-id")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	// Check that token is not empty
	if token == "" {
		t.Error("Generated token is empty")
	}
}

func TestSetUserID(t *testing.T) {
	// Create a test context
	ctx := context.Background()
	
	// Set user ID in context
	ctx = SetUserID(ctx, "test-user-id")
	
	// Check that user ID is correctly set
	userID := ctx.Value(UserIDKey)
	if userID == nil {
		t.Error("User ID not found in context")
		return
	}
	
	if userID.(string) != "test-user-id" {
		t.Errorf("Incorrect user ID in context: got %v want %v", userID.(string), "test-user-id")
	}
}
