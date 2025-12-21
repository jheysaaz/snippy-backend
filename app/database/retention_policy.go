// Package database contains retention policy utilities.
package database

import (
	"context"
	"log"
	"time"
)

// RetentionPolicy defines how long to keep different types of data
type RetentionPolicy struct {
	SnippetVersionDays     int // Keep snippet versions for this many days
	SoftDeletedSnippetDays int // Keep soft-deleted snippets for this many days
	SoftDeletedUserDays    int // Keep soft-deleted users for this many days
	IdleSessionDays        int // Auto-logout sessions idle for this many days
}

// DefaultRetentionPolicy returns the default retention policy
func DefaultRetentionPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		SnippetVersionDays:     60, // 60 days for snippet versions
		SoftDeletedSnippetDays: 90, // 90 days for soft-deleted snippets
		SoftDeletedUserDays:    30, // 30 days for soft-deleted users
		IdleSessionDays:        7,  // 7 days of inactivity before auto-logout
	}
}

// CleanupOldData removes data based on retention policy
func CleanupOldData(policy *RetentionPolicy) error {
	if policy == nil {
		policy = DefaultRetentionPolicy()
	}

	ctx := context.Background()

	// Calculate cutoff dates
	versionCutoff := time.Now().AddDate(0, 0, -policy.SnippetVersionDays)
	snippetCutoff := time.Now().AddDate(0, 0, -policy.SoftDeletedSnippetDays)
	userCutoff := time.Now().AddDate(0, 0, -policy.SoftDeletedUserDays)

	// 0. Auto-logout idle sessions
	log.Printf("Logging out sessions idle for more than %d days", policy.IdleSessionDays)
	result, err := DB.ExecContext(ctx, `
		UPDATE sessions 
		SET active = false, logged_out_at = NOW()
		WHERE active = true AND last_activity < NOW() - INTERVAL '%d days'
	`, policy.IdleSessionDays)
	if err != nil {
		log.Printf("Error logging out idle sessions: %v", err)
		// Don't return error, continue with other cleanup
	} else {
		rowsAffected, errRows := result.RowsAffected()
		if errRows != nil {
			log.Printf("Error getting rows affected: %v", errRows)
		} else {
			log.Printf("Logged out %d idle sessions", rowsAffected)
		}
	}

	// 0.1 Cleanup expired and old revoked refresh tokens
	if _, err := DB.ExecContext(ctx, `
		DELETE FROM refresh_tokens
		WHERE (expires_at < NOW() - INTERVAL '7 days')
		   OR (revoked = TRUE AND created_at < NOW() - INTERVAL '7 days')
	`); err != nil {
		log.Printf("Error cleaning up expired/revoked refresh tokens: %v", err)
	}

	// 1. Delete old snippet versions (older than 60 days)
	log.Printf("Deleting snippet versions older than %v", versionCutoff)
	result, err = DB.ExecContext(ctx, `
		DELETE FROM snippet_history
		WHERE changed_at < $1
	`, versionCutoff)
	if err != nil {
		log.Printf("Error deleting old snippet versions: %v", err)
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
	} else {
		log.Printf("Deleted %d old snippet versions", rowsAffected)
	}

	// 2. Permanently delete soft-deleted snippets and their history (older than 90 days)
	log.Printf("Deleting soft-deleted snippets older than %v", snippetCutoff)

	// First delete their history
	result, err = DB.ExecContext(ctx, `
		DELETE FROM snippet_history
		WHERE snippet_id IN (
			SELECT id FROM snippets WHERE is_deleted = true AND deleted_at < $1
		)
	`, snippetCutoff)
	if err != nil {
		log.Printf("Error deleting snippet history for old soft-deleted snippets: %v", err)
		return err
	}
	historyDeleted, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected for history: %v", err)
		historyDeleted = 0
	}

	// Then delete the snippets
	result, err = DB.ExecContext(ctx, `
		DELETE FROM snippets
		WHERE is_deleted = true AND deleted_at < $1
	`, snippetCutoff)
	if err != nil {
		log.Printf("Error deleting old soft-deleted snippets: %v", err)
		return err
	}
	snippetsDeleted, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
	} else {
		log.Printf("Deleted %d old soft-deleted snippets and %d associated history entries", snippetsDeleted, historyDeleted)
	}

	// 3. Permanently delete soft-deleted users and all their associated data (older than 60 days)
	log.Printf("Deleting soft-deleted users older than %v", userCutoff)

	// Get user IDs to delete
	rows, err := DB.QueryContext(ctx, `
		SELECT id FROM users WHERE is_deleted = true AND deleted_at < $1
	`, userCutoff)
	if err != nil {
		log.Printf("Error fetching soft-deleted users: %v", err)
		return err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("Error closing user rows: %v", cerr)
		}
	}()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			log.Printf("Error scanning user ID: %v", err)
			continue
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating user rows: %v", err)
		return err
	}

	if len(userIDs) > 0 {
		// Delete snippets and history first (cascade)
		for _, userID := range userIDs {
			// Delete snippet history first
			result, err := DB.ExecContext(ctx, `
				DELETE FROM snippet_history
				WHERE snippet_id IN (
					SELECT id FROM snippets WHERE user_id = $1
				)
			`, userID)
			if err != nil {
				log.Printf("Error deleting snippet history for user %s: %v", userID, err)
				continue
			}
			historyDeleted, errH := result.RowsAffected()
			if errH != nil {
				historyDeleted = 0
			}

			// Delete snippets
			result, err = DB.ExecContext(ctx, `DELETE FROM snippets WHERE user_id = $1`, userID)
			if err != nil {
				log.Printf("Error deleting snippets for user %s: %v", userID, err)
				continue
			}
			snippetsDeleted, errS := result.RowsAffected()
			if errS != nil {
				snippetsDeleted = 0
			}

			// Delete refresh tokens
			result, err = DB.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
			if err != nil {
				log.Printf("Error deleting refresh tokens for user %s: %v", userID, err)
				continue
			}
			tokensDeleted, errT := result.RowsAffected()
			if errT != nil {
				tokensDeleted = 0
			}

			// Delete user
			_, err = DB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
			if err != nil {
				log.Printf("Error deleting user %s: %v", userID, err)
				continue
			}

			log.Printf("Deleted user %s with %d snippets, %d history entries, %d refresh tokens",
				userID, snippetsDeleted, historyDeleted, tokensDeleted)
		}
	}

	log.Println("Data cleanup completed successfully")
	return nil
}
