package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jheysaaz/snippy-backend/app/auth"
	"github.com/jheysaaz/snippy-backend/app/database"
	_ "github.com/lib/pq"
)

// Test user UUID - consistent across all tests
const testUserID = "123e4567-e89b-12d3-a456-426614174000"

// generateTestJWT creates a valid JWT token for testing
// The user_id should match the test user created in the database
func generateTestJWT() string {
	claims := jwt.MapClaims{
		"user_id":  testUserID,
		"username": "testuser",
		"email":    "test@example.com",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Use test secret or default
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "test-secret-key-for-ci-only"
	}

	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// getTestDBURL returns the database URL for testing
// Checks environment variable first, falls back to default
func getHandlersTestDBURL() string {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return dbURL
	}
	return "postgres://postgres:postgres@localhost:5432/snippy_test?sslmode=disable"
}

// Setup test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use test database URL or skip if not available
	dbURL := getHandlersTestDBURL()
	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Skipping integration tests: PostgreSQL not available")
	}

	if pingErr := testDB.Ping(); pingErr != nil {
		t.Skip("Skipping integration tests: Cannot connect to PostgreSQL")
	}

	// Clean up test data - drop in reverse dependency order
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippet_history")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS refresh_tokens")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS sessions")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS users")

	// Initialize schema with NEW structure (label, shortcut, content)
	schema := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name VARCHAR(255),
		avatar_url TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
		deleted_at TIMESTAMP WITH TIME ZONE
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		device_info TEXT,
		ip_address_hash TEXT,
		user_agent TEXT,
		active BOOLEAN DEFAULT true,
		last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP WITH TIME ZONE,
		logged_out_at TIMESTAMP WITH TIME ZONE
	);

	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
		token TEXT NOT NULL UNIQUE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		revoked BOOLEAN DEFAULT FALSE
	);

	CREATE TABLE IF NOT EXISTS snippets (
		id SERIAL PRIMARY KEY,
		label VARCHAR(255) NOT NULL,
		shortcut VARCHAR(50),
		content TEXT NOT NULL,
		tags TEXT[],
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
		deleted_at TIMESTAMP WITH TIME ZONE
	);

	CREATE TABLE IF NOT EXISTS snippet_history (
		id SERIAL PRIMARY KEY,
		snippet_id INTEGER NOT NULL REFERENCES snippets(id) ON DELETE CASCADE,
		version_number INTEGER NOT NULL,
		label VARCHAR(255) NOT NULL,
		shortcut VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		tags TEXT[],
		changed_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		change_type VARCHAR(20) NOT NULL CHECK (change_type IN ('create', 'edit', 'restore', 'soft_delete')),
		changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		change_notes TEXT,
		UNIQUE(snippet_id, version_number)
	);
	`
	if _, execErr := testDB.Exec(schema); execErr != nil {
		t.Fatalf("Failed to create test schema: %v", execErr)
	}

	// Delete existing test user if it exists (from previous test runs)
	_, _ = testDB.Exec(`DELETE FROM users WHERE username = 'testuser' OR email = 'test@example.com' OR id = $1`, testUserID)

	// Create test user with matching ID from generateTestJWT()
	_, err = testDB.Exec(`
		INSERT INTO users (id, username, email, password_hash, full_name, avatar_url)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, testUserID, "testuser", "test@example.com", "dummy-hash", "", "")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return testDB
} // Clean up test database
func cleanupTestDB(_ *testing.T, testDB *sql.DB) {
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippet_history")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS refresh_tokens")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS sessions")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS users")
	testDB.Close()
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestCreateSnippetValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        string
		expectedStatus int
	}{
		{
			name: "Valid snippet",
			payload: `{
				"label": "Test Snippet",
				"shortcut": "js-test",
				"content": "console.log('test');",
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing label",
			payload: `{
				"content": "console.log('test');"
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing content",
			payload: `{
				"label": "Test Snippet"
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			payload:        `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test database
			testDB := setupTestDB(t)
			defer cleanupTestDB(t, testDB)

			// Set database.DB for handlers
			database.DB = testDB

			router := gin.New()
			// Add auth middleware for create endpoint
			router.POST("/api/v1/snippets", auth.Middleware(), CreateSnippet)

			req, _ := http.NewRequestWithContext(context.Background(), "POST", "/api/v1/snippets", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestJWT())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGetSnippetsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Set database.DB for handlers
	database.DB = testDB

	// Insert test data with NEW schema (label, shortcut, content)
	// Use the same user_id as in generateTestJWT()
	_, err := testDB.Exec(`
		INSERT INTO snippets (label, shortcut, content, tags, user_id)
		VALUES
			('Python Snippet', 'py-hello', 'print("hello")', ARRAY['python', 'basics'], $1),
			('JavaScript Snippet', 'js-hello', 'console.log("hello")', ARRAY['javascript'], $1)
	`, testUserID)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/snippets", auth.Middleware(), GetCurrentUserSnippets)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkCount     bool
		expectedCount  int
	}{
		{
			name:           "Get all snippets",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkCount:     true,
			expectedCount:  2,
		},
		{
			name:           "Filter by tag",
			queryParams:    "?tag=basics",
			expectedStatus: http.StatusOK,
			checkCount:     true,
			expectedCount:  1,
		},
		{
			name:           "Search snippets",
			queryParams:    "?search=Python",
			expectedStatus: http.StatusOK,
			checkCount:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/v1/snippets"+tt.queryParams, nil)
			req.Header.Set("Authorization", "Bearer "+generateTestJWT())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkCount {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				count := int(response["count"].(float64))
				if count != tt.expectedCount {
					t.Errorf("Expected count %d, got %d", tt.expectedCount, count)
				}
			}
		})
	}
}

