-- Migration 003: Add snippet version history table
-- This allows tracking all changes to snippets and enables rollback functionality

CREATE TABLE IF NOT EXISTS snippet_history (
    id SERIAL PRIMARY KEY,
    snippet_id INTEGER NOT NULL REFERENCES snippets(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    
    -- Snapshot of snippet data at this version
    label VARCHAR(255) NOT NULL,
    shortcut VARCHAR(50),
    content TEXT NOT NULL,
    tags TEXT[],
    
    -- Change tracking
    changed_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    change_type VARCHAR(20) NOT NULL CHECK (change_type IN ('create', 'edit', 'restore', 'soft_delete')),
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    -- Optional change description
    change_notes TEXT,
    
    CONSTRAINT unique_snippet_version UNIQUE (snippet_id, version_number)
);

-- Indexes for efficient querying
CREATE INDEX idx_snippet_history_snippet_id ON snippet_history(snippet_id);
CREATE INDEX idx_snippet_history_changed_at ON snippet_history(changed_at);
CREATE INDEX idx_snippet_history_changed_by ON snippet_history(changed_by);
CREATE INDEX idx_snippet_history_change_type ON snippet_history(change_type);

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

-- Trigger function to automatically create history entry on snippet creation
CREATE OR REPLACE FUNCTION create_snippet_history_on_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO snippet_history (
        snippet_id, version_number, label, shortcut, content, tags,
        changed_by, change_type, changed_at
    ) VALUES (
        NEW.id, 1, NEW.label, NEW.shortcut, NEW.content, NEW.tags,
        NEW.user_id, 'create', NEW.created_at
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to snippets table
CREATE TRIGGER trigger_snippet_history_on_insert
AFTER INSERT ON snippets
FOR EACH ROW
EXECUTE FUNCTION create_snippet_history_on_insert();

COMMENT ON TABLE snippet_history IS 'Version history for snippets, enabling rollback and audit trail';
COMMENT ON COLUMN snippet_history.version_number IS 'Monotonically increasing version number per snippet';
COMMENT ON COLUMN snippet_history.change_type IS 'Type of change: create, edit, restore, soft_delete';
