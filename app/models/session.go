// Package models provides data models and database helpers.
package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jheysaaz/snippy-backend/app/database"
)

// CreateSession creates a new user session.
func CreateSession(ctx context.Context, userID, deviceInfo, ipAddress, userAgent, refreshTokenID string) (*Session, error) {
	// Hash the IP address for privacy
	ipHash := hashIP(ipAddress)

	expiresAt := timePtr(time.Now().Add(RefreshTokenDuration))

	query := `
		INSERT INTO sessions (user_id, device_info, ip_address_hash, user_agent, refresh_token_id, active, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, device_info, ip_address_hash, user_agent, refresh_token_id, active, last_activity, created_at, expires_at, logged_out_at
	`

	row := database.DB.QueryRowContext(ctx, query, userID, deviceInfo, ipHash, userAgent, refreshTokenID, true, expiresAt)
	return scanSession(row)
}

// GetUserSessions gets all active sessions for a user.
func GetUserSessions(ctx context.Context, userID string) ([]Session, error) {
	query := `
		SELECT id, user_id, device_info, ip_address_hash, user_agent, refresh_token_id, active, last_activity, created_at, expires_at, logged_out_at
		FROM sessions
		WHERE user_id = $1 AND active = true
		ORDER BY last_activity DESC
	`

	rows, err := database.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// log closing error to avoid empty-block revive warnings
			fmt.Printf("error closing session rows: %v\n", err)
		}
	}()

	sessions := make([]Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}

	return sessions, rows.Err()
}

// GetSessionByID gets a specific session.
func GetSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	query := `
		SELECT id, user_id, device_info, ip_address_hash, user_agent, refresh_token_id, active, last_activity, created_at, expires_at, logged_out_at
		FROM sessions
		WHERE id = $1
	`

	row := database.DB.QueryRowContext(ctx, query, sessionID)
	return scanSession(row)
}

// UpdateSessionActivity updates the last activity timestamp for a session.
func UpdateSessionActivity(ctx context.Context, sessionID string) error {
	query := `UPDATE sessions SET last_activity = NOW() WHERE id = $1`
	_, err := database.DB.ExecContext(ctx, query, sessionID)
	return err
}

// LogoutSession marks a session as inactive.
func LogoutSession(ctx context.Context, sessionID string) error {
	query := `UPDATE sessions SET active = false, logged_out_at = NOW() WHERE id = $1`
	_, err := database.DB.ExecContext(ctx, query, sessionID)
	return err
}

// LogoutAllUserSessions marks all sessions for a user as inactive.
func LogoutAllUserSessions(ctx context.Context, userID string) error {
	query := `UPDATE sessions SET active = false, logged_out_at = NOW() WHERE user_id = $1 AND active = true`
	_, err := database.DB.ExecContext(ctx, query, userID)
	return err
}

// DeleteExpiredSessions permanently deletes expired sessions.
func DeleteExpiredSessions(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < NOW() OR (logged_out_at IS NOT NULL AND logged_out_at < NOW() - INTERVAL '30 days')`
	result, err := database.DB.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// LogoutIdleSessions logs out sessions that have been idle for more than the specified days.
func LogoutIdleSessions(ctx context.Context, idleDays int) (int64, error) {
	query := `
		UPDATE sessions 
		SET active = false, logged_out_at = NOW()
		WHERE active = true AND last_activity < NOW() - INTERVAL '%d days'
	`
	result, err := database.DB.ExecContext(ctx, fmt.Sprintf(query, idleDays))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// scanSession scans a database row into a Session struct
func scanSession(scanner interface {
	Scan(dest ...interface{}) error
}) (*Session, error) {
	var session Session

	err := scanner.Scan(
		&session.ID,
		&session.UserID,
		&session.DeviceInfo,
		&session.IPAddressHash,
		&session.UserAgent,
		&session.RefreshTokenID,
		&session.Active,
		&session.LastActivity,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LoggedOutAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}

	return &session, nil
}

// hashIP hashes an IP address for privacy
func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}
