package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jheysaaz/snippy-backend/app/auth"
	"github.com/jheysaaz/snippy-backend/app/database"
	"github.com/jheysaaz/snippy-backend/app/models"
)

func TestLoginWithUsernameOrEmail(t *testing.T) {
	// Setup test database
	testDB := setupTestDB(t)
	defer func() {
		if err := testDB.Close(); err != nil {
			t.Logf("Error closing test database: %v", err)
		}
	}()

	// Initialize database
	dbURL := getHandlersTestDBURL()
	os.Setenv("DATABASE_URL", dbURL)
	defer os.Unsetenv("DATABASE_URL")

	if err := database.Init(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.DB.Close()

	// Set database.DB for handlers
	database.DB = testDB

	// Clean up any existing test users
	_, _ = testDB.Exec(`DELETE FROM users WHERE username = 'testuser' OR email = 'test@example.com'`)

	// Create a test user with all fields
	hashedPassword, err := auth.HashPassword("testpassword123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	_, err = testDB.Exec(`
		INSERT INTO users (username, email, password_hash, full_name, avatar_url)
		VALUES ('testuser', 'test@example.com', $1, '', '')
	`, hashedPassword)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	router := gin.New()
	router.POST("/api/v1/auth/login", Login)

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
			loginReq := models.LoginRequest{
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
				var response models.LoginResponse
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

func TestCheckAvailability(t *testing.T) {
	testDB := setupTestDB(t)
	defer func() {
		if err := testDB.Close(); err != nil {
			t.Logf("Error closing test database: %v", err)
		}
	}()

	dbURL := getHandlersTestDBURL()
	os.Setenv("DATABASE_URL", dbURL)
	defer os.Unsetenv("DATABASE_URL")

	if err := database.Init(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.DB.Close()

	database.DB = testDB

	// Create a test user
	hashedPassword, _ := auth.HashPassword("testpassword123")
	_, _ = testDB.Exec(`
		INSERT INTO users (username, email, password_hash, full_name, avatar_url)
		VALUES ('existinguser', 'existing@example.com', $1, '', '')
	`, hashedPassword)

	router := gin.New()
	router.GET("/api/v1/auth/availability", CheckAvailability)

	tests := []struct {
		name               string
		username           string
		email              string
		expectedStatus     int
		usernameAvailable  *bool
		emailAvailable     *bool
	}{
		{
			name:               "both available",
			username:           "newuser",
			email:              "new@example.com",
			expectedStatus:     http.StatusOK,
			usernameAvailable:  boolPtr(true),
			emailAvailable:     boolPtr(true),
		},
		{
			name:               "username taken",
			username:           "existinguser",
			email:              "new@example.com",
			expectedStatus:     http.StatusOK,
			usernameAvailable:  boolPtr(false),
			emailAvailable:     boolPtr(true),
		},
		{
			name:               "email taken",
			username:           "newuser",
			email:              "existing@example.com",
			expectedStatus:     http.StatusOK,
			usernameAvailable:  boolPtr(true),
			emailAvailable:     boolPtr(false),
		},
		{
			name:           "no parameters",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/auth/availability?"
			if tt.username != "" {
				url += "username=" + tt.username + "&"
			}
			if tt.email != "" {
				url += "email=" + tt.email
			}

			req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var response struct {
					UsernameAvailable *bool `json:"usernameAvailable,omitempty"`
					EmailAvailable    *bool `json:"emailAvailable,omitempty"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if tt.usernameAvailable != nil && response.UsernameAvailable != nil {
					if *response.UsernameAvailable != *tt.usernameAvailable {
						t.Errorf("Expected usernameAvailable=%v, got %v", *tt.usernameAvailable, *response.UsernameAvailable)
					}
				}

				if tt.emailAvailable != nil && response.EmailAvailable != nil {
					if *response.EmailAvailable != *tt.emailAvailable {
						t.Errorf("Expected emailAvailable=%v, got %v", *tt.emailAvailable, *response.EmailAvailable)
					}
				}
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
