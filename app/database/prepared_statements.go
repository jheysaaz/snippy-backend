// Package database provides prepared statements for commonly used queries.
package database

import (
	"context"
	"database/sql"
	"log"
	"sync"
)

// PreparedStatements holds prepared statements for frequently used queries
type PreparedStatements struct {
	// Ownership check - used in update, delete, history, restore endpoints
	SnippetOwnership *sql.Stmt
}

var (
	preparedStmts *PreparedStatements
	prepareOnce   sync.Once
	prepareErr    error
)

// InitPreparedStatements initializes prepared statements after DB is ready.
// Call this after database.Init() in main.go
func InitPreparedStatements() error {
	prepareOnce.Do(func() {
		preparedStmts, prepareErr = initPreparedStatements()
		if prepareErr != nil {
			log.Printf("Warning: Failed to initialize prepared statements: %v", prepareErr)
		} else {
			log.Println("Prepared statements initialized successfully")
		}
	})
	return prepareErr
}

// GetPreparedStatements returns the singleton prepared statements instance
func GetPreparedStatements() *PreparedStatements {
	return preparedStmts
}

// initPreparedStatements creates all prepared statements
func initPreparedStatements() (*PreparedStatements, error) {
	if DB == nil {
		return nil, nil // DB not initialized yet
	}

	ctx := context.Background()
	stmts := &PreparedStatements{}

	var err error

	// Snippet ownership check - used in update, delete, history, restore
	// This query is called frequently, preparing it saves parsing overhead
	stmts.SnippetOwnership, err = DB.PrepareContext(ctx, `
		SELECT user_id FROM snippets WHERE id = $1
	`)
	if err != nil {
		return nil, err
	}

	return stmts, nil
}

// CheckSnippetOwnership checks if a snippet exists and returns its owner.
// Uses prepared statement for better performance on repeated calls.
func CheckSnippetOwnership(ctx context.Context, snippetID int64) (ownerID sql.NullString, err error) {
	stmts := GetPreparedStatements()
	if stmts != nil && stmts.SnippetOwnership != nil {
		// Use prepared statement
		err = stmts.SnippetOwnership.QueryRowContext(ctx, snippetID).Scan(&ownerID)
	} else {
		// Fallback to non-prepared query
		err = DB.QueryRowContext(ctx, `SELECT user_id FROM snippets WHERE id = $1`, snippetID).Scan(&ownerID)
	}
	return
}

// Close closes all prepared statements. Call on graceful shutdown.
func (ps *PreparedStatements) Close() error {
	if ps == nil {
		return nil
	}

	if ps.SnippetOwnership != nil {
		if err := ps.SnippetOwnership.Close(); err != nil {
			return err
		}
	}

	return nil
}
