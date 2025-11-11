package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// getUsers retrieves all users
func getUsers(c *gin.Context) {
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	users := make([]User, 0, 10) // Pre-allocate with reasonable capacity
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan user"})
			return
		}
		users = append(users, *user)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users, "count": len(users)})
}

// getUser retrieves a single user by ID
func getUser(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	row := db.QueryRow(query, id)
	user, err := scanUser(row)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// getUserByUsername retrieves a user by username
func getUserByUsername(c *gin.Context) {
	username := c.Param("username")

	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	row := db.QueryRow(query, username)
	user, err := scanUser(row)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// createUser creates a new user
func createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	query := `
		INSERT INTO users (username, email, password_hash, full_name, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
	`

	row := db.QueryRow(
		query,
		req.Username,
		req.Email,
		passwordHash,
		req.FullName,
		req.AvatarURL,
	)

	user, err := scanUser(row)
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "username") {
				c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			} else if strings.Contains(err.Error(), "email") {
				c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
			} else {
				c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// updateUser updates an existing user
func updateUser(c *gin.Context) {
	id := c.Param("id")

	// Get authenticated user ID
	authUserID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if user is updating their own profile
	if authUserID != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own profile"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic UPDATE query based on provided fields
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Username != nil {
		updates = append(updates, "username = $"+strconv.Itoa(argPos))
		args = append(args, *req.Username)
		argPos++
	}
	if req.Email != nil {
		updates = append(updates, "email = $"+strconv.Itoa(argPos))
		args = append(args, *req.Email)
		argPos++
	}
	if req.Password != nil && *req.Password != "" {
		// Hash the new password
		hash, err := HashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
			return
		}
		updates = append(updates, "password_hash = $"+strconv.Itoa(argPos))
		args = append(args, hash)
		argPos++

	}
	if req.FullName != nil {
		updates = append(updates, "full_name = $"+strconv.Itoa(argPos))
		args = append(args, *req.FullName)
		argPos++
	}
	if req.AvatarURL != nil {
		updates = append(updates, "avatar_url = $"+strconv.Itoa(argPos))
		args = append(args, *req.AvatarURL)
		argPos++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add ID as last argument
	args = append(args, id)

	query := `
		UPDATE users
		SET ` + strings.Join(updates, ", ") + `
		WHERE id = $` + strconv.Itoa(argPos) + `
		RETURNING id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
	`

	row := db.QueryRow(query, args...)
	user, err := scanUser(row)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "username") {
				c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			} else if strings.Contains(err.Error(), "email") {
				c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
			} else {
				c.JSON(http.StatusConflict, gin.H{"error": "Duplicate value"})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// deleteUser deletes a user
func deleteUser(c *gin.Context) {
	id := c.Param("id")

	// Get authenticated user ID
	authUserID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if user is deleting their own account
	if authUserID != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own account"})
		return
	}

	query := `DELETE FROM users WHERE id = $1`

	result, err := db.Exec(query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify deletion"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// getUserSnippets retrieves all snippets for a specific user
func getUserSnippets(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT id, title, description, category, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user snippets"})
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

// login handles user login with email and password
func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	row := db.QueryRow(query, req.Email)
	user, err := scanUser(row)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate"})
		return
	}

	// Check password
	if !CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate JWT access token (short-lived)
	accessToken, err := GenerateAccessToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	// Generate refresh token (long-lived)
	refreshToken, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Get device info from user agent (optional)
	deviceInfo := c.GetHeader("User-Agent")

	// Store refresh token in database
	if err := storeRefreshToken(user.ID, refreshToken, deviceInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store refresh token"})
		return
	}

	// Return user info and both tokens
	response := LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(AccessTokenDuration.Seconds()),
	}

	c.JSON(http.StatusOK, response)
}

// refreshAccessToken generates a new access token using a valid refresh token
func refreshAccessToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate refresh token
	rt, err := validateRefreshToken(req.RefreshToken)
	if err != nil {
		if err == ErrTokenExpired {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired, please login again"})
		} else if err == ErrTokenRevoked {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token revoked, please login again"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		}
		return
	}

	// Get user from database
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	row := db.QueryRow(query, rt.UserID)
	user, err := scanUser(row)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	// Generate new access token
	accessToken, err := GenerateAccessToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	// Return new access token
	response := RefreshTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(AccessTokenDuration.Seconds()),
	}

	c.JSON(http.StatusOK, response)
}

// logout revokes the refresh token
func logout(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Revoke the refresh token
	if err := revokeRefreshToken(req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// logoutAll revokes all refresh tokens for the authenticated user
func logoutAll(c *gin.Context) {
	// Get user ID from JWT token in context (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Revoke all tokens for this user
	if err := revokeAllUserTokens(userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout from all devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out from all devices successfully"})
}
