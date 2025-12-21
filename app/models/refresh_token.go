// Package models provides data models and database helpers.
package models

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jheysaaz/snippy-backend/app/database"
)

// Refresh token errors
var (
	ErrTokenRevoked = errors.New("refresh token has been revoked")
	ErrTokenExpired = errors.New("refresh token has expired")
)

const (
	// AccessTokenDuration - short-lived access token (15 minutes)
	AccessTokenDuration = 15 * time.Minute

	// RefreshTokenDuration - long-lived refresh token (90 days)
	RefreshTokenDuration = 3 * 30 * 24 * time.Hour
)

// GenerateRefreshToken creates a cryptographically secure random token.
func GenerateRefreshToken() (string, error) {
	// 32 bytes = 256 bits of entropy
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Base64 URL-safe encoding
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// StoreRefreshToken saves a refresh token to the database.
func StoreRefreshToken(ctx context.Context, userID, token, deviceInfo string) error {
	expiresAt := time.Now().Add(RefreshTokenDuration)

	_, err := database.DB.ExecContext(ctx, `
		INSERT INTO refresh_tokens (user_id, token, expires_at, device_info)
		VALUES ($1, $2, $3, $4)
	`, userID, token, expiresAt, deviceInfo)

	return err
}

// ValidateRefreshToken checks if a refresh token is valid and not expired/revoked.
func ValidateRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	var rt RefreshToken

	err := database.DB.QueryRowContext(ctx, `
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

// RevokeRefreshToken marks a refresh token as revoked.
func RevokeRefreshToken(ctx context.Context, token string) error {
	_, err := database.DB.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE token = $1
	`, token)

	return err
}

// RevokeAllUserTokens revokes all refresh tokens for a user.
func RevokeAllUserTokens(ctx context.Context, userID string) error {
	_, err := database.DB.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE user_id = $1 AND revoked = FALSE
	`, userID)

	return err
}

// CleanupExpiredTokens removes expired and revoked tokens (run periodically).
func CleanupExpiredTokens(ctx context.Context) error {
	// Delete tokens that expired more than 7 days ago
	_, err := database.DB.ExecContext(ctx, `
		DELETE FROM refresh_tokens
		WHERE (expires_at < NOW() - INTERVAL '7 days')
		   OR (revoked = TRUE AND created_at < NOW() - INTERVAL '7 days')
	`)

	return err
}
