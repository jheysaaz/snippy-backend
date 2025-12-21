// Package handlers exposes HTTP handlers for the API.
package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jheysaaz/snippy-backend/app/database"
	"github.com/jheysaaz/snippy-backend/app/models"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// getSnippets retrieves all snippets with optional filtering
// @Summary List all snippets
// @Description Get all snippets with optional tag, search, and limit filters
// @Tags snippets
// @Accept json
// @Produce json
// @Param tag query string false "Filter by tag"
// @Param search query string false "Search in label"
// @Param limit query int false "Limit results (max 100)"
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
func getSnippets(c *gin.Context) {
	// Optional query parameters for filtering
	tag := c.Query("tag")
	search := c.Query("search")
	limitStr := c.Query("limit")

	// Build query with optional filters
	query := `
		SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE is_deleted = false
	`
	args := []interface{}{}
	argPos := 1

	if tag != "" {
		query += " AND $" + strconv.Itoa(argPos) + " = ANY(tags)"
		args = append(args, tag)
		argPos++
	}

	if search != "" {
		// Use full-text search index for better performance
		query += " AND to_tsvector('english', coalesce(label, '')) @@ plainto_tsquery('english', $" + strconv.Itoa(argPos) + ")"
		args = append(args, search)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	// Add limit if provided (max 100)
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil && limit > 0 {
			if limit > 100 {
				limit = 100 // Cap at 100 for performance
			}
			query += " LIMIT $" + strconv.Itoa(argPos)
			args = append(args, limit)
		}
	}

	rows, err := database.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch snippets")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error closing snippet rows: %v", err)
		}
	}()

	snippets := make([]models.Snippet, 0, 10)
	for rows.Next() {
		snippet, err := models.ScanSnippet(rows)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to scan snippet")
			return
		}
		snippets = append(snippets, *snippet)
	}

	if err := rows.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "Error iterating snippets")
		return
	}

	respondWithCount(c, snippets, len(snippets))
}

// syncSnippets returns snippets changed since a given timestamp for the authenticated user
// @Summary Sync snippets since timestamp
// @Description Returns snippets added/updated and deleted since the given timestamp
// @Tags snippets
// @Produce json
// @Param updated_since query string true "RFC3339 timestamp"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /snippets/sync [get]
func syncSnippets(c *gin.Context) {
	updatedSinceStr := c.Query("updated_since")
	if updatedSinceStr == "" {
		respondError(c, http.StatusBadRequest, "updated_since query param is required")
		return
	}

	updatedSince, err := time.Parse(time.RFC3339, updatedSinceStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "updated_since must be RFC3339 format")
		return
	}

	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Fetch created snippets since the timestamp
	queryCreated := `
		SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE user_id = $1 AND is_deleted = false AND created_at > $2
		ORDER BY created_at ASC
	`

	createdRows, err := database.DB.QueryContext(c.Request.Context(), queryCreated, userID, updatedSince)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch created snippets")
		return
	}
	defer func() {
		if err := createdRows.Close(); err != nil {
			log.Printf("error closing created rows: %v", err)
		}
	}()

	created := make([]models.Snippet, 0, 10)
	for createdRows.Next() {
		snippet, scanErr := models.ScanSnippet(createdRows)
		if scanErr != nil {
			respondError(c, http.StatusInternalServerError, "Failed to scan snippet")
			return
		}
		created = append(created, *snippet)
	}
	if errRows := createdRows.Err(); errRows != nil {
		respondError(c, http.StatusInternalServerError, "Error iterating created snippets")
		return
	}

	// Fetch updated snippets (exclude those newly created to avoid duplicates)
	queryUpdated := `
		SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE user_id = $1 AND is_deleted = false AND updated_at > $2 AND created_at <= $2
		ORDER BY updated_at ASC
	`

	updatedRows, err := database.DB.QueryContext(c.Request.Context(), queryUpdated, userID, updatedSince)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch updated snippets")
		return
	}
	defer func() {
		if err := updatedRows.Close(); err != nil {
			log.Printf("error closing updated rows: %v", err)
		}
	}()

	updated := make([]models.Snippet, 0, 10)
	for updatedRows.Next() {
		snippet, scanErr := models.ScanSnippet(updatedRows)
		if scanErr != nil {
			respondError(c, http.StatusInternalServerError, "Failed to scan snippet")
			return
		}
		updated = append(updated, *snippet)
	}
	if errRows := updatedRows.Err(); errRows != nil {
		respondError(c, http.StatusInternalServerError, "Error iterating updated snippets")
		return
	}

	// Fetch deletions
	queryDeleted := `
		SELECT id, deleted_at
		FROM snippets
		WHERE user_id = $1 AND is_deleted = true AND deleted_at IS NOT NULL AND deleted_at > $2
		ORDER BY deleted_at ASC
	`

	deletedRows, err := database.DB.QueryContext(c.Request.Context(), queryDeleted, userID, updatedSince)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch deleted snippets")
		return
	}
	defer func() {
		if err := deletedRows.Close(); err != nil {
			log.Printf("error closing deleted rows: %v", err)
		}
	}()

	type deletedSnippet struct {
		DeletedAt *time.Time `json:"deletedAt"`
		ID        int64      `json:"id"`
	}

	deleted := make([]deletedSnippet, 0, 10)
	for deletedRows.Next() {
		var item deletedSnippet
		if scanErr := deletedRows.Scan(&item.ID, &item.DeletedAt); scanErr != nil {
			respondError(c, http.StatusInternalServerError, "Failed to scan deleted snippet")
			return
		}
		deleted = append(deleted, item)
	}
	if err := deletedRows.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "Error iterating deleted snippets")
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{
		"created": created,
		"updated": updated,
		"deleted": deleted,
	})
}

