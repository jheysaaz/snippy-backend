-- Rollback migration 003: Remove snippet version history

DROP TRIGGER IF EXISTS trigger_snippet_history_on_insert ON snippets;
DROP FUNCTION IF EXISTS create_snippet_history_on_insert();
DROP FUNCTION IF EXISTS get_next_snippet_version(INTEGER);
DROP TABLE IF EXISTS snippet_history CASCADE;
