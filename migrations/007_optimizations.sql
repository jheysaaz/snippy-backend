-- Migration 007: Database size and performance optimizations
-- These changes optimize storage, improve query performance, and reduce resource usage

-- ============================================================================
-- 1. PARTIAL INDEXES for is_deleted queries (replaces inefficient full indexes)
-- ============================================================================
-- Most queries filter WHERE is_deleted = false, partial indexes are more efficient

-- Drop existing inefficient boolean indexes
DROP INDEX IF EXISTS idx_users_is_deleted;
DROP INDEX IF EXISTS idx_snippets_is_deleted;

-- Create partial indexes for common "not deleted" queries
CREATE INDEX IF NOT EXISTS idx_users_active ON users(id) WHERE is_deleted = false;
CREATE INDEX IF NOT EXISTS idx_snippets_active ON snippets(user_id, created_at DESC) WHERE is_deleted = false;

-- ============================================================================
-- 2. COMPOSITE INDEX for sync endpoint optimization
-- ============================================================================
-- syncSnippets queries by user_id + is_deleted + created_at/updated_at
CREATE INDEX IF NOT EXISTS idx_snippets_sync ON snippets(user_id, updated_at DESC) WHERE is_deleted = false;
CREATE INDEX IF NOT EXISTS idx_snippets_deleted_sync ON snippets(user_id, deleted_at DESC) WHERE is_deleted = true AND deleted_at IS NOT NULL;

-- ============================================================================
-- 3. Remove low-selectivity index (change_type has only 4 values)
-- ============================================================================
DROP INDEX IF EXISTS idx_snippet_history_change_type;

-- ============================================================================
-- 4. PARTIAL INDEX for active sessions (most queries filter active = true)
-- ============================================================================
DROP INDEX IF EXISTS idx_sessions_active;
CREATE INDEX IF NOT EXISTS idx_sessions_active_user ON sessions(user_id, last_activity DESC) WHERE active = true;

-- ============================================================================
-- 5. COVERING INDEX for refresh token validation (avoid table lookup)
-- ============================================================================
DROP INDEX IF EXISTS idx_refresh_tokens_token;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_lookup ON refresh_tokens(token) 
    INCLUDE (id, session_id, expires_at, revoked, created_at) WHERE revoked = false;

-- ============================================================================
-- 6. Optimize snippet_history for range queries
-- ============================================================================
-- Composite index for efficient version retrieval
DROP INDEX IF EXISTS idx_snippet_history_snippet_id;
CREATE INDEX IF NOT EXISTS idx_snippet_history_version ON snippet_history(snippet_id, version_number DESC);

-- ============================================================================
-- 7. Add missing index for user lookup by login (username OR email)
-- ============================================================================
-- Login queries use: WHERE (username = $1 OR email = $1)
-- The existing unique constraints handle this, but adding functional index for OR query
CREATE INDEX IF NOT EXISTS idx_users_login ON users(username, email) WHERE is_deleted = false;

-- ============================================================================
-- 8. Cleanup: Remove redundant created_at DESC indexes (rarely queried alone)
-- ============================================================================
-- Keep idx_snippets_created_at as it's used for ORDER BY in getSnippets
-- But idx_users_created_at is rarely used alone, replaced by composite above

-- ============================================================================
-- VACUUM ANALYZE to update statistics after index changes
-- ============================================================================
-- Run manually after migration: VACUUM ANALYZE;
