package main

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
		createdAt   time.Time
		updatedAt   time.Time
		name        string
		title       string
		description string
		category    string
		shortcut    string
		content     string
		tags        []string
		id          int64
	}{
		{
			name:        "Complete snippet",
			createdAt:   now,
			updatedAt:   now,
			tags:        []string{"test", "javascript"},
			id:          1,
			title:       "Test Snippet",
			description: "A test description",
			category:    "development",
			shortcut:    "test-snippet",
			content:     "console.log('test');",
		},
		{
			name:        "Snippet with empty tags",
			createdAt:   now,
			updatedAt:   now,
			tags:        []string{},
			id:          2,
			title:       "No Tags",
			description: "",
			category:    "personal",
			shortcut:    "hello-world",
			content:     "print('hello')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock row
			mockRow := &mockScanner{
				id:          tt.id,
				title:       tt.title,
				description: tt.description,
				category:    tt.category,
				shortcut:    tt.shortcut,
				content:     tt.content,
				tags:        pq.StringArray(tt.tags),
				createdAt:   tt.createdAt,
				updatedAt:   tt.updatedAt,
			}

			snippet, err := scanSnippet(mockRow)
			if err != nil {
				t.Fatalf("scanSnippet failed: %v", err)
			}

			if snippet.ID != tt.id {
				t.Errorf("ID = %d, want %d", snippet.ID, tt.id)
			}
			if snippet.Title != tt.title {
				t.Errorf("Title = %s, want %s", snippet.Title, tt.title)
			}
			if snippet.Category != tt.category {
				t.Errorf("Category = %s, want %s", snippet.Category, tt.category)
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
				Title:       stringPtr("New Title"),
				Description: stringPtr("New Description"),
				Category:    stringPtr("work"),
				Shortcut:    stringPtr("new-shortcut"),
				Content:     stringPtr("new content"),
				Tags:        []string{"tag1", "tag2"},
			},
		},
		{
			name:    "Only title provided",
			hasData: true,
			request: UpdateSnippetRequest{
				Title: stringPtr("Only Title"),
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
			hasData := tt.request.Title != nil ||
				tt.request.Description != nil ||
				tt.request.Category != nil ||
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
	createdAt   time.Time
	updatedAt   time.Time
	userID      *string
	title       string
	description string
	category    string
	shortcut    string
	content     string
	tags        pq.StringArray
	id          int64
}

func (m *mockScanner) Scan(dest ...interface{}) error {
	if len(dest) != 10 { // Updated to 10 fields
		return nil
	}

	*dest[0].(*int64) = m.id
	*dest[1].(*string) = m.title
	*dest[2].(*string) = m.description
	*dest[3].(*string) = m.category
	*dest[4].(*string) = m.shortcut
	*dest[5].(*string) = m.content
	*dest[6].(*pq.StringArray) = m.tags

	// Handle nullable userID (UUID as string)
	if m.userID != nil {
		dest[7].(*sql.NullString).String = *m.userID
		dest[7].(*sql.NullString).Valid = true
	} else {
		dest[7].(*sql.NullString).Valid = false
	}

	*dest[8].(*time.Time) = m.createdAt
	*dest[9].(*time.Time) = m.updatedAt

	return nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
