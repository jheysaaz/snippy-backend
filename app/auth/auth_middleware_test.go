package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jheysaaz/snippy-backend/app/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMiddleware(t *testing.T) {
	// Set test JWT secret
	testSecret := "test-middleware-secret"
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Unsetenv("JWT_SECRET")

	// Reinitialize jwtSecret
	jwtSecret = []byte(getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"))

	testUser := &models.User{
		ID:       "123e4567-e89b-12d3-a456-426614174000",
		Username: "testuser",
		Email:    "test@example.com",
	}

	validToken, err := GenerateAccessToken(testUser)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create expired token
	expiredClaims := &Claims{
		UserID:   testUser.ID,
		Username: testUser.Username,
		Email:    testUser.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredToken, _ := expiredTokenObj.SignedString(jwtSecret)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectUserID   bool
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectUserID:   true,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "invalid format - no Bearer",
			authHeader:     validToken,
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "malformed Bearer format",
			authHeader:     "Bearer",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(Middleware())
			router.GET("/protected", func(c *gin.Context) {
				userID, exists := GetUserIDFromContext(c)
				if !exists {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found in context"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"user_id": userID})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectUserID && w.Code == http.StatusOK {
				// Verify user_id is in context
				if !contains(w.Body.String(), testUser.ID) {
					t.Errorf("Expected user_id %s in response, got %s", testUser.ID, w.Body.String())
				}
			}
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		setUserID   bool
		expectFound bool
	}{
		{"user_id exists", "test-user-123", true, true},
		{"user_id missing", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.setUserID {
				c.Set("user_id", tt.userID)
			}

			userID, found := GetUserIDFromContext(c)

			if found != tt.expectFound {
				t.Errorf("Expected found=%v, got %v", tt.expectFound, found)
			}

			if tt.expectFound && userID != tt.userID {
				t.Errorf("Expected user_id=%s, got %s", tt.userID, userID)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
