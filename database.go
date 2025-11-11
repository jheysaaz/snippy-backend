package main

import (
	"database/sql"
	"log"
)

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
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- Create index on created_at for sorting (performance optimization)
	CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

	-- Create index on username for fast lookups
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

	-- Create index on email for fast lookups
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	-- Create snippets table
	CREATE TABLE IF NOT EXISTS snippets (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		category VARCHAR(100) NOT NULL,
		shortcut VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		tags TEXT[], -- PostgreSQL array for tags
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- Create index on user_id for fast user snippet lookups
	CREATE INDEX IF NOT EXISTS idx_snippets_user_id ON snippets(user_id);

	-- Create index on created_at for sorting (performance optimization)
	CREATE INDEX IF NOT EXISTS idx_snippets_created_at ON snippets(created_at DESC);

	-- Create index on category for filtering
	CREATE INDEX IF NOT EXISTS idx_snippets_category ON snippets(category);

	-- Create index on shortcut for fast lookups
	CREATE INDEX IF NOT EXISTS idx_snippets_shortcut ON snippets(shortcut);

	-- Create GIN index on tags array for fast array searches
	CREATE INDEX IF NOT EXISTS idx_snippets_tags ON snippets USING GIN(tags);

	-- Create full-text search index on title and description
	CREATE INDEX IF NOT EXISTS idx_snippets_search ON snippets USING GIN(
		to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
	);

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
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	log.Println("Database schema initialized successfully")
	return nil
}

// Helper function to convert sql.NullString to string pointer
func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Helper function to convert string pointer to sql.NullString
func ptrToNullString(s *string) sql.NullString {
	if s != nil {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{Valid: false}
}
