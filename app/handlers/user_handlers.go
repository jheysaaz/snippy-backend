// Package handlers exposes HTTP handlers for the API.
package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"

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
		SELECT id, username, email, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE is_deleted = false
		ORDER BY created_at DESC
	`

	rows, err := database.DB.QueryContext(c.Request.Context(), query)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error closing user rows: %v", err)
		}
	}()

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
		SELECT id, username, email, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_deleted = false
	`

	row := database.DB.QueryRowContext(c.Request.Context(), query, id)
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
		RETURNING id, username, email, full_name, avatar_url, created_at, updated_at
	`

	row := database.DB.QueryRowContext(
		c.Request.Context(),
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
		WHERE id = $6 AND is_deleted = false
		RETURNING id, username, email, full_name, avatar_url, created_at, updated_at
	`

	row := database.DB.QueryRowContext(c.Request.Context(), query, usernameVal, emailVal, passwordHashVal, fullNameVal, avatarURLVal, id)
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

	// Revoke all refresh tokens for the user (logout from all devices)
	if err := models.RevokeAllUserTokens(c.Request.Context(), id); err != nil {
		log.Printf("Failed to revoke all tokens for user %s: %v", id, err)
		// Don't return error, continue with user deletion
	}

	query := `UPDATE users SET is_deleted = true, deleted_at = NOW() WHERE id = $1 AND is_deleted = false`

	result, err := database.DB.ExecContext(c.Request.Context(), query, id)
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
		SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at
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

	rows, err := database.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch user snippets")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error closing user snippets rows: %v", err)
		}
	}()

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

