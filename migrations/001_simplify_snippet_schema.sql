-- Migration: Simplify snippet schema
-- Description: Remove description and category fields, rename title to label
-- Date: 2025-11-12

-- Step 1: Add the new 'label' column
ALTER TABLE snippets ADD COLUMN IF NOT EXISTS label VARCHAR(255);

-- Step 2: Copy data from 'title' to 'label'
UPDATE snippets SET label = title WHERE label IS NULL;

-- Step 3: Make 'label' NOT NULL after data migration
ALTER TABLE snippets ALTER COLUMN label SET NOT NULL;

-- Step 4: Drop the old index on category
DROP INDEX IF EXISTS idx_snippets_category;

-- Step 5: Update the full-text search index
DROP INDEX IF EXISTS idx_snippets_search;
CREATE INDEX idx_snippets_search ON snippets
USING GIN (to_tsvector('english', coalesce(label, '')));

-- Step 6: Drop the old columns
ALTER TABLE snippets DROP COLUMN IF EXISTS title;
ALTER TABLE snippets DROP COLUMN IF EXISTS description;
ALTER TABLE snippets DROP COLUMN IF EXISTS category;

-- Verification query (run this to check the migration)
-- SELECT column_name, data_type, is_nullable
-- FROM information_schema.columns
-- WHERE table_name = 'snippets'
-- ORDER BY ordinal_position;
