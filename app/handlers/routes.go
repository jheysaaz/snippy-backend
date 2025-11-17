package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/jheysaaz/snippy-backend/app/auth"
)

// Public exports for route handlers

// Auth handlers
var (
	Login              = login
	RefreshAccessToken = refreshAccessToken
	Logout             = logout
	LogoutAll          = logoutAll
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
	GetSnippets     = getSnippets
	CreateSnippet   = createSnippet
	GetSnippet      = getSnippet
	UpdateSnippet   = updateSnippet
	DeleteSnippet   = deleteSnippet
	GetUserSnippets = getUserSnippets
)

// GetCurrentUser returns the currently authenticated user
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
func GetCurrentUserSnippets(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	getUserSnippets(c, userID)
}
