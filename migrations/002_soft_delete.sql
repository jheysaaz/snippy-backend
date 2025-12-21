-- Add soft delete columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Add soft delete columns to snippets table
ALTER TABLE snippets ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT false;
ALTER TABLE snippets ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Create index for soft delete queries
CREATE INDEX IF NOT EXISTS idx_users_is_deleted ON users(is_deleted);
CREATE INDEX IF NOT EXISTS idx_snippets_is_deleted ON snippets(is_deleted);