// getSnippet retrieves a single snippet by ID
// @Summary Get snippet by ID
// @Description Get a single snippet by its ID
// @Tags snippets
// @Accept json
// @Produce json
// @Param id path int true "Snippet ID"
// @Success 200 {object} models.Snippet
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /snippets/{id} [get]
func getSnippet(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid snippet ID")
		return
	}

	query := `
		SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE id = $1 AND is_deleted = false
	`

	row := database.DB.QueryRowContext(c.Request.Context(), query, id)
	snippet, err := models.ScanSnippet(row)
	if handleScanError(c, err, "Snippet not found") {
		return
	}

	respondSuccess(c, http.StatusOK, snippet)
}

// createSnippet creates a new snippet
// @Summary Create a snippet
// @Description Create a new snippet for the authenticated user
// @Tags snippets
// @Accept json
// @Produce json
// @Param snippet body models.CreateSnippetRequest true "Snippet data"
// @Success 201 {object} models.Snippet
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Security BearerAuth
// @Router /snippets [post]
func createSnippet(c *gin.Context) {
	var req models.CreateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get authenticated user ID from context
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Ensure tags is not nil
	if req.Tags == nil {
		req.Tags = []string{}
	}

	query := `
		INSERT INTO snippets (label, shortcut, content, tags, user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, label, shortcut, content, tags, user_id, created_at, updated_at
	`

	row := database.DB.QueryRowContext(
		c.Request.Context(),
		query,
		req.Label,
		req.Shortcut,
		req.Content,
		pq.Array(req.Tags),
		userID, // Use authenticated user ID
	)

	snippet, err := models.ScanSnippet(row)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create snippet")
		return
	}

	respondSuccess(c, http.StatusCreated, snippet)
}

// buildSnippetUpdateQuery constructs the dynamic UPDATE query for snippet updates
// removed dynamic update builder in favor of static, parameterized UPDATE

