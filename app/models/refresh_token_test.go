package models

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

func TestGenerateRefreshToken(t *testing.T) {
	token1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if len(token1) == 0 {
		t.Error("GenerateRefreshToken() returned empty token")
	}

	// Generate another token to verify uniqueness
	token2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() second call error = %v", err)
	}

	if token1 == token2 {
		t.Error("GenerateRefreshToken() generated duplicate tokens")
	}

	// Verify it's valid base64
	_, err = base64.URLEncoding.DecodeString(token1)
	if err != nil {
		t.Errorf("GenerateRefreshToken() generated invalid base64: %v", err)
	}
}

func TestAccessTokenDuration(t *testing.T) {
	expected := 15 * time.Minute
	if AccessTokenDuration != expected {
		t.Errorf("AccessTokenDuration = %v, want %v", AccessTokenDuration, expected)
	}
}

func TestRefreshTokenDuration(t *testing.T) {
	expected := 3 * 30 * 24 * time.Hour // 90 days
	if RefreshTokenDuration != expected {
		t.Errorf("RefreshTokenDuration = %v, want %v", RefreshTokenDuration, expected)
	}
}

// Test error conditions
func TestGenerateRefreshTokenEntropy(t *testing.T) {
	// Generate multiple tokens to check for randomness
	tokens := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		token, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken() iteration %d error = %v", i, err)
		}

		if tokens[token] {
			t.Errorf("GenerateRefreshToken() generated duplicate token at iteration %d", i)
		}
		tokens[token] = true
	}

	if len(tokens) != iterations {
		t.Errorf("Expected %d unique tokens, got %d", iterations, len(tokens))
	}
}

// Mock test for token validation errors
func TestTokenErrors(t *testing.T) {
	if ErrTokenRevoked == nil {
		t.Error("ErrTokenRevoked should be defined")
	}
	if ErrTokenExpired == nil {
		t.Error("ErrTokenExpired should be defined")
	}

	if ErrTokenRevoked.Error() != "refresh token has been revoked" {
		t.Errorf("ErrTokenRevoked message = %v, want 'refresh token has been revoked'", ErrTokenRevoked.Error())
	}

	if ErrTokenExpired.Error() != "refresh token has expired" {
		t.Errorf("ErrTokenExpired message = %v, want 'refresh token has expired'", ErrTokenExpired.Error())
	}
}

// Test that crypto/rand works properly (edge case)
func TestCryptoRandAvailability(t *testing.T) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		t.Fatalf("crypto/rand.Read() failed: %v - system entropy may be unavailable", err)
	}

	// Check that we got non-zero bytes
	allZero := true
	for _, b := range bytes {
		if b != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		t.Error("crypto/rand.Read() returned all zeros - system entropy issue")
	}
}
