package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/jheysaaz/snippy-backend/app/database"
	"github.com/jheysaaz/snippy-backend/app/models"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// getSnippets retrieves all snippets with optional filtering
func getSnippets(c *gin.Context) {
	// Optional query parameters for filtering
	tag := c.Query("tag")
	search := c.Query("search")
	limitStr := c.Query("limit")

	// Build query with optional filters
	query := `
		SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE 1=1
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

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch snippets")
		return
	}
	defer rows.Close()

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

// getSnippet retrieves a single snippet by ID
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
		WHERE id = $1
	`

	row := database.DB.QueryRow(query, id)
	snippet, err := models.ScanSnippet(row)
	if handleScanError(c, err, "Snippet not found") {
		return
	}

	respondSuccess(c, http.StatusOK, snippet)
}

// createSnippet creates a new snippet
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

	row := database.DB.QueryRow(
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
	err = database.DB.QueryRow(checkQuery, id).Scan(&snippetUserID)
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

	// Static UPDATE using COALESCE to only update provided fields
	query := `
		UPDATE snippets
		SET
			label = COALESCE($1, label),
			shortcut = COALESCE($2, shortcut),
			content = COALESCE($3, content),
			tags = COALESCE($4, tags)
		WHERE id = $5
		RETURNING id, label, shortcut, content, tags, user_id, created_at, updated_at
	`

	row := database.DB.QueryRow(query, labelVal, shortcutVal, contentVal, tagsVal, id)
	snippet, err := models.ScanSnippet(row)
	if handleScanError(c, err, "Snippet not found") {
		return
	}

	respondSuccess(c, http.StatusOK, snippet)
}

// deleteSnippet deletes a snippet
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
	err = database.DB.QueryRow(checkQuery, id).Scan(&snippetUserID)
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

	query := `DELETE FROM snippets WHERE id = $1`

	result, err := database.DB.Exec(query, id)
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

	respondSuccess(c, http.StatusOK, gin.H{"message": "Snippet deleted successfully"})
}