// updateSnippet updates an existing snippet
// @Summary Update a snippet
// @Description Update an existing snippet (owner only)
// @Tags snippets
// @Accept json
// @Produce json
// @Param id path int true "Snippet ID"
// @Param snippet body models.UpdateSnippetRequest true "Update data"
// @Success 200 {object} models.Snippet
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /snippets/{id} [put]
func updateSnippet(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid snippet ID")
		return
	}

	// Get authenticated user ID
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Check if snippet exists and belongs to user
	var snippetUserID sql.NullString
	checkQuery := `SELECT user_id FROM snippets WHERE id = $1`
	err = database.DB.QueryRowContext(c.Request.Context(), checkQuery, id).Scan(&snippetUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check snippet ownership"})
		return
	}

	// Verify ownership
	if !snippetUserID.Valid || !checkOwnership(c, snippetUserID.String, userID, "snippet") {
		return
	}

	var req models.UpdateSnippetRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	// Validate that at least one field is provided
	if req.Label == nil && req.Shortcut == nil && req.Content == nil && req.Tags == nil {
		respondError(c, http.StatusBadRequest, "No fields to update")
		return
	}

	// Prepare nullable parameters for static, parameterized UPDATE
	labelVal := valueOrNilString(req.Label)
	shortcutVal := valueOrNilString(req.Shortcut)
	contentVal := valueOrNilString(req.Content)
	tagsVal := arrayOrNilStringSlice(req.Tags)
	changeNotes := valueOrNilString(req.ChangeNotes)

	// Get current snippet data for history
	var currentSnippet models.Snippet
	var tags pq.StringArray
	var currentUserID sql.NullString
	currentQuery := `SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at FROM snippets WHERE id = $1`
	err = database.DB.QueryRowContext(c.Request.Context(), currentQuery, id).Scan(
		&currentSnippet.ID,
		&currentSnippet.Label,
		&currentSnippet.Shortcut,
		&currentSnippet.Content,
		&tags,
		&currentUserID,
		&currentSnippet.CreatedAt,
		&currentSnippet.UpdatedAt,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch current snippet")
		return
	}
	// currentSnippet.Tags = tags // Unused, removed

	// Static UPDATE using COALESCE to only update provided fields
	query := `
		UPDATE snippets
		SET
			label = COALESCE($1, label),
			shortcut = COALESCE($2, shortcut),
			content = COALESCE($3, content),
			tags = COALESCE($4, tags)
		WHERE id = $5 AND is_deleted = false
		RETURNING id, label, shortcut, content, tags, user_id, created_at, updated_at
	`

	row := database.DB.QueryRowContext(c.Request.Context(), query, labelVal, shortcutVal, contentVal, tagsVal, id)
	snippet, err := models.ScanSnippet(row)
	if handleScanError(c, err, "Snippet not found") {
		return
	}

	// Create history entry after successful update
	historyQuery := `
		INSERT INTO snippet_history (
			snippet_id, version_number, label, shortcut, content, tags,
			changed_by, change_type, change_notes
		) VALUES (
			$1, get_next_snippet_version($1), $2, $3, $4, $5, $6, 'edit', $7
		)
	`
	_, err = database.DB.ExecContext(c.Request.Context(), historyQuery,
		snippet.ID,
		snippet.Label,
		snippet.Shortcut,
		snippet.Content,
		pq.Array(snippet.Tags),
		userID,
		changeNotes,
	)
	if err != nil {
		log.Printf("Failed to create snippet history: %v", err)
		// Don't fail the request if history fails
	}

	respondSuccess(c, http.StatusOK, snippet)
}

// deleteSnippet deletes a snippet
// @Summary Delete a snippet
// @Description Delete a snippet (owner only)
// @Tags snippets
// @Accept json
// @Produce json
// @Param id path int true "Snippet ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /snippets/{id} [delete]
func deleteSnippet(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid snippet ID")
		return
	}

	// Get authenticated user ID
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Check if snippet exists and belongs to user
	var snippetUserID sql.NullString
	checkQuery := `SELECT user_id FROM snippets WHERE id = $1`
	err = database.DB.QueryRowContext(c.Request.Context(), checkQuery, id).Scan(&snippetUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check snippet ownership"})
		return
	}

	// Verify ownership
	if !snippetUserID.Valid || !checkOwnership(c, snippetUserID.String, userID, "snippet") {
		return
	}

	// Get snippet data before deletion for history
	var snippet models.Snippet
	var tags pq.StringArray
	var currentUserID sql.NullString
	snippetQuery := `SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at FROM snippets WHERE id = $1`
	err = database.DB.QueryRowContext(c.Request.Context(), snippetQuery, id).Scan(
		&snippet.ID,
		&snippet.Label,
		&snippet.Shortcut,
		&snippet.Content,
		&tags,
		&currentUserID,
		&snippet.CreatedAt,
		&snippet.UpdatedAt,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch snippet")
		return
	}
	snippet.Tags = tags

	query := `UPDATE snippets SET is_deleted = true, deleted_at = NOW() WHERE id = $1 AND is_deleted = false`

	result, err := database.DB.ExecContext(c.Request.Context(), query, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete snippet")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to verify deletion")
		return
	}

	if rowsAffected == 0 {
		respondError(c, http.StatusNotFound, "Snippet not found")
		return
	}

	// Create history entry for soft delete
	historyQuery := `
		INSERT INTO snippet_history (
			snippet_id, version_number, label, shortcut, content, tags,
			changed_by, change_type, change_notes
		) VALUES (
			$1, get_next_snippet_version($1), $2, $3, $4, $5, $6, 'soft_delete', 'Snippet marked as deleted'
		)
	`
	_, err = database.DB.ExecContext(c.Request.Context(), historyQuery,
		snippet.ID,
		snippet.Label,
		snippet.Shortcut,
		snippet.Content,
		pq.Array(snippet.Tags),
		userID,
	)
	if err != nil {
		log.Printf("Failed to create snippet deletion history: %v", err)
		// Don't fail the request if history fails
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "Snippet deleted successfully"})
}

