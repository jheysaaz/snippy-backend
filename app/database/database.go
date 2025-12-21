package database

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DB is the global database connection
var DB *sql.DB

// Init initializes the database connection and schema
func Init() error {
	// Get PostgreSQL connection string from environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/snippy?sslmode=disable"
	}

	// Connect to PostgreSQL
	var err error
	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		return err
	}

	// Configure connection pool for performance
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute) // Close idle connections after 5 minutes

	// Test database connection and wait for it to be ready
	for {
		if err := DB.Ping(); err != nil {
			log.Printf("Failed to ping PostgreSQL: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	// Initialize database schema
	return initDatabase()
}

// initDatabase creates tables and indexes if they don't exist
func initDatabase() error {
	schema := `
	-- Enable UUID extension for PostgreSQL
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	-- Create users table with UUID
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name VARCHAR(255),
		avatar_url TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		is_deleted BOOLEAN DEFAULT FALSE,
		deleted_at TIMESTAMP WITH TIME ZONE
	);

	-- Create index on created_at for sorting (performance optimization)
	CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

	-- Index on is_deleted for soft-delete lookups
	CREATE INDEX IF NOT EXISTS idx_users_is_deleted ON users(is_deleted);

	-- Create index on username for fast lookups
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

	-- Create index on email for fast lookups
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	-- Create refresh_tokens table for persistent authentication
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token TEXT NOT NULL UNIQUE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		revoked BOOLEAN DEFAULT FALSE,
		device_info TEXT  -- Optional: store device/browser info
	);

	-- Create indexes for refresh_tokens
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

	-- Create snippets table
	CREATE TABLE IF NOT EXISTS snippets (
		id SERIAL PRIMARY KEY,
		label VARCHAR(255) NOT NULL,
		shortcut VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		tags TEXT[], -- PostgreSQL array for tags
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		is_deleted BOOLEAN DEFAULT FALSE,
		deleted_at TIMESTAMP WITH TIME ZONE
	);

	-- Create index on user_id for fast user snippet lookups
	CREATE INDEX IF NOT EXISTS idx_snippets_user_id ON snippets(user_id);

	-- Create index on created_at for sorting (performance optimization)
	CREATE INDEX IF NOT EXISTS idx_snippets_created_at ON snippets(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_snippets_is_deleted ON snippets(is_deleted);

	-- Create index on shortcut for fast lookups
	CREATE INDEX IF NOT EXISTS idx_snippets_shortcut ON snippets(shortcut);

	-- Create GIN index on tags array for fast array searches
	CREATE INDEX IF NOT EXISTS idx_snippets_tags ON snippets USING GIN(tags);

	-- Create full-text search index on label
	CREATE INDEX IF NOT EXISTS idx_snippets_search ON snippets USING GIN(
		to_tsvector('english', coalesce(label, ''))
	);

	-- Create snippet_history table for version tracking
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

	-- Create indexes for snippet_history
	CREATE INDEX IF NOT EXISTS idx_snippet_history_snippet_id ON snippet_history(snippet_id);
	CREATE INDEX IF NOT EXISTS idx_snippet_history_changed_at ON snippet_history(changed_at DESC);
	CREATE INDEX IF NOT EXISTS idx_snippet_history_changed_by ON snippet_history(changed_by);
	CREATE INDEX IF NOT EXISTS idx_snippet_history_change_type ON snippet_history(change_type);

	-- Function to get next version number for a snippet
	CREATE OR REPLACE FUNCTION get_next_snippet_version(p_snippet_id INTEGER)
	RETURNS INTEGER AS $$
	DECLARE
		next_version INTEGER;
	BEGIN
		SELECT COALESCE(MAX(version_number), 0) + 1
		INTO next_version
		FROM snippet_history
		WHERE snippet_id = p_snippet_id;
		
		RETURN next_version;
	END;
	$$ LANGUAGE plpgsql;

	-- Trigger to automatically create history entry when snippet is created
	CREATE OR REPLACE FUNCTION trigger_snippet_history_on_insert()
	RETURNS TRIGGER AS $$
	BEGIN
		INSERT INTO snippet_history (
			snippet_id,
			version_number,
			label,
			shortcut,
			content,
			tags,
			changed_by,
			change_type,
			change_notes
		) VALUES (
			NEW.id,
			1,
			NEW.label,
			NEW.shortcut,
			NEW.content,
			NEW.tags,
			NEW.user_id,
			'create',
			'Initial version'
		);
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS trigger_snippet_history_on_insert ON snippets;
	CREATE TRIGGER trigger_snippet_history_on_insert
		AFTER INSERT ON snippets
		FOR EACH ROW
		EXECUTE FUNCTION trigger_snippet_history_on_insert();

	-- Create trigger to automatically update updated_at
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	DROP TRIGGER IF EXISTS update_snippets_updated_at ON snippets;
	CREATE TRIGGER update_snippets_updated_at
		BEFORE UPDATE ON snippets
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();

	DROP TRIGGER IF EXISTS update_users_updated_at ON users;
	CREATE TRIGGER update_users_updated_at
		BEFORE UPDATE ON users
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();

	-- Create sessions table for user session tracking
	CREATE TABLE IF NOT EXISTS sessions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		device_info TEXT,
		ip_address_hash TEXT,
		user_agent TEXT,
		refresh_token_id UUID REFERENCES refresh_tokens(id) ON DELETE CASCADE,
		active BOOLEAN DEFAULT true,
		last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP WITH TIME ZONE,
		logged_out_at TIMESTAMP WITH TIME ZONE
	);

	-- Create indexes for session lookups
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(active);
	CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

	-- Trigger to update last_activity when session is accessed
	CREATE OR REPLACE FUNCTION update_session_last_activity()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.last_activity = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS trigger_update_session_last_activity ON sessions;
	CREATE TRIGGER trigger_update_session_last_activity
		BEFORE UPDATE ON sessions
		FOR EACH ROW
		EXECUTE FUNCTION update_session_last_activity();
	`

	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	log.Println("Database schema initialized successfully")
	return nil
}
