-- Rollback sessions table migration
DROP TRIGGER IF EXISTS trigger_update_session_last_activity ON sessions;
DROP FUNCTION IF EXISTS update_session_last_activity();
DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_created_at;
DROP INDEX IF EXISTS idx_sessions_active;
DROP INDEX IF EXISTS idx_sessions_user_id;
DROP TABLE IF EXISTS sessions;
