package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/jheysaaz/snippy-backend/app/auth"
	"github.com/jheysaaz/snippy-backend/app/database"
	"github.com/jheysaaz/snippy-backend/app/models"

	"github.com/gin-gonic/gin"
)

// getUsers retrieves all users
// @Summary List all users
// @Description Get all users
// @Tags users
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /users [get]
func getUsers(c *gin.Context) {
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at, is_deleted, deleted_at
		FROM users
		WHERE is_deleted = false
		ORDER BY created_at DESC
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer rows.Close()

	users := make([]models.User, 0, 10)
	for rows.Next() {
		user, err := models.ScanUser(rows)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to scan user")
			return
		}
		users = append(users, *user)
	}

	if err := rows.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "Error iterating users")
		return
	}

	respondWithCount(c, users, len(users))
}

// getUser retrieves a single user by ID
// @Summary Get user by ID
// @Description Get a single user by ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id} [get]
func getUser(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	row := database.DB.QueryRow(query, id)
	user, err := models.ScanUser(row)
	if handleScanError(c, err, "User not found") {
		return
	}

	respondSuccess(c, http.StatusOK, user)
}

// createUser creates a new user
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.CreateUserRequest true "User data"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /auth/register [post]
func createUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to process password")
		return
	}

	query := `
		INSERT INTO users (username, email, password_hash, full_name, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, email, password_hash, full_name, avatar_url, created_at, updated_at, is_deleted, deleted_at
	`

	row := database.DB.QueryRow(
		query,
		req.Username,
		req.Email,
		passwordHash,
		req.FullName,
		req.AvatarURL,
	)

	user, err := models.ScanUser(row)
	if err != nil {
		if handleUserUniqueViolation(c, err) {
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	respondSuccess(c, http.StatusCreated, user)
}

// removed dynamic update builder in favor of static, parameterized UPDATE

// updateUser updates an existing user
// @Summary Update user
// @Description Update user profile (own account only)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body models.UpdateUserRequest true "Update data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id} [put]
func updateUser(c *gin.Context) {
	id := c.Param("id")

	// Ensure the authenticated user is updating their own profile
	if !ensureOwnAccount(c, id) {
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that at least one field is provided
	if req.Username == nil && req.Email == nil && (req.Password == nil || *req.Password == "") && req.FullName == nil && req.AvatarURL == nil {
		respondError(c, http.StatusBadRequest, "No fields to update")
		return
	}

	// Prepare nullable parameters for static, parameterized UPDATE
	usernameVal := valueOrNilString(req.Username)
	emailVal := valueOrNilString(req.Email)
	passwordHashVal, hashErr := hashedPasswordOrNil(req.Password)
	if hashErr != nil {
		respondError(c, http.StatusInternalServerError, "Failed to process password")
		return
	}
	fullNameVal := valueOrNilString(req.FullName)
	avatarURLVal := valueOrNilString(req.AvatarURL)

	query := `
		UPDATE users
		SET
			username = COALESCE($1, username),
			email = COALESCE($2, email),
			password_hash = COALESCE($3, password_hash),
			full_name = COALESCE($4, full_name),
			avatar_url = COALESCE($5, avatar_url)
		WHERE id = $6
		RETURNING id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
	`

	row := database.DB.QueryRow(query, usernameVal, emailVal, passwordHashVal, fullNameVal, avatarURLVal, id)
	user, err := models.ScanUser(row)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		if handleUserUniqueViolation(c, err) {
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to update user")
		return
	}

	respondSuccess(c, http.StatusOK, user)
}

// deleteUser deletes a user
// @Summary Delete user
// @Description Delete user account (own account only)
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /users/{id} [delete]
func deleteUser(c *gin.Context) {
	id := c.Param("id")

	// Get authenticated user ID
	authUserID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Check if user is deleting their own account
	if !checkOwnership(c, id, authUserID, "account") {
		return
	}

	query := `UPDATE users SET is_deleted = true, deleted_at = NOW() WHERE id = $1 AND is_deleted = false`

	result, err := database.DB.Exec(query, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to verify deletion")
		return
	}

	if rowsAffected == 0 {
		respondError(c, http.StatusNotFound, "User not found")
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// getUserSnippets retrieves all snippets for a specific user
func getUserSnippets(c *gin.Context, userID string) {

	// Optional query parameters for filtering
	tag := c.Query("tag")
	search := c.Query("search")
	limitStr := c.Query("limit")

	// Build query with optional filters
	query := `
		SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at, is_deleted, deleted_at
		FROM snippets
		WHERE user_id = $1 AND is_deleted = false
	`
	args := []interface{}{userID}
	argPos := 2

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
		respondError(c, http.StatusInternalServerError, "Failed to fetch user snippets")
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

// login handles user login with username or email and password
// @Summary Login
// @Description Authenticate with username/email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Query for user by username OR email
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at, is_deleted, deleted_at
		FROM users
		WHERE (username = $1 OR email = $1) AND is_deleted = false
	`

	row := database.DB.QueryRow(query, req.Login)
	user, err := models.ScanUser(row)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusUnauthorized, "Invalid username/email or password")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to authenticate")
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		respondError(c, http.StatusUnauthorized, "Invalid username/email or password")
		return
	}

	// Generate JWT access token (short-lived)
	accessToken, err := auth.GenerateAccessToken(user)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to generate access token")
		return
	}

	// Generate refresh token (long-lived)
	refreshToken, err := models.GenerateRefreshToken()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	// Get device info from user agent (optional)
	deviceInfo := c.GetHeader("User-Agent")

	// Store refresh token in database
	if err := models.StoreRefreshToken(user.ID, refreshToken, deviceInfo); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to store refresh token")
		return
	}

	// Return user info and both tokens
	response := models.LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(models.AccessTokenDuration.Seconds()),
	}

	respondSuccess(c, http.StatusOK, response)
}

// refreshAccessToken generates a new access token using a valid refresh token
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param token body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} models.RefreshTokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/refresh [post]
func refreshAccessToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate refresh token
	rt, err := models.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		switch err {
		case models.ErrTokenExpired:
			respondError(c, http.StatusUnauthorized, "Refresh token expired, please login again")
		case models.ErrTokenRevoked:
			respondError(c, http.StatusUnauthorized, "Refresh token revoked, please login again")
		default:
			respondError(c, http.StatusUnauthorized, "Invalid refresh token")
		}
		return
	}

	// Get user from database
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at, is_deleted, deleted_at
		FROM users
		WHERE id = $1 AND is_deleted = false
	`

	row := database.DB.QueryRow(query, rt.UserID)
	user, err := models.ScanUser(row)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusUnauthorized, "User not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	// Generate new access token
	accessToken, err := auth.GenerateAccessToken(user)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to generate access token")
		return
	}

	// Return new access token
	response := models.RefreshTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(models.AccessTokenDuration.Seconds()),
	}

	respondSuccess(c, http.StatusOK, response)
}

// logout revokes the refresh token
// @Summary Logout
// @Description Revoke the refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param token body models.RefreshTokenRequest true "Refresh token to revoke"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /auth/logout [post]
func logout(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Revoke the refresh token
	if err := models.RevokeRefreshToken(req.RefreshToken); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// logoutAll revokes all refresh tokens for the authenticated user
// @Summary Logout from all devices
// @Description Revoke all refresh tokens for the authenticated user
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /auth/logout-all [post]
func logoutAll(c *gin.Context) {
	// Get user ID from JWT token in context (set by AuthMiddleware)
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Revoke all tokens for this user
	if err := models.RevokeAllUserTokens(userID); err != nil {
		log.Printf("Failed to revoke all tokens for user %v: %v", userID, err)
		respondError(c, http.StatusInternalServerError, "Failed to logout from all devices")
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "Logged out from all devices successfully"})
}
