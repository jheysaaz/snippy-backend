// Package handlers provides role management endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jheysaaz/snippy-backend/app/models"
)

// GetUserRoles retrieves all roles for a specific user
// @Summary Get user roles
// @Description Get all roles assigned to a user (admin only)
// @Tags roles
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /admin/users/{userId}/roles [get]
func getUserRoles(c *gin.Context) {
	userID := c.Param("userId")

	roles, err := models.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch user roles")
		return
	}

	respondWithCount(c, roles, len(roles))
}

// AssignUserRole assigns a role to a user
// @Summary Assign role to user
// @Description Assign a role to a user (admin only)
// @Tags roles
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param role body object{roleName:string} true "Role to assign"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /admin/users/{userId}/roles [post]
func assignUserRole(c *gin.Context) {
	userID := c.Param("userId")

	var req struct {
		RoleName string `json:"roleName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Role name is required")
		return
	}

	// Get the admin user ID who is assigning the role
	adminUserID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	err := models.AssignRole(c.Request.Context(), userID, req.RoleName, &adminUserID)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{
		"message": "Role assigned successfully",
		"userId":  userID,
		"role":    req.RoleName,
	})
}

// RevokeUserRole removes a role from a user
// @Summary Revoke role from user
// @Description Remove a role from a user (admin only)
// @Tags roles
// @Produce json
// @Param userId path string true "User ID"
// @Param roleName path string true "Role name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /admin/users/{userId}/roles/{roleName} [delete]
func revokeUserRole(c *gin.Context) {
	userID := c.Param("userId")
	roleName := c.Param("roleName")

	err := models.RevokeRole(c.Request.Context(), userID, roleName)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{
		"message": "Role revoked successfully",
		"userId":  userID,
		"role":    roleName,
	})
}

// GetMyRoles retrieves roles for the authenticated user
// @Summary Get my roles
// @Description Get roles for the authenticated user
// @Tags roles
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /users/me/roles [get]
func getMyRoles(c *gin.Context) {
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	roles, err := models.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch roles")
		return
	}

	respondWithCount(c, roles, len(roles))
}

// GetAllRoles retrieves all available roles in the system
// @Summary Get all roles
// @Description Get all available roles (public)
// @Tags roles
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /roles [get]
func getAllRoles(c *gin.Context) {
	roles, err := models.GetAllRoles(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch roles")
		return
	}

	respondWithCount(c, roles, len(roles))
}
