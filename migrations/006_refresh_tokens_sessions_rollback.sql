-- Rollback changes to refresh_tokens and sessions

-- 1. Re-add refresh_token_id to sessions (nullable)
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS refresh_token_id UUID REFERENCES refresh_tokens(id) ON DELETE CASCADE;

-- 2. Re-add user_id and device_info to refresh_tokens
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS device_info TEXT;

-- 3. Drop session_id from refresh_tokens
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS session_id;

-- 4. Drop index on session_id
DROP INDEX IF EXISTS idx_refresh_tokens_session_id;
