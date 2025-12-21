package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/jheysaaz/snippy-backend/app/auth"
)

// Public exports for route handlers

// Auth handlers
var (
	Login              = login
	CheckAvailability  = checkAvailability
	RefreshAccessToken = refreshAccessToken
	Logout             = logout
	LogoutAll          = logoutAll
	GetSessions        = getSessions
	LogoutSession      = logoutSession
)

// User handlers
var (
	GetUsers   = getUsers
	CreateUser = createUser
	GetUser    = getUser
	UpdateUser = updateUser
	DeleteUser = deleteUser
)

// Snippet handlers
var (
	GetSnippets           = getSnippets
	SyncSnippets          = syncSnippets
	CreateSnippet         = createSnippet
	GetSnippet            = getSnippet
	UpdateSnippet         = updateSnippet
	DeleteSnippet         = deleteSnippet
	GetUserSnippets       = getUserSnippets
	GetSnippetHistory     = getSnippetHistory
	RestoreSnippetVersion = restoreSnippetVersion
)

// GetCurrentUser returns the currently authenticated user
// @Summary Get current user profile
// @Description Get the profile of the authenticated user
// @Tags users
// @Produce json
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]string
// @Security BearerAuth
// @Router /users/profile [get]
func GetCurrentUser(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	c.Params = []gin.Param{{Key: "id", Value: userID}}
	getUser(c)
}

// UpdateCurrentUser updates the currently authenticated user
// @Summary Update current user profile
// @Description Update the profile of the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Param user body models.UpdateUserRequest true "Update data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Security BearerAuth
// @Router /users/profile [put]
func UpdateCurrentUser(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	c.Params = []gin.Param{{Key: "id", Value: userID}}
	updateUser(c)
}

// GetCurrentUserSnippets returns snippets for the currently authenticated user
// @Summary Get current user's snippets
// @Description Get all snippets belonging to the authenticated user
// @Tags snippets
// @Produce json
// @Param tag query string false "Filter by tag"
// @Param search query string false "Search in label"
// @Param limit query int false "Limit results (max 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Security BearerAuth
// @Router /snippets [get]
func GetCurrentUserSnippets(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	getUserSnippets(c, userID)
}
