package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

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

	// Initialize schema
	schema := `
	CREATE TABLE IF NOT EXISTS snippets (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		code TEXT NOT NULL,
		language VARCHAR(50),
		tags TEXT[],
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return testDB
}

// Clean up test database
func cleanupTestDB(t *testing.T, testDB *sql.DB) {
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
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
				"code": "console.log('test');",
				"language": "javascript",
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing title",
			payload: `{
				"code": "console.log('test');",
				"language": "javascript"
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing code",
			payload: `{
				"title": "Test Snippet",
				"language": "javascript"
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
			router.POST("/api/v1/snippets", createSnippet)

			req, _ := http.NewRequest("POST", "/api/v1/snippets", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
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

	// Insert test data
	_, err := testDB.Exec(`
		INSERT INTO snippets (title, description, code, language, tags)
		VALUES 
			('Python Snippet', 'A Python example', 'print("hello")', 'python', ARRAY['python', 'basics']),
			('JavaScript Snippet', 'A JS example', 'console.log("hello")', 'javascript', ARRAY['javascript'])
	`)
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
			name:           "Filter by language",
			queryParams:    "?language=python",
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

	// Insert test snippet
	var snippetID int64
	err := testDB.QueryRow(`
		INSERT INTO snippets (title, description, code, language, tags)
		VALUES ('Test Snippet', 'Description', 'code here', 'go', ARRAY['test'])
		RETURNING id
	`).Scan(&snippetID)
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

	// Insert test snippet
	err := testDB.QueryRow(`
		INSERT INTO snippets (title, description, code, language, tags)
		VALUES ('Original Title', 'Original Description', 'original code', 'javascript', ARRAY['test'])
		RETURNING id
	`).Scan(new(int64))
	if err != nil {
		t.Fatalf("Failed to insert test snippet: %v", err)
	}

	router := gin.New()
	router.PUT("/api/v1/snippets/:id", updateSnippet)

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
				"code": "new code"
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

	// Insert test snippet
	err := testDB.QueryRow(`
		INSERT INTO snippets (title, description, code, language, tags)
		VALUES ('To Delete', 'Description', 'code', 'go', ARRAY['test'])
		RETURNING id
	`).Scan(new(int64))
	if err != nil {
		t.Fatalf("Failed to insert test snippet: %v", err)
	}

	router := gin.New()
	router.DELETE("/api/v1/snippets/:id", deleteSnippet)

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
	if headers.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header Access-Control-Allow-Origin not set correctly")
	}
}
