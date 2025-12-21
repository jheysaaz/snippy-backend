// Package middleware provides role-based authorization.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jheysaaz/snippy-backend/app/models"
)

// RequireRole middleware ensures the authenticated user has the specified role.
func RequireRole(roleName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userIDStr, ok := userID.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
			c.Abort()
			return
		}

		hasRole, err := models.HasRole(c.Request.Context(), userIDStr, roleName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user role"})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole middleware ensures the authenticated user has at least one of the specified roles.
func RequireAnyRole(roleNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userIDStr, ok := userID.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
			c.Abort()
			return
		}

		hasRole, err := models.HasAnyRole(c.Request.Context(), userIDStr, roleNames)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user roles"})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminOnly is a convenience middleware that requires admin role.
var AdminOnly = RequireRole(models.RoleAdmin)

// PremiumOnly is a convenience middleware that requires premium role.
var PremiumOnly = RequireRole(models.RolePremium)

// TesterOrAdmin is a convenience middleware that requires tester or admin role.
var TesterOrAdmin = RequireAnyRole(models.RoleTester, models.RoleAdmin)

// RequirePermission middleware ensures the user has a specific permission.
// Permissions are feature flags that map to roles (e.g., "sessions_access").
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userIDStr, ok := userID.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
			c.Abort()
			return
		}

		hasPermission, err := models.HasPermission(c.Request.Context(), userIDStr, permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SessionsAccess requires sessions_access permission (tester, premium, or admin).
var SessionsAccess = RequirePermission("sessions_access")
