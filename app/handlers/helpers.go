package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jheysaaz/snippy-backend/app/auth"
	"github.com/lib/pq"
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

// ensureOwnAccount verifies the authenticated user matches the target user ID
func ensureOwnAccount(c *gin.Context, targetUserID string) bool {
	userID, exists := getAuthUserID(c)
	if !exists {
		return false
	}
	if !checkOwnership(c, targetUserID, userID, "profile") {
		return false
	}
	return true
}

// valueOrNilString converts a *string into a driver-compatible value or nil
func valueOrNilString(p *string) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

// arrayOrNilStringSlice converts a []string into a pq.Array or nil
func arrayOrNilStringSlice(s []string) interface{} {
	if s == nil {
		return nil
	}
	return pq.Array(s)
}

// hashedPasswordOrNil returns a hashed password when provided, else nil
func hashedPasswordOrNil(p *string) (interface{}, error) {
	if p == nil || *p == "" {
		return nil, nil
	}
	return auth.HashPassword(*p)
}

// handleUserUniqueViolation translates DB unique constraint errors for users
func handleUserUniqueViolation(c *gin.Context, err error) bool {
	if !strings.Contains(err.Error(), "duplicate key") {
		return false
	}
	switch {
	case strings.Contains(err.Error(), "username"):
		respondError(c, http.StatusConflict, "Username already exists")
	case strings.Contains(err.Error(), "email"):
		respondError(c, http.StatusConflict, "Email already exists")
	default:
		respondError(c, http.StatusConflict, "Duplicate value")
	}
	return true
}