// getSnippetHistory retrieves version history for a snippet
// @Summary Get snippet history
// @Description Get all versions of a snippet (owner only)
// @Tags snippets
// @Produce json
// @Param id path int true "Snippet ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /snippets/{id}/history [get]
func getSnippetHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid snippet ID")
		return
	}

	// Get authenticated user ID
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Check if snippet exists and belongs to user
	var snippetUserID sql.NullString
	checkQuery := `SELECT user_id FROM snippets WHERE id = $1`
	err = database.DB.QueryRowContext(c.Request.Context(), checkQuery, id).Scan(&snippetUserID)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "Snippet not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to check snippet ownership")
		return
	}

	if !snippetUserID.Valid || !checkOwnership(c, snippetUserID.String, userID, "snippet") {
		return
	}

	// Get history
	query := `
		SELECT id, snippet_id, version_number, label, shortcut, content, tags,
		       changed_by, change_type, changed_at, change_notes
		FROM snippet_history
		WHERE snippet_id = $1
		ORDER BY version_number DESC
	`

	rows, err := database.DB.QueryContext(c.Request.Context(), query, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch history")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error closing history rows: %v", err)
		}
	}()

	history := make([]models.SnippetHistory, 0)
	for rows.Next() {
		var h models.SnippetHistory
		var tags pq.StringArray
		var changeNotes sql.NullString

		err := rows.Scan(
			&h.ID,
			&h.SnippetID,
			&h.VersionNumber,
			&h.Label,
			&h.Shortcut,
			&h.Content,
			&tags,
			&h.ChangedBy,
			&h.ChangeType,
			&h.ChangedAt,
			&changeNotes,
		)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to scan history")
			return
		}

		h.Tags = tags
		if changeNotes.Valid {
			h.ChangeNotes = &changeNotes.String
		}

		history = append(history, h)
	}

	if err := rows.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "Error iterating history")
		return
	}

	respondWithCount(c, history, len(history))
}

// restoreSnippetVersion restores a snippet to a previous version
// @Summary Restore snippet version
// @Description Restore a snippet to a specific version (owner only)
// @Tags snippets
// @Accept json
// @Produce json
// @Param id path int true "Snippet ID"
// @Param versionNumber path int true "Version Number"
// @Success 200 {object} models.Snippet
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /snippets/{id}/restore/{versionNumber} [post]
func restoreSnippetVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid snippet ID")
		return
	}

	versionStr := c.Param("versionNumber")
	versionNumber, err := strconv.Atoi(versionStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid version number")
		return
	}

	// Get authenticated user ID
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Check if snippet exists and belongs to user
	var snippetUserID sql.NullString
	checkQuery := `SELECT user_id FROM snippets WHERE id = $1`
	err = database.DB.QueryRowContext(c.Request.Context(), checkQuery, id).Scan(&snippetUserID)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "Snippet not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to check snippet ownership")
		return
	}

	if !snippetUserID.Valid || !checkOwnership(c, snippetUserID.String, userID, "snippet") {
		return
	}

	// Get the historical version
	var historicalVersion models.SnippetHistory
	var tags pq.StringArray
	historyQuery := `
		SELECT label, shortcut, content, tags
		FROM snippet_history
		WHERE snippet_id = $1 AND version_number = $2
	`
	err = database.DB.QueryRowContext(c.Request.Context(), historyQuery, id, versionNumber).Scan(
		&historicalVersion.Label,
		&historicalVersion.Shortcut,
		&historicalVersion.Content,
		&tags,
	)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "Version not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch version")
		return
	}
	historicalVersion.Tags = tags

	// Restore the snippet to this version
	updateQuery := `
		UPDATE snippets
		SET label = $1, shortcut = $2, content = $3, tags = $4, is_deleted = false, deleted_at = NULL
		WHERE id = $5
		RETURNING id, label, shortcut, content, tags, user_id, created_at, updated_at
	`

	row := database.DB.QueryRowContext(c.Request.Context(), updateQuery,
		historicalVersion.Label,
		historicalVersion.Shortcut,
		historicalVersion.Content,
		pq.Array(historicalVersion.Tags),
		id,
	)

	snippet, err := models.ScanSnippet(row)
	if handleScanError(c, err, "Failed to restore snippet") {
		return
	}

	// Create history entry for restore
	historyInsert := `
		INSERT INTO snippet_history (
			snippet_id, version_number, label, shortcut, content, tags,
			changed_by, change_type, change_notes
		) VALUES (
			$1, get_next_snippet_version($1), $2, $3, $4, $5, $6, 'restore', $7
		)
	`
	changeNotes := "Restored to version " + versionStr
	_, err = database.DB.ExecContext(c.Request.Context(), historyInsert,
		snippet.ID,
		snippet.Label,
		snippet.Shortcut,
		snippet.Content,
		pq.Array(snippet.Tags),
		userID,
		changeNotes,
	)
	if err != nil {
		log.Printf("Failed to create restore history: %v", err)
	}

	respondSuccess(c, http.StatusOK, snippet)
}
