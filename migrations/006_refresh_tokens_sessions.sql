-- Refactor refresh_tokens to belong to sessions and decouple sessions from a single token

-- 1. Remove refresh_token_id from sessions (if exists)
ALTER TABLE sessions DROP COLUMN IF EXISTS refresh_token_id;

-- 2. Add session_id to refresh_tokens and drop user_id
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS session_id UUID REFERENCES sessions(id) ON DELETE CASCADE;

-- Optional: for legacy rows without a session_id, mark them revoked to prevent use
UPDATE refresh_tokens SET revoked = TRUE WHERE session_id IS NULL AND revoked = FALSE;

-- 3. Drop user_id and device_info columns if exist (data now derives user via session)
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS user_id;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS device_info;

-- 4. Add index on session_id
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_session_id ON refresh_tokens(session_id);