// checkAvailability verifies if a username or email is available for registration
// @Summary Check username/email availability
// @Description Quickly verify whether a username or email can be used
// @Tags auth
// @Produce json
// @Param username query string false "Username to check"
// @Param email query string false "Email to check"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /auth/availability [get]
func checkAvailability(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	email := strings.TrimSpace(c.Query("email"))

	if username == "" && email == "" {
		respondError(c, http.StatusBadRequest, "username or email required")
		return
	}

	conditions := make([]string, 0, 2)
	args := make([]interface{}, 0, 2)

	if username != "" {
		conditions = append(conditions, "username = $"+strconv.Itoa(len(args)+1))
		args = append(args, username)
	}

	if email != "" {
		conditions = append(conditions, "email = $"+strconv.Itoa(len(args)+1))
		args = append(args, email)
	}

	query := `
		SELECT username, email
		FROM users
		WHERE is_deleted = false AND (` + strings.Join(conditions, " OR ") + `)
		LIMIT 1
	`

	var existingUsername sql.NullString
	var existingEmail sql.NullString
	err := database.DB.QueryRowContext(c.Request.Context(), query, args...).Scan(&existingUsername, &existingEmail)
	if err != nil && err != sql.ErrNoRows {
		respondError(c, http.StatusInternalServerError, "Failed to check availability")
		return
	}

	var usernameAvailable *bool
	if username != "" {
		available := !existingUsername.Valid || existingUsername.String != username
		usernameAvailable = &available
	}

	var emailAvailable *bool
	if email != "" {
		available := !existingEmail.Valid || existingEmail.String != email
		emailAvailable = &available
	}

	response := struct {
		UsernameAvailable *bool `json:"usernameAvailable,omitempty"`
		EmailAvailable    *bool `json:"emailAvailable,omitempty"`
	}{
		UsernameAvailable: usernameAvailable,
		EmailAvailable:    emailAvailable,
	}

	respondSuccess(c, http.StatusOK, response)
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
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE (username = $1 OR email = $1) AND is_deleted = false
	`

	row := database.DB.QueryRowContext(c.Request.Context(), query, req.Login)
	user, err := models.ScanUserForAuth(row)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusUnauthorized, "Invalid username/email or password")
		return
	}
	if err != nil {
		log.Printf("login query failed for %q: %v", req.Login, err)
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
	if errStore := models.StoreRefreshToken(c.Request.Context(), user.ID, refreshToken, deviceInfo); errStore != nil {
		log.Printf("failed to store refresh token for %q: %v", user.ID, errStore)
		respondError(c, http.StatusInternalServerError, "Failed to store refresh token")
		return
	}

	// Get client IP address
	clientIP := c.ClientIP()

	// Create a session for this login
	_, err = models.CreateSession(c.Request.Context(), user.ID, deviceInfo, clientIP, c.GetHeader("User-Agent"), "")
	if err != nil {
		log.Printf("failed to create session for user %q: %v", user.ID, err)
		// Don't fail login if session creation fails, just log it
	}

	// Set refresh token as HTTP-only secure cookie
	c.SetCookie(
		"refresh_token", // name
		refreshToken,    // value
		int(models.RefreshTokenDuration.Seconds()), // maxAge in seconds
		"/",                             // path
		"",                              // domain (empty = current domain)
		c.Request.URL.Scheme == "https", // secure (true for HTTPS)
		true,                            // httpOnly
	)

	// Return user info and access token only (refresh token in cookie)
	response := models.LoginResponse{
		User:        user,
		AccessToken: accessToken,
		ExpiresIn:   int64(models.AccessTokenDuration.Seconds()),
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
	// Try to get refresh token from cookie first, then from request body
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		// Cookie not found, try to get from request body
		var req models.RefreshTokenRequest
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil || req.RefreshToken == "" {
			respondError(c, http.StatusBadRequest, "Refresh token required in cookie or body")
			return
		}
		refreshToken = req.RefreshToken
	}

	// Validate refresh token
	rt, err := models.ValidateRefreshToken(c.Request.Context(), refreshToken)
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
		SELECT id, username, email, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_deleted = false
	`

	row := database.DB.QueryRowContext(c.Request.Context(), query, rt.UserID)
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
	// Try to get refresh token from cookie first, then from request body
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		// Cookie not found, try to get from request body
		var req models.RefreshTokenRequest
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil || req.RefreshToken == "" {
			respondError(c, http.StatusBadRequest, "Refresh token required in cookie or body")
			return
		}
		refreshToken = req.RefreshToken
	}

	// Revoke the refresh token
	if err := models.RevokeRefreshToken(c.Request.Context(), refreshToken); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	// Clear the refresh token cookie
	c.SetCookie(
		"refresh_token",
		"",
		-1, // maxAge -1 deletes the cookie
		"/",
		"",
		c.Request.URL.Scheme == "https",
		true,
	)

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
	if err := models.RevokeAllUserTokens(c.Request.Context(), userID); err != nil {
		log.Printf("Failed to revoke all tokens for user %v: %v", userID, err)
		respondError(c, http.StatusInternalServerError, "Failed to logout from all devices")
		return
	}

	// Logout all sessions for this user
	if err := models.LogoutAllUserSessions(c.Request.Context(), userID); err != nil {
		log.Printf("Failed to logout all sessions for user %v: %v", userID, err)
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "Logged out from all devices successfully"})
}

// getSessions retrieves all active sessions for the authenticated user
// @Summary Get user sessions
// @Description Get all active sessions for the authenticated user
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /auth/sessions [get]
func getSessions(c *gin.Context) {
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	sessions, err := models.GetUserSessions(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch sessions")
		return
	}

	respondWithCount(c, sessions, len(sessions))
}

// logoutSession logs out a specific session
// @Summary Logout from a session
// @Description Logout from a specific session
// @Tags auth
// @Produce json
// @Param sessionId path string true "Session ID"
// @Success 200 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /auth/sessions/{sessionId} [post]
func logoutSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	userID, exists := getAuthUserID(c)
	if !exists {
		return
	}

	// Get the session to verify ownership
	session, err := models.GetSessionByID(c.Request.Context(), sessionID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Session not found")
		return
	}

	// Verify the session belongs to the authenticated user
	if session.UserID != userID {
		respondError(c, http.StatusForbidden, "Cannot logout someone else's session")
		return
	}

	// Logout the session
	if err := models.LogoutSession(c.Request.Context(), sessionID); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to logout session")
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "Session logged out successfully"})
}
