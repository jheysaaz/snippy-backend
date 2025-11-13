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
func getUsers(c *gin.Context) {
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
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

// getUserByUsername retrieves a user by username
func getUserByUsername(c *gin.Context) {
	username := c.Param("username")

	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	row := database.DB.QueryRow(query, username)
	user, err := models.ScanUser(row)
	if handleScanError(c, err, "User not found") {
		return
	}

	respondSuccess(c, http.StatusOK, user)
}

// createUser creates a new user
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
		RETURNING id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
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
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key") {
			switch {
			case strings.Contains(err.Error(), "username"):
				respondError(c, http.StatusConflict, "Username already exists")
			case strings.Contains(err.Error(), "email"):
				respondError(c, http.StatusConflict, "Email already exists")
			default:
				respondError(c, http.StatusConflict, "User already exists")
			}
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	respondSuccess(c, http.StatusCreated, user)
}

// buildUserUpdateQuery constructs the dynamic UPDATE query for user updates
func buildUserUpdateQuery(req *models.UpdateUserRequest) ([]string, []interface{}, error) {
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
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			return nil, nil, err
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
	}

	return updates, args, nil
}

// updateUser updates an existing user
func updateUser(c *gin.Context) {
	id := c.Param("id")

	// Get authenticated user ID
	authUserID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if user is updating their own profile
	if !checkOwnership(c, id, authUserID, "profile") {
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic UPDATE query based on provided fields
	updates, args, err := buildUserUpdateQuery(&req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to process password")
		return
	}

	if len(updates) == 0 {
		respondError(c, http.StatusBadRequest, "No fields to update")
		return
	}

	// Add ID as last argument
	args = append(args, id)
	argPos := len(args)

	query := `
		UPDATE users
		SET ` + strings.Join(updates, ", ") + `
		WHERE id = $` + strconv.Itoa(argPos) + `
		RETURNING id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
	`

	row := database.DB.QueryRow(query, args...)
	user, err := models.ScanUser(row)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key") {
			switch {
			case strings.Contains(err.Error(), "username"):
				respondError(c, http.StatusConflict, "Username already exists")
			case strings.Contains(err.Error(), "email"):
				respondError(c, http.StatusConflict, "Email already exists")
			default:
				respondError(c, http.StatusConflict, "Duplicate value")
			}
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to update user")
		return
	}

	respondSuccess(c, http.StatusOK, user)
}

// deleteUser deletes a user
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

	query := `DELETE FROM users WHERE id = $1`

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
func getUserSnippets(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT id, label, shortcut, content, tags, user_id, created_at, updated_at
		FROM snippets
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := database.DB.Query(query, id)
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
		WHERE username = $1 OR email = $1
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
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
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
