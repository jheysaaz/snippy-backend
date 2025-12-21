package database

import (
	"log"
	"time"
)

// RetentionPolicy defines how long to keep different types of data
type RetentionPolicy struct {
	SnippetVersionDays     int // Keep snippet versions for this many days
	SoftDeletedSnippetDays int // Keep soft-deleted snippets for this many days
	SoftDeletedUserDays    int // Keep soft-deleted users for this many days
}

// DefaultRetentionPolicy returns the default retention policy
func DefaultRetentionPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		SnippetVersionDays:     60, // 60 days for snippet versions
		SoftDeletedSnippetDays: 90, // 90 days for soft-deleted snippets
		SoftDeletedUserDays:    30, // 30 days for soft-deleted users
	}
}

// CleanupOldData removes data based on retention policy
func CleanupOldData(policy *RetentionPolicy) error {
	if policy == nil {
		policy = DefaultRetentionPolicy()
	}

	// Calculate cutoff dates
	versionCutoff := time.Now().AddDate(0, 0, -policy.SnippetVersionDays)
	snippetCutoff := time.Now().AddDate(0, 0, -policy.SoftDeletedSnippetDays)
	userCutoff := time.Now().AddDate(0, 0, -policy.SoftDeletedUserDays)

	// 1. Delete old snippet versions (older than 30 days)
	log.Printf("Deleting snippet versions older than %v", versionCutoff)
	result, err := DB.Exec(`
		DELETE FROM snippet_history
		WHERE changed_at < $1
	`, versionCutoff)
	if err != nil {
		log.Printf("Error deleting old snippet versions: %v", err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("Deleted %d old snippet versions", rowsAffected)

	// 2. Permanently delete soft-deleted snippets and their history (older than 90 days)
	log.Printf("Deleting soft-deleted snippets older than %v", snippetCutoff)

	// First delete their history
	result, err = DB.Exec(`
		DELETE FROM snippet_history
		WHERE snippet_id IN (
			SELECT id FROM snippets WHERE is_deleted = true AND deleted_at < $1
		)
	`, snippetCutoff)
	if err != nil {
		log.Printf("Error deleting snippet history for old soft-deleted snippets: %v", err)
		return err
	}
	historyDeleted, _ := result.RowsAffected()

	// Then delete the snippets
	result, err = DB.Exec(`
		DELETE FROM snippets
		WHERE is_deleted = true AND deleted_at < $1
	`, snippetCutoff)
	if err != nil {
		log.Printf("Error deleting old soft-deleted snippets: %v", err)
		return err
	}
	snippetsDeleted, _ := result.RowsAffected()
	log.Printf("Deleted %d old soft-deleted snippets and %d associated history entries", snippetsDeleted, historyDeleted)

	// 3. Permanently delete soft-deleted users and all their associated data (older than 60 days)
	log.Printf("Deleting soft-deleted users older than %v", userCutoff)

	// Get user IDs to delete
	rows, err := DB.Query(`
		SELECT id FROM users WHERE is_deleted = true AND deleted_at < $1
	`, userCutoff)
	if err != nil {
		log.Printf("Error fetching soft-deleted users: %v", err)
		return err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			log.Printf("Error scanning user ID: %v", err)
			continue
		}
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) > 0 {
		// Delete snippets and history first (cascade)
		for _, userID := range userIDs {
			// Delete snippet history first
			result, err := DB.Exec(`
				DELETE FROM snippet_history
				WHERE snippet_id IN (
					SELECT id FROM snippets WHERE user_id = $1
				)
			`, userID)
			if err != nil {
				log.Printf("Error deleting snippet history for user %s: %v", userID, err)
				continue
			}
			historyDeleted, _ := result.RowsAffected()

			// Delete snippets
			result, err = DB.Exec(`DELETE FROM snippets WHERE user_id = $1`, userID)
			if err != nil {
				log.Printf("Error deleting snippets for user %s: %v", userID, err)
				continue
			}
			snippetsDeleted, _ := result.RowsAffected()

			// Delete refresh tokens
			result, err = DB.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
			if err != nil {
				log.Printf("Error deleting refresh tokens for user %s: %v", userID, err)
				continue
			}
			tokensDeleted, _ := result.RowsAffected()

			// Delete user
			result, err = DB.Exec(`DELETE FROM users WHERE id = $1`, userID)
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
