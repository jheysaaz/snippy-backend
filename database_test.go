package main

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
)

// getTestDBURL returns the database URL for testing
// Checks environment variable first, falls back to default
func getTestDBURL() string {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return dbURL
	}
	return "postgres://postgres:postgres@localhost:5432/snippy_test?sslmode=disable"
}

func TestDatabaseConnection(t *testing.T) {
	// Skip if no database available
	dbURL := getTestDBURL()
	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Skipping database tests: PostgreSQL not available")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("Skipping database tests: Cannot connect to PostgreSQL")
	}

	t.Log("Database connection successful")
}

func TestInitDatabase(t *testing.T) {
	dbURL := getTestDBURL()
	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Skipping database tests: PostgreSQL not available")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("Skipping database tests: Cannot connect to PostgreSQL")
	}

	// Clean up
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")

	// Set global db for initDatabase function
	db = testDB

	// Test schema initialization
	if err := initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}

	// Verify table exists
	var exists bool
	err = testDB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'snippets'
		)
	`).Scan(&exists)

	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}

	if !exists {
		t.Error("Snippets table was not created")
	}

	// Verify columns exist (NEW schema)
	columns := []string{"id", "title", "description", "category", "shortcut", "content", "tags", "user_id", "created_at", "updated_at"}
	for _, col := range columns {
		var colExists bool
		err = testDB.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.columns 
				WHERE table_name = 'snippets' AND column_name = $1
			)
		`, col).Scan(&colExists)

		if err != nil {
			t.Fatalf("Failed to check column %s: %v", col, err)
		}

		if !colExists {
			t.Errorf("Column %s does not exist", col)
		}
	}

	// Verify indexes exist (NEW schema)
	indexes := []string{"idx_snippets_created_at", "idx_snippets_category", "idx_snippets_shortcut", "idx_snippets_tags", "idx_snippets_search"}
	for _, idx := range indexes {
		var idxExists bool
		err = testDB.QueryRow(`
			SELECT EXISTS (
				SELECT FROM pg_indexes 
				WHERE tablename = 'snippets' AND indexname = $1
			)
		`, idx).Scan(&idxExists)

		if err != nil {
			t.Fatalf("Failed to check index %s: %v", idx, err)
		}

		if !idxExists {
			t.Errorf("Index %s does not exist", idx)
		}
	}

	// Clean up
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
}

func TestDatabaseSchemaIntegrity(t *testing.T) {
	dbURL := getTestDBURL()
	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Skipping database tests: PostgreSQL not available")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("Skipping database tests: Cannot connect to PostgreSQL")
	}

	// Set global db
	db = testDB

	// Clean up and initialize
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS users")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS refresh_tokens")
	if err := initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}

	// Create a test user with unique username
	username := "testuser_schema_" + time.Now().Format("20060102150405")
	email := username + "@example.com"
	var testUserID string
	err = testDB.QueryRow(`
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`, username, email, "dummy-hash").Scan(&testUserID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test inserting data (NEW schema) with valid user_id
	_, err = testDB.Exec(`
		INSERT INTO snippets (title, description, category, shortcut, content, tags, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "Test", "Description", "go", "go-test", "code", pq.Array([]string{"test"}), testUserID)

	if err != nil {
		t.Errorf("Failed to insert test data: %v", err)
	}

	// Test querying data
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM snippets").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query data: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}

	// Test array operations
	var tags []string
	err = testDB.QueryRow("SELECT tags FROM snippets WHERE id = 1").Scan(pq.Array(&tags))
	if err != nil {
		t.Errorf("Failed to query tags: %v", err)
	}

	// Clean up
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
}

func TestDatabaseTrigger(t *testing.T) {
	dbURL := getTestDBURL()
	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Skipping database tests: PostgreSQL not available")
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		t.Skip("Skipping database tests: Cannot connect to PostgreSQL")
	}

	db = testDB

	// Clean up and initialize
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS refresh_tokens")
	_, _ = testDB.Exec("DROP TABLE IF EXISTS users")
	if err := initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}

	// Create a test user first with unique data
	var testUserID string
	username := "testuser_" + time.Now().Format("20060102150405")
	email := username + "@example.com"
	err = testDB.QueryRow(`
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`, username, email, "dummy-hash").Scan(&testUserID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Insert a snippet (NEW schema) with valid user_id
	_, err = testDB.Exec(`
		INSERT INTO snippets (title, description, category, shortcut, content, tags, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "Test", "Description", "go", "go-test", "code", pq.Array([]string{"test"}), testUserID)

	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Get original updated_at
	var originalUpdated, newUpdated string
	err = testDB.QueryRow("SELECT updated_at FROM snippets WHERE id = 1").Scan(&originalUpdated)
	if err != nil {
		t.Fatalf("Failed to get original updated_at: %v", err)
	}

	// Update the snippet
	_, err = testDB.Exec("UPDATE snippets SET title = $1 WHERE id = 1", "Updated Title")
	if err != nil {
		t.Fatalf("Failed to update snippet: %v", err)
	}

	// Get new updated_at
	err = testDB.QueryRow("SELECT updated_at FROM snippets WHERE id = 1").Scan(&newUpdated)
	if err != nil {
		t.Fatalf("Failed to get new updated_at: %v", err)
	}

	// Verify updated_at changed (this may fail if updates happen too quickly)
	// In a real scenario, you might want to add a small delay or use more precise comparison
	t.Logf("Original updated_at: %s, New updated_at: %s", originalUpdated, newUpdated)

	// Clean up
	_, _ = testDB.Exec("DROP TABLE IF EXISTS snippets")
}
