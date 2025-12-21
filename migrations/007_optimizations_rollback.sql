-- Rollback Migration 007: Restore original indexes

-- Restore original boolean indexes
DROP INDEX IF EXISTS idx_users_active;
DROP INDEX IF EXISTS idx_snippets_active;
CREATE INDEX IF NOT EXISTS idx_users_is_deleted ON users(is_deleted);
CREATE INDEX IF NOT EXISTS idx_snippets_is_deleted ON snippets(is_deleted);

-- Remove optimization indexes
DROP INDEX IF EXISTS idx_snippets_sync;
DROP INDEX IF EXISTS idx_snippets_deleted_sync;
DROP INDEX IF EXISTS idx_sessions_active_user;
DROP INDEX IF EXISTS idx_refresh_tokens_token_lookup;
DROP INDEX IF EXISTS idx_snippet_history_version;
DROP INDEX IF EXISTS idx_users_login;

-- Restore original indexes
CREATE INDEX IF NOT EXISTS idx_snippet_history_change_type ON snippet_history(change_type);
CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(active);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX IF NOT EXISTS idx_snippet_history_snippet_id ON snippet_history(snippet_id);
