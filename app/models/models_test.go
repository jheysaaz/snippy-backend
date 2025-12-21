package models

import (
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestSnippetScanFunction(t *testing.T) {
	// Test scanSnippet with mock data
	now := time.Now()

	tests := []struct {
		createdAt time.Time
		updatedAt time.Time
		name      string
		label     string
		shortcut  string
		content   string
		tags      []string
		id        int64
	}{
		{
			name:      "Complete snippet",
			createdAt: now,
			updatedAt: now,
			tags:      []string{"test", "javascript"},
			id:        1,
			label:     "Test Snippet",
			shortcut:  "test-snippet",
			content:   "console.log('test');",
		},
		{
			name:      "Snippet with empty tags",
			createdAt: now,
			updatedAt: now,
			tags:      []string{},
			id:        2,
			label:     "No Tags",
			shortcut:  "hello-world",
			content:   "print('hello')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock row
			mockRow := &mockScanner{
				id:        tt.id,
				label:     tt.label,
				shortcut:  tt.shortcut,
				content:   tt.content,
				tags:      pq.StringArray(tt.tags),
				createdAt: tt.createdAt,
				updatedAt: tt.updatedAt,
			}

			snippet, err := ScanSnippet(mockRow)
			if err != nil {
				t.Fatalf("scanSnippet failed: %v", err)
			}

			if snippet.ID != tt.id {
				t.Errorf("ID = %d, want %d", snippet.ID, tt.id)
			}
			if snippet.Label != tt.label {
				t.Errorf("Label = %s, want %s", snippet.Label, tt.label)
			}
			if snippet.Shortcut != tt.shortcut {
				t.Errorf("Shortcut = %s, want %s", snippet.Shortcut, tt.shortcut)
			}
			if snippet.Content != tt.content {
				t.Errorf("Content = %s, want %s", snippet.Content, tt.content)
			}
			if len(snippet.Tags) != len(tt.tags) {
				t.Errorf("Tags length = %d, want %d", len(snippet.Tags), len(tt.tags))
			}
		})
	}
}

func TestUpdateSnippetRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request UpdateSnippetRequest
		hasData bool
	}{
		{
			name:    "All fields provided",
			hasData: true,
			request: UpdateSnippetRequest{
				Label:    stringPtr("New Label"),
				Shortcut: stringPtr("new-shortcut"),
				Content:  stringPtr("new content"),
				Tags:     []string{"tag1", "tag2"},
			},
		},
		{
			name:    "Only label provided",
			hasData: true,
			request: UpdateSnippetRequest{
				Label: stringPtr("Only Label"),
			},
		},
		{
			name:    "No fields provided",
			hasData: false,
			request: UpdateSnippetRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasData := tt.request.Label != nil ||
				tt.request.Shortcut != nil ||
				tt.request.Content != nil ||
				tt.request.Tags != nil

			if hasData != tt.hasData {
				t.Errorf("Expected hasData=%v, got %v", tt.hasData, hasData)
			}
		})
	}
}

// Mock scanner for testing
type mockScanner struct {
	createdAt time.Time
	updatedAt time.Time
	userID    *string
	label     string
	shortcut  string
	content   string
	tags      pq.StringArray
	id        int64
}

func (m *mockScanner) Scan(dest ...interface{}) error {
	if len(dest) != 8 { // Updated to 8 fields
		return nil
	}

	*dest[0].(*int64) = m.id
	*dest[1].(*string) = m.label
	*dest[2].(*string) = m.shortcut
	*dest[3].(*string) = m.content
	*dest[4].(*pq.StringArray) = m.tags

	// Handle nullable userID (UUID as string)
	if m.userID != nil {
		dest[7].(*sql.NullString).String = *m.userID
		dest[7].(*sql.NullString).Valid = true
	} else {
		dest[5].(*sql.NullString).Valid = false
	}

	*dest[6].(*time.Time) = m.createdAt
	*dest[7].(*time.Time) = m.updatedAt

	return nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

// Test User scanning functions
func TestScanUser(t *testing.T) {
	now := time.Now()

	mockRow := &mockUserScanner{
		id:        "123e4567-e89b-12d3-a456-426614174000",
		username:  "testuser",
		email:     "test@example.com",
		fullName:  "Test User",
		avatarURL: "https://example.com/avatar.jpg",
		createdAt: now,
		updatedAt: now,
	}

	user, err := ScanUser(mockRow)
	if err != nil {
		t.Fatalf("ScanUser failed: %v", err)
	}

	if user.ID != mockRow.id {
		t.Errorf("ID = %s, want %s", user.ID, mockRow.id)
	}
	if user.Username != mockRow.username {
		t.Errorf("Username = %s, want %s", user.Username, mockRow.username)
	}
	if user.Email != mockRow.email {
		t.Errorf("Email = %s, want %s", user.Email, mockRow.email)
	}
}

func TestScanUserForAuth(t *testing.T) {
	now := time.Now()

	mockRow := &mockUserAuthScanner{
		id:           "123e4567-e89b-12d3-a456-426614174000",
		username:     "testuser",
		email:        "test@example.com",
		passwordHash: "$argon2id$v=19$m=65536,t=1,p=4$salt$hash",
		fullName:     "Test User",
		avatarURL:    "https://example.com/avatar.jpg",
		createdAt:    now,
		updatedAt:    now,
	}

	user, err := ScanUserForAuth(mockRow)
	if err != nil {
		t.Fatalf("ScanUserForAuth failed: %v", err)
	}

	if user.ID != mockRow.id {
		t.Errorf("ID = %s, want %s", user.ID, mockRow.id)
	}
	if user.PasswordHash != mockRow.passwordHash {
		t.Errorf("PasswordHash = %s, want %s", user.PasswordHash, mockRow.passwordHash)
	}
}

// Mock user scanner for ScanUser
type mockUserScanner struct {
	createdAt time.Time
	updatedAt time.Time
	username  string
	email     string
	fullName  string
	avatarURL string
	id        string
}

func (m *mockUserScanner) Scan(dest ...interface{}) error {
	if len(dest) != 7 {
		return sql.ErrNoRows
	}

	*dest[0].(*string) = m.id
	*dest[1].(*string) = m.username
	*dest[2].(*string) = m.email
	*dest[3].(*string) = m.fullName
	*dest[4].(*string) = m.avatarURL
	*dest[5].(*time.Time) = m.createdAt
	*dest[6].(*time.Time) = m.updatedAt

	return nil
}

// Mock user scanner for ScanUserForAuth
type mockUserAuthScanner struct {
	createdAt    time.Time
	updatedAt    time.Time
	username     string
	email        string
	passwordHash string
	fullName     string
	avatarURL    string
	id           string
}

func (m *mockUserAuthScanner) Scan(dest ...interface{}) error {
	if len(dest) != 8 {
		return sql.ErrNoRows
	}

	*dest[0].(*string) = m.id
	*dest[1].(*string) = m.username
	*dest[2].(*string) = m.email
	*dest[3].(*string) = m.passwordHash
	*dest[4].(*string) = m.fullName
	*dest[5].(*string) = m.avatarURL
	*dest[6].(*time.Time) = m.createdAt
	*dest[7].(*time.Time) = m.updatedAt

	return nil
}
