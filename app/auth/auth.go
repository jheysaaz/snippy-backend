// Package auth provides authentication, JWT handling, and password hashing.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jheysaaz/snippy-backend/app/models"
	"golang.org/x/crypto/argon2"
)

// Argon2 parameters
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// HashPassword hashes a password using Argon2id
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// Generate a random salt
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encode to base64 for storage: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return encoded hash with parameters
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads, b64Salt, b64Hash)

	return encodedHash, nil
}

// CheckPassword verifies a password against its Argon2id hash
func CheckPassword(password, encodedHash string) bool {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	// Verify it's argon2id
	if parts[1] != "argon2id" {
		return false
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false
	}

	if version != argon2.Version {
		return false
	}

	var memory, time uint32
	var threads uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Hash the input password with the same parameters
	inputHash := argon2.IDKey([]byte(password), salt, time, memory, threads, argon2KeyLen)

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(inputHash, decodedHash) == 1
}

// GenerateRandomToken generates a random token for sessions/auth
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateOAuthToken validates OAuth tokens from Google or Apple
// Note: In production, you should verify the token with the OAuth provider's API
// TODO: Implement actual OAuth token validation
func ValidateOAuthToken(provider, _ string) error {
	// This is a placeholder. In production, you need to:
	// 1. For Google: Verify the token using Google's tokeninfo endpoint
	// 2. For Apple: Verify the JWT token using Apple's public keys

	// Example structure for production implementation:
	switch provider {
	case "google":
		// Verify with Google's API
		// https://oauth2.googleapis.com/tokeninfo?id_token=TOKEN
		return errors.New("google OAuth validation not implemented - requires API key")
	case "apple":
		// Verify Apple's JWT token
		// Use Apple's public keys to verify the signature
		return errors.New("apple OAuth validation not implemented - requires Apple credentials")
	default:
		return fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
	// return map[string]string{
	//     "id": "oauth_user_id",
	//     "email": "user@example.com",
	//     "name": "User Name",
	//     "picture": "https://...",
	// }, nil
}

// Note: To implement OAuth properly, you need to:
// 1. Register your app with Google Cloud Console (for Google)
// 2. Register your app with Apple Developer (for Apple Sign In)
// 3. Install OAuth libraries: go get google.golang.org/api/oauth2/v2
// 4. Verify tokens server-side before trusting them

// JWT Token Management

var jwtSecret = []byte(getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"))

// Claims represents the JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token for a user (DEPRECATED - use GenerateAccessToken)
// Kept for backward compatibility
func GenerateToken(user *models.User) (string, error) {
	return GenerateAccessToken(user)
}

// GenerateAccessToken generates a short-lived JWT access token for a user
func GenerateAccessToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(models.AccessTokenDuration) // 15 minutes

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// SECURITY: Validate the algorithm to prevent algorithm substitution attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// Helper function to get env variable or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
