package models

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Snippet represents a code snippet
type Snippet struct {
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"-" db:"deleted_at"`
	UserID    *string    `json:"userId,omitempty" db:"user_id"`
	Label     string     `json:"label" db:"label"`
	Shortcut  string     `json:"shortcut" db:"shortcut"`
	Content   string     `json:"content" db:"content"`
	Tags      []string   `json:"tags" db:"tags"`
	ID        int64      `json:"id" db:"id"`
	IsDeleted bool       `json:"-" db:"is_deleted"`
}

// CreateSnippetRequest for creating a new snippet
type CreateSnippetRequest struct {
	Label    string   `json:"label" binding:"required,max=255"`
	Shortcut string   `json:"shortcut" binding:"required,max=50"`    // Short string without spaces
	Content  string   `json:"content" binding:"required,max=100000"` // 100KB max
	Tags     []string `json:"tags" binding:"max=20,dive,max=50"`     // Max 20 tags, each max 50 chars
	// UserID is now extracted from JWT token, not from request body
}

// UpdateSnippetRequest for updating an existing snippet
type UpdateSnippetRequest struct {
	Label       *string  `json:"label,omitempty"`
	Shortcut    *string  `json:"shortcut,omitempty"`
	Content     *string  `json:"content,omitempty"`
	UserID      *string  `json:"userId,omitempty"`      // UUID as string
	ChangeNotes *string  `json:"changeNotes,omitempty"` // Optional description of the change
	Tags        []string `json:"tags,omitempty"`
}

// SnippetHistory represents a version in snippet history
type SnippetHistory struct {
	ChangedAt     time.Time `json:"changedAt"`
	ChangeNotes   *string   `json:"changeNotes,omitempty"`
	Label         string    `json:"label"`
	Shortcut      string    `json:"shortcut"`
	Content       string    `json:"content"`
	ChangedBy     string    `json:"changedBy"`
	ChangeType    string    `json:"changeType"`
	Tags          []string  `json:"tags"`
	ID            int64     `json:"id"`
	SnippetID     int64     `json:"snippetId"`
	VersionNumber int       `json:"versionNumber"`
}

// scanSnippet scans a database row into a Snippet struct
func ScanSnippet(scanner interface {
	Scan(dest ...interface{}) error
}) (*Snippet, error) {
	var s Snippet
	var tags pq.StringArray
	var userID sql.NullString // UUID stored as string

	err := scanner.Scan(
		&s.ID,
		&s.Label,
		&s.Shortcut,
		&s.Content,
		&tags,
		&userID,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	s.Tags = tags
	if userID.Valid {
		s.UserID = &userID.String
	}

	return &s, nil
}

// User represents a user in the system
type User struct {
	ID           string     `json:"id"` // UUID as string
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"-"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Never expose password hash in JSON
	FullName     string     `json:"fullName"`
	AvatarURL    string     `json:"avatarUrl"`
	IsDeleted    bool       `json:"-"`
}

// CreateUserRequest for creating a new user
type CreateUserRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50,alphanum"`
	Email     string `json:"email" binding:"required,email,max=255"`
	Password  string `json:"password" binding:"required,min=8,max=128"` // Min 8 chars for security
	FullName  string `json:"fullName" binding:"omitempty,max=255"`
	AvatarURL string `json:"avatarUrl" binding:"omitempty,max=500,url"`
}

// LoginRequest for user login
type LoginRequest struct {
	Login    string `json:"login" binding:"required"` // Can be username or email
	Password string `json:"password" binding:"required"`
}

// LoginResponse returned after successful login
type LoginResponse struct {
	User        *User  `json:"user"`
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"` // Access token expiration in seconds
	// RefreshToken is now sent as an HTTP-only cookie for security
}

// RefreshTokenRequest for refreshing access token
// RefreshToken is optional because it can come from cookie
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken,omitempty"`
}

// RefreshTokenResponse returned after refreshing token
type RefreshTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"` // Access token expiration in seconds
}

// RefreshToken database model
type RefreshToken struct {
	ID         string    `json:"id"`
	ExpiresAt  time.Time `json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
	UserID     string    `json:"userId"`
	Token      string    `json:"token"`
	DeviceInfo string    `json:"deviceInfo,omitempty"`
	Revoked    bool      `json:"revoked"`
}

// UpdateUserRequest for updating an existing user
type UpdateUserRequest struct {
	Username  *string `json:"username,omitempty"`
	Email     *string `json:"email,omitempty"`
	Password  *string `json:"password,omitempty"` // Will be hashed
	FullName  *string `json:"fullName,omitempty"`
	AvatarURL *string `json:"avatarUrl,omitempty"`
}

// scanUser scans a database row into a User struct
func ScanUser(scanner interface {
	Scan(dest ...interface{}) error
}) (*User, error) {
	var user User

	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ScanUserForAuth scans a database row into a User struct including password_hash (for authentication)
func ScanUserForAuth(scanner interface {
	Scan(dest ...interface{}) error
}) (*User, error) {
	var user User

	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Session represents a user session
type Session struct {
	LastActivity   time.Time  `json:"lastActivity"`
	CreatedAt      time.Time  `json:"createdAt"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	LoggedOutAt    *time.Time `json:"loggedOutAt,omitempty"`
	DeviceInfo     *string    `json:"deviceInfo,omitempty"`
	IPAddressHash  *string    `json:"-"` // Hash of IP, not exposed in API
	UserAgent      *string    `json:"userAgent,omitempty"`
	RefreshTokenID *string    `json:"refreshTokenId,omitempty"`
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	Active         bool       `json:"active"`
}
