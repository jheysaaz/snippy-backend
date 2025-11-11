package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

	if err := testDB.Ping(); err != nil {
		t.Skip("Skipping integration tests: Cannot connect to PostgreSQL")
	}

	// Clean up test data
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS users")

	// Initialize schema with NEW structure (category, shortcut, content)
	schema := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS snippets (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		category VARCHAR(100),
		shortcut VARCHAR(50),
		content TEXT NOT NULL,
		tags TEXT[],
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Create test user with matching ID from generateTestJWT()
	_, err = testDB.Exec(`
		INSERT INTO users (id, username, email, password_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, testUserID, "testuser", "test@example.com", "dummy-hash")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return testDB
}

// Clean up test database
func cleanupTestDB(t *testing.T, testDB *sql.DB) {
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS users")
	testDB.Close()
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
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
				"title": "Test Snippet",
				"description": "Test description",
				"category": "javascript",
				"shortcut": "js-test",
				"content": "console.log('test');",
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing title",
			payload: `{
				"category": "javascript",
				"content": "console.log('test');"
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing content",
			payload: `{
				"title": "Test Snippet",
				"category": "javascript"
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

			// Set global db variable for handler
			db = testDB

			router := gin.New()
			// Add auth middleware for create endpoint
			router.POST("/api/v1/snippets", AuthMiddleware(), createSnippet)

			req, _ := http.NewRequest("POST", "/api/v1/snippets", bytes.NewBufferString(tt.payload))
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

	// Set global db variable
	db = testDB

	// Insert test data with NEW schema (category, shortcut, content)
	// Use the same user_id as in generateTestJWT()
	_, err := testDB.Exec(`
		INSERT INTO snippets (title, description, category, shortcut, content, tags, user_id)
		VALUES 
			('Python Snippet', 'A Python example', 'python', 'py-hello', 'print("hello")', ARRAY['python', 'basics'], $1),
			('JavaScript Snippet', 'A JS example', 'javascript', 'js-hello', 'console.log("hello")', ARRAY['javascript'], $1)
	`, testUserID)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/snippets", getSnippets)

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
			name:           "Filter by category",
			queryParams:    "?category=python",
			expectedStatus: http.StatusOK,
			checkCount:     true,
			expectedCount:  1,
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
			req, _ := http.NewRequest("GET", "/api/v1/snippets"+tt.queryParams, nil)
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

	db = testDB

	// Insert test snippet with NEW schema using matching user_id
	var snippetID int64
	err := testDB.QueryRow(`
		INSERT INTO snippets (title, description, category, shortcut, content, tags, user_id)
		VALUES ('Test Snippet', 'Description', 'go', 'go-test', 'code here', ARRAY['test'], $1)
		RETURNING id
	`, testUserID).Scan(&snippetID)
	if err != nil {
		t.Fatalf("Failed to insert test snippet: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/snippets/:id", getSnippet)

	tests := []struct {
		name           string
		snippetID      string
		expectedStatus int
	}{
		{
			name:           "Get existing snippet",
			snippetID:      "1",
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
			req, _ := http.NewRequest("GET", "/api/v1/snippets/"+tt.snippetID, nil)
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

	db = testDB

	// Insert test snippet with NEW schema using matching user_id
	err := testDB.QueryRow(`
		INSERT INTO snippets (title, description, category, shortcut, content, tags, user_id)
		VALUES ('Original Title', 'Original Description', 'javascript', 'js-orig', 'original code', ARRAY['test'], $1)
		RETURNING id
	`, testUserID).Scan(new(int64))
	if err != nil {
		t.Fatalf("Failed to insert test snippet: %v", err)
	}

	router := gin.New()
	router.PUT("/api/v1/snippets/:id", AuthMiddleware(), updateSnippet)

	tests := []struct {
		name           string
		snippetID      string
		payload        string
		expectedStatus int
	}{
		{
			name:      "Update title only",
			snippetID: "1",
			payload: `{
				"title": "Updated Title"
			}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Update multiple fields",
			snippetID: "1",
			payload: `{
				"title": "New Title",
				"content": "new code"
			}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Update non-existent snippet",
			snippetID: "999",
			payload: `{
				"title": "New Title"
			}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid snippet ID",
			snippetID:      "invalid",
			payload:        `{"title": "New Title"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("PUT", "/api/v1/snippets/"+tt.snippetID, bytes.NewBufferString(tt.payload))
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

	db = testDB

	// Insert test snippet with NEW schema using matching user_id
	err := testDB.QueryRow(`
		INSERT INTO snippets (title, description, category, shortcut, content, tags, user_id)
		VALUES ('To Delete', 'Description', 'go', 'go-delete', 'code', ARRAY['test'], $1)
		RETURNING id
	`, testUserID).Scan(new(int64))
	if err != nil {
		t.Fatalf("Failed to insert test snippet: %v", err)
	}

	router := gin.New()
	router.DELETE("/api/v1/snippets/:id", AuthMiddleware(), deleteSnippet)

	tests := []struct {
		name           string
		snippetID      string
		expectedStatus int
	}{
		{
			name:           "Delete existing snippet",
			snippetID:      "1",
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
			req, _ := http.NewRequest("DELETE", "/api/v1/snippets/"+tt.snippetID, nil)
			req.Header.Set("Authorization", "Bearer "+generateTestJWT())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Set CORS env for test
	os.Setenv("CORS_ALLOWED_ORIGINS", "*")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")
	
	router := gin.New()
	router.Use(corsMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	// Test OPTIONS request
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
	}

	// Test CORS headers
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	headers := w.Header()
	allowedOrigin := headers.Get("Access-Control-Allow-Origin")
	if allowedOrigin != "*" {
		t.Errorf("CORS header Access-Control-Allow-Origin not set correctly, got: %s", allowedOrigin)
	}
}
