package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jheysaaz/snippy-backend/app/models"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "mySecurePassword123", false},
		{"empty password", "", true}, // Argon2 requires password
		{"long password", string(make([]byte, 100)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash")
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	password := "correctPassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"correct password", password, hash, true},
		{"wrong password", "wrongPassword", hash, false},
		{"empty password", "", hash, false},
		{"invalid hash", password, "invalid-hash", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPassword(tt.password, tt.hash); got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateAccessToken(t *testing.T) {
	// Set test JWT secret to match the one used in jwtSecret variable
	testSecret := "test-secret-key-for-testing"
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Unsetenv("JWT_SECRET")

	// Reinitialize jwtSecret after env change
	jwtSecret = []byte(getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"))

	testUser := &models.User{
		ID:       "123e4567-e89b-12d3-a456-426614174000",
		Username: "testuser",
		Email:    "test@example.com",
	}

	token, err := GenerateAccessToken(testUser)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}

	// Verify token structure
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(testSecret), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !parsedToken.Valid {
		t.Error("Generated token is invalid")
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to extract claims")
	}

	if claims["user_id"] != testUser.ID {
		t.Errorf("Token user_id = %v, want %v", claims["user_id"], testUser.ID)
	}

	if claims["username"] != testUser.Username {
		t.Errorf("Token username = %v, want %v", claims["username"], testUser.Username)
	}
}

func TestValidateToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key")
	defer os.Unsetenv("JWT_SECRET")

	testUser := &models.User{
		ID:       "123e4567-e89b-12d3-a456-426614174000",
		Username: "testuser",
		Email:    "test@example.com",
	}

	validToken, _ := GenerateAccessToken(testUser)

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
	expiredToken, _ := expiredTokenObj.SignedString([]byte("test-secret-key"))

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"valid token", validToken, false},
		{"expired token", expiredToken, true},
		{"invalid token", "invalid.token.here", true},
		{"empty token", "", true},
		{"malformed token", "not-a-jwt-token", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && claims == nil {
				t.Error("ValidateToken() returned nil claims")
			}
			if !tt.wantErr && claims.UserID != testUser.ID {
				t.Errorf("ValidateToken() user_id = %v, want %v", claims.UserID, testUser.ID)
			}
		})
	}
}

func TestGenerateRandomToken(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"short token", 16, false},
		{"medium token", 32, false},
		{"long token", 64, false},
		{"zero length", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateRandomToken(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRandomToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(token) == 0 && tt.length > 0 {
				t.Error("GenerateRandomToken() returned empty token")
			}
		})
	}
}
