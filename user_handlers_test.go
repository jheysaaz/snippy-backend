package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginWithUsernameOrEmail(t *testing.T) {
	// Setup test database
	testDB := setupTestDB(t)
	defer func() {
		if err := testDB.Close(); err != nil {
			t.Logf("Error closing test database: %v", err)
		}
	}()

	// Initialize schema
	// Set global db variable first
	db = testDB

	if err := initDatabase(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Set global db variable
	db = testDB

	// Clean up any existing test users
	_, _ = testDB.Exec(`DELETE FROM users WHERE username = 'testuser' OR email = 'test@example.com'`)

	// Create a test user with all fields
	hashedPassword, err := HashPassword("testpassword123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	_, err = testDB.Exec(`
		INSERT INTO users (username, email, password_hash, full_name, avatar_url)
		VALUES ('testuser', 'test@example.com', $1, 'Test User', 'https://example.com/avatar.png')
	`, hashedPassword)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	router := gin.New()
	router.POST("/api/v1/auth/login", login)

	tests := []struct {
		name           string
		loginValue     string
		password       string
		expectedStatus int
		expectToken    bool
	}{
		{
			name:           "Login with username",
			loginValue:     "testuser",
			password:       "testpassword123",
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name:           "Login with email",
			loginValue:     "test@example.com",
			password:       "testpassword123",
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name:           "Login with wrong password",
			loginValue:     "testuser",
			password:       "wrongpassword",
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name:           "Login with non-existent user",
			loginValue:     "nonexistent",
			password:       "testpassword123",
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginReq := LoginRequest{
				Login:    tt.loginValue,
				Password: tt.password,
			}

			body, _ := json.Marshal(loginReq)
			req, _ := http.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectToken {
				var response LoginResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if response.AccessToken == "" {
					t.Error("Expected access token, got empty string")
				}

				if response.RefreshToken == "" {
					t.Error("Expected refresh token, got empty string")
				}

				if response.User == nil {
					t.Error("Expected user object, got nil")
				} else {
					if response.User.Username != "testuser" {
						t.Errorf("Expected username 'testuser', got '%s'", response.User.Username)
					}
					if response.User.Email != "test@example.com" {
						t.Errorf("Expected email 'test@example.com', got '%s'", response.User.Email)
					}
				}
			}
		})
	}
}
