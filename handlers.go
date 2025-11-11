package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// getSnippets retrieves all snippets with optional filtering
func getSnippets(c *gin.Context) {
	// Optional query parameters for filtering
	category := c.Query("category")
	tag := c.Query("tag")
	search := c.Query("search")
	limitStr := c.Query("limit")

	// Build query with optional filters
	query := `
		SELECT id, title, description, category, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if category != "" {
		query += " AND category = $" + strconv.Itoa(argPos)
		args = append(args, category)
		argPos++
	}

	if tag != "" {
		query += " AND $" + strconv.Itoa(argPos) + " = ANY(tags)"
		args = append(args, tag)
		argPos++
	}

	if search != "" {
		// Use full-text search index for better performance
		query += " AND to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '')) @@ plainto_tsquery('english', $" + strconv.Itoa(argPos) + ")"
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

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch snippets"})
		return
	}
	defer rows.Close()

	snippets := make([]Snippet, 0, 10) // Pre-allocate with reasonable capacity
	for rows.Next() {
		snippet, err := scanSnippet(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan snippet"})
			return
		}
		snippets = append(snippets, *snippet)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating snippets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snippets": snippets, "count": len(snippets)})
}

// getSnippet retrieves a single snippet by ID
func getSnippet(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid snippet ID"})
		return
	}

	query := `
		SELECT id, title, description, category, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE id = $1
	`

	row := db.QueryRow(query, id)
	snippet, err := scanSnippet(row)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch snippet"})
		return
	}

	c.JSON(http.StatusOK, snippet)
}

// createSnippet creates a new snippet
func createSnippet(c *gin.Context) {
	var req CreateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get authenticated user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Ensure tags is not nil
	if req.Tags == nil {
		req.Tags = []string{}
	}

	query := `
		INSERT INTO snippets (title, description, category, shortcut, content, tags, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, description, category, shortcut, content, tags, user_id, created_at, updated_at
	`

	row := db.QueryRow(
		query,
		req.Title,
		req.Description,
		req.Category,
		req.Shortcut,
		req.Content,
		pq.Array(req.Tags),
		userID, // Use authenticated user ID
	)

	snippet, err := scanSnippet(row)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create snippet"})
		return
	}

	c.JSON(http.StatusCreated, snippet)
}

// buildSnippetUpdateQuery constructs the dynamic UPDATE query for snippet updates
func buildSnippetUpdateQuery(req *UpdateSnippetRequest) ([]string, []interface{}) {
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Title != nil {
		updates = append(updates, "title = $"+strconv.Itoa(argPos))
		args = append(args, *req.Title)
		argPos++
	}
	if req.Description != nil {
		updates = append(updates, "description = $"+strconv.Itoa(argPos))
		args = append(args, *req.Description)
		argPos++
	}
	if req.Category != nil {
		updates = append(updates, "category = $"+strconv.Itoa(argPos))
		args = append(args, *req.Category)
		argPos++
	}
	if req.Shortcut != nil {
		updates = append(updates, "shortcut = $"+strconv.Itoa(argPos))
		args = append(args, *req.Shortcut)
		argPos++
	}
	if req.Content != nil {
		updates = append(updates, "content = $"+strconv.Itoa(argPos))
		args = append(args, *req.Content)
		argPos++
	}
	if req.Tags != nil {
		updates = append(updates, "tags = $"+strconv.Itoa(argPos))
		args = append(args, pq.Array(req.Tags))
	}

	return updates, args
}

// updateSnippet updates an existing snippet
func updateSnippet(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid snippet ID"})
		return
	}

	// Get authenticated user ID
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if snippet exists and belongs to user
	var snippetUserID sql.NullString
	checkQuery := `SELECT user_id FROM snippets WHERE id = $1`
	err = db.QueryRow(checkQuery, id).Scan(&snippetUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check snippet ownership"})
		return
	}

	// Verify ownership
	if !snippetUserID.Valid || snippetUserID.String != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this snippet"})
		return
	}

	var req UpdateSnippetRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	// Build dynamic UPDATE query based on provided fields
	updates, args := buildSnippetUpdateQuery(&req)

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add ID as last argument
	args = append(args, id)
	argPos := len(args)

	query := `
		UPDATE snippets
		SET ` + strings.Join(updates, ", ") + `
		WHERE id = $` + strconv.Itoa(argPos) + `
		RETURNING id, title, description, category, shortcut, content, tags, user_id, created_at, updated_at
	`

	row := db.QueryRow(query, args...)
	snippet, err := scanSnippet(row)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update snippet"})
		return
	}

	c.JSON(http.StatusOK, snippet)
}

// deleteSnippet deletes a snippet
func deleteSnippet(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid snippet ID"})
		return
	}

	// Get authenticated user ID
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if snippet exists and belongs to user
	var snippetUserID sql.NullString
	checkQuery := `SELECT user_id FROM snippets WHERE id = $1`
	err = db.QueryRow(checkQuery, id).Scan(&snippetUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check snippet ownership"})
		return
	}

	// Verify ownership
	if !snippetUserID.Valid || snippetUserID.String != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this snippet"})
		return
	}

	query := `DELETE FROM snippets WHERE id = $1`

	result, err := db.Exec(query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete snippet"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify deletion"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Snippet deleted successfully"})
}
