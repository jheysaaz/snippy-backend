package main

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Snippet represents a code snippet
type Snippet struct {
	ID          int64     `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description,omitempty" db:"description"`
	Category    string    `json:"category" db:"category"`
	Shortcut    string    `json:"shortcut" db:"shortcut"` // Short string without spaces
	Content     string    `json:"content" db:"content"`
	Tags        []string  `json:"tags" db:"tags"`
	UserID      *string   `json:"userId,omitempty" db:"user_id"` // UUID as string
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// CreateSnippetRequest for creating a new snippet
type CreateSnippetRequest struct {
	Title       string   `json:"title" binding:"required,max=255"`
	Description string   `json:"description" binding:"max=5000"`
	Category    string   `json:"category" binding:"required,max=100"`
	Shortcut    string   `json:"shortcut" binding:"required,max=50"`    // Short string without spaces
	Content     string   `json:"content" binding:"required,max=100000"` // 100KB max
	Tags        []string `json:"tags" binding:"max=20,dive,max=50"`     // Max 20 tags, each max 50 chars
	// UserID is now extracted from JWT token, not from request body
}

// UpdateSnippetRequest for updating an existing snippet
type UpdateSnippetRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Shortcut    *string  `json:"shortcut,omitempty"`
	Content     *string  `json:"content,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	UserID      *string  `json:"userId,omitempty"` // UUID as string
}

// scanSnippet scans a database row into a Snippet struct
func scanSnippet(scanner interface {
	Scan(dest ...interface{}) error
}) (*Snippet, error) {
	var s Snippet
	var tags pq.StringArray
	var userID sql.NullString // UUID stored as string

	err := scanner.Scan(
		&s.ID,
		&s.Title,
		&s.Description,
		&s.Category,
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
	ID           string    `json:"id"` // UUID as string
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose password hash in JSON
	FullName     string    `json:"fullName"`
	AvatarURL    string    `json:"avatarUrl"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// CreateUserRequest for creating a new user
type CreateUserRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50,alphanum"`
	Email     string `json:"email" binding:"required,email,max=255"`
	Password  string `json:"password" binding:"required,min=8,max=128"` // Min 8 chars for security
	FullName  string `json:"fullName" binding:"max=255"`
	AvatarURL string `json:"avatarUrl" binding:"max=500,url"`
}

// LoginRequest for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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
func scanUser(scanner interface {
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