func TestGetSingleSnippet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	database.DB = testDB

	// Insert test snippet with NEW schema using matching user_id
	var snippetID int64
	err := testDB.QueryRow(`
		INSERT INTO snippets (label, shortcut, content, tags, user_id)
		VALUES ('Test Snippet', 'go-test', 'code here', ARRAY['test'], $1)
		RETURNING id
	`, testUserID).Scan(&snippetID)
	if err != nil {
		t.Fatalf("Failed to insert test snippet: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/snippets/:id", auth.Middleware(), GetSnippet)

	tests := []struct {
		name           string
		snippetID      string
		expectedStatus int
	}{
		{
			name:           "Get existing snippet",
			snippetID:      fmt.Sprintf("%d", snippetID),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existent snippet",
			snippetID:      "999",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid snippet ID",
			snippetID:      "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/v1/snippets/"+tt.snippetID, nil)
			req.Header.Set("Authorization", "Bearer "+generateTestJWT())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestUpdateSnippet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	database.DB = testDB

	// Insert test snippet with NEW schema using matching user_id
	var snippetID int64
	err := testDB.QueryRow(`
		INSERT INTO snippets (label, shortcut, content, tags, user_id)
		VALUES ('Original Label', 'js-orig', 'original code', ARRAY['test'], $1)
		RETURNING id
	`, testUserID).Scan(&snippetID)
	if err != nil {
		t.Fatalf("Failed to insert test snippet: %v", err)
	}

	snippetIDStr := fmt.Sprintf("%d", snippetID)

	router := gin.New()
	router.PUT("/api/v1/snippets/:id", auth.Middleware(), UpdateSnippet)

	tests := []struct {
		name           string
		snippetID      string
		payload        string
		expectedStatus int
	}{
		{
			name:      "Update label only",
			snippetID: snippetIDStr,
			payload: `{
				"label": "Updated Label"
			}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Update multiple fields",
			snippetID: snippetIDStr,
			payload: `{
				"label": "New Label",
				"content": "new code"
			}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Update non-existent snippet",
			snippetID: "999999",
			payload: `{
				"label": "New Label"
			}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid snippet ID",
			snippetID:      "invalid",
			payload:        `{"label": "New Label"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(context.Background(), "PUT", "/api/v1/snippets/"+tt.snippetID, bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestJWT())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestDeleteSnippet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	database.DB = testDB

	// Insert test snippet with NEW schema using matching user_id
	var snippetID int64
	err := testDB.QueryRow(`
		INSERT INTO snippets (label, shortcut, content, tags, user_id)
		VALUES ('To Delete', 'go-delete', 'code', ARRAY['test'], $1)
		RETURNING id
	`, testUserID).Scan(&snippetID)
	if err != nil {
		t.Fatalf("Failed to insert test snippet: %v", err)
	}

	router := gin.New()
	router.DELETE("/api/v1/snippets/:id", auth.Middleware(), DeleteSnippet)

	tests := []struct {
		name           string
		snippetID      string
		expectedStatus int
	}{
		{
			name:           "Delete existing snippet",
			snippetID:      fmt.Sprintf("%d", snippetID),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete non-existent snippet",
			snippetID:      "999",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid snippet ID",
			snippetID:      "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(context.Background(), "DELETE", "/api/v1/snippets/"+tt.snippetID, nil)
			req.Header.Set("Authorization", "Bearer "+generateTestJWT())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
