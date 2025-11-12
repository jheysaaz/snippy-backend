-- Rollback Migration: Simplify snippet schema
-- Description: Restore description and category fields, rename label back to title
-- Date: 2025-11-12
-- WARNING: This rollback will result in NULL values for description and category

-- Step 1: Add back the old columns
ALTER TABLE snippets ADD COLUMN IF NOT EXISTS title VARCHAR(255);
ALTER TABLE snippets ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE snippets ADD COLUMN IF NOT EXISTS category VARCHAR(100);

-- Step 2: Copy data from 'label' back to 'title'
UPDATE snippets SET title = label WHERE title IS NULL;

-- Step 3: Make 'title' NOT NULL after data migration
ALTER TABLE snippets ALTER COLUMN title SET NOT NULL;

-- Step 4: Recreate the category index
CREATE INDEX IF NOT EXISTS idx_snippets_category ON snippets(category);

-- Step 5: Update the full-text search index to include both fields
DROP INDEX IF EXISTS idx_snippets_search;
CREATE INDEX idx_snippets_search ON snippets
USING GIN (
    to_tsvector('english',
        coalesce(title, '') || ' ' ||
        coalesce(description, '')
    )
);

-- Step 6: Drop the 'label' column
ALTER TABLE snippets DROP COLUMN IF EXISTS label;

-- Verification query (run this to check the rollback)
-- SELECT column_name, data_type, is_nullable
-- FROM information_schema.columns
-- WHERE table_name = 'snippets'
-- ORDER BY ordinal_position;
