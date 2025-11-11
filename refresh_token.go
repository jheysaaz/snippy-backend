package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"time"
)

// Refresh token errors
var (
	ErrTokenRevoked = errors.New("refresh token has been revoked")
	ErrTokenExpired = errors.New("refresh token has expired")
)

const (
	// AccessTokenDuration - short-lived access token (15 minutes)
	AccessTokenDuration = 15 * time.Minute

	// RefreshTokenDuration - long-lived refresh token (30 days)
	RefreshTokenDuration = 30 * 24 * time.Hour
)

// generateRefreshToken creates a cryptographically secure random token
func generateRefreshToken() (string, error) {
	// 32 bytes = 256 bits of entropy
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Base64 URL-safe encoding
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// storeRefreshToken saves a refresh token to the database
func storeRefreshToken(userID, token, deviceInfo string) error {
	expiresAt := time.Now().Add(RefreshTokenDuration)

	_, err := db.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at, device_info)
		VALUES ($1, $2, $3, $4)
	`, userID, token, expiresAt, deviceInfo)

	return err
}

// validateRefreshToken checks if a refresh token is valid and not expired/revoked
func validateRefreshToken(token string) (*RefreshToken, error) {
	var rt RefreshToken

	err := db.QueryRow(`
		SELECT id, user_id, token, expires_at, created_at, revoked, device_info
		FROM refresh_tokens
		WHERE token = $1
	`, token).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.Token,
		&rt.ExpiresAt,
		&rt.CreatedAt,
		&rt.Revoked,
		&rt.DeviceInfo,
	)

	if err != nil {
		return nil, err
	}

	// Check if token is revoked
	if rt.Revoked {
		return nil, ErrTokenRevoked
	}

	// Check if token is expired
	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return &rt, nil
}

// revokeRefreshToken marks a refresh token as revoked
func revokeRefreshToken(token string) error {
	_, err := db.Exec(`
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE token = $1
	`, token)

	return err
}

// revokeAllUserTokens revokes all refresh tokens for a user
func revokeAllUserTokens(userID string) error {
	_, err := db.Exec(`
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE user_id = $1 AND revoked = FALSE
	`, userID)

	return err
}

// cleanupExpiredTokens removes expired and revoked tokens (run periodically)
func cleanupExpiredTokens() error {
	// Delete tokens that expired more than 7 days ago
	_, err := db.Exec(`
		DELETE FROM refresh_tokens
		WHERE (expires_at < NOW() - INTERVAL '7 days')
		   OR (revoked = TRUE AND created_at < NOW() - INTERVAL '7 days')
	`)

	return err
}

// startTokenCleanupJob runs a background job to cleanup expired tokens
func startTokenCleanupJob() {
	ticker := time.NewTicker(24 * time.Hour) // Run once per day
	defer ticker.Stop()

	// Run immediately on start
	if err := cleanupExpiredTokens(); err != nil {
		log.Printf("Token cleanup failed: %v", err)
	} else {
		log.Println("Token cleanup completed successfully")
	}

	// Then run periodically
	for range ticker.C {
		if err := cleanupExpiredTokens(); err != nil {
			log.Printf("Token cleanup failed: %v", err)
		} else {
			log.Println("Token cleanup completed successfully")
		}
	}
}
