# Database Migrations

This directory contains SQL migration scripts for the Snippy backend database.

## Migration Files

- `001_simplify_snippet_schema.sql` - Removes description/category fields, renames title to label
- `001_simplify_snippet_schema_rollback.sql` - Rollback script to restore old schema

## How to Apply Migrations

### Option 1: Using psql (Recommended for Production)

```bash
# Connect to your production database
psql -h your-db-host -U your-db-user -d snippy -f migrations/001_simplify_snippet_schema.sql
```

### Option 2: Using Docker (for Testing)

```bash
# Copy the migration file into the container
docker cp migrations/001_simplify_snippet_schema.sql snippy-postgres:/tmp/

# Execute the migration
docker exec -i snippy-postgres psql -U snippy -d snippy -f /tmp/001_simplify_snippet_schema.sql
```

### Option 3: Manual Execution

1. Connect to your database using your preferred client (pgAdmin, DBeaver, etc.)
2. Open `migrations/001_simplify_snippet_schema.sql`
3. Execute the script

## Verification

After running the migration, verify the schema:

```sql
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'snippets'
ORDER BY ordinal_position;
```

Expected columns:

- `id` (integer, NOT NULL)
- `label` (character varying, NOT NULL)
- `shortcut` (character varying, nullable)
- `content` (text, NOT NULL)
- `tags` (ARRAY, nullable)
- `user_id` (uuid, nullable)
- `created_at` (timestamp with time zone, nullable)
- `updated_at` (timestamp with time zone, nullable)

## Rollback

If you need to rollback the migration:

```bash
psql -h your-db-host -U your-db-user -d snippy -f migrations/001_simplify_snippet_schema_rollback.sql
```

**⚠️ WARNING**: Rolling back will result in NULL values for `description` and `category` fields since this data was deleted in the migration.

## Before Migration Checklist

- [ ] **Backup your database** before running any migration
- [ ] Test the migration on a staging/development database first
- [ ] Ensure the new backend code is deployed AFTER the migration
- [ ] Notify API consumers about the schema changes (breaking changes!)

## Breaking Changes

This migration introduces breaking API changes:

### Removed Fields:

- `description` (removed completely)
- `category` (removed completely)

### Renamed Fields:

- `title` → `label`

### API Impact:

- All API requests/responses using `title` must now use `label`
- Any filtering by `category` will no longer work
- Client applications must be updated accordingly

## Migration Order

⚠️ **IMPORTANT**: Follow this order to avoid downtime:

1. **Backup database**
2. **Deploy new backend code** (with label field support)
3. **Run migration** (removes old fields)
4. **Update client applications** to use new field names
