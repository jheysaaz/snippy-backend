package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jheysaaz/snippy-backend/app/auth"
)

// respondError sends a JSON error response
func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// respondSuccess sends a JSON success response
func respondSuccess(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// respondWithCount sends a JSON response with items and count
func respondWithCount(c *gin.Context, items interface{}, count int) {
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"count": count,
	})
}

// getAuthUserID retrieves authenticated user ID or sends unauthorized error
func getAuthUserID(c *gin.Context) (string, bool) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		respondError(c, http.StatusUnauthorized, "Authentication required")
	}
	return userID, exists
}

// handleScanError handles database row scanning errors
func handleScanError(c *gin.Context, err error, notFoundMsg string) bool {
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, notFoundMsg)
		return true
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Database error")
		return true
	}
	return false
}

// checkOwnership verifies resource belongs to authenticated user
func checkOwnership(c *gin.Context, resourceUserID, authUserID string, resourceType string) bool {
	if resourceUserID != authUserID {
		respondError(c, http.StatusForbidden, "You don't have permission to access this "+resourceType)
		return false
	}
	return true
}
