-- Migration 036: Align app_settings column names with the Go model and add
-- a UNIQUE constraint on (scope, key) to enable ON CONFLICT (scope, key)
-- upserts in UpdateAPILimit and SetSetting.
--
-- The table was originally created (migration 007) with columns named k and v.
-- The Go AppSetting model uses gorm:"column:key" and gorm:"column:value", so
-- the columns must be renamed to match.  The rename steps are guarded by
-- existence checks so this migration is safe to run on a database that was
-- set up with key/value directly (e.g. a fresh install configured by hand).
BEGIN;

-- Step 1: Rename k -> key if the old column name still exists.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'app_settings' AND column_name = 'k'
    ) THEN
        ALTER TABLE app_settings RENAME COLUMN k TO key;
    END IF;
END;
$$;

-- Step 2: Rename v -> value if the old column name still exists.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'app_settings' AND column_name = 'v'
    ) THEN
        ALTER TABLE app_settings RENAME COLUMN v TO value;
    END IF;
END;
$$;

-- Step 3: Drop legacy unique constraint (was on (scope, k); now obsolete).
ALTER TABLE app_settings DROP CONSTRAINT IF EXISTS unique_scope_key;

-- Step 4: Drop legacy single-column index on k (renamed away above).
DROP INDEX IF EXISTS idx_setting_key;

-- Step 5: Add the unique constraint on (scope, key) that ON CONFLICT requires.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uq_app_settings_scope_key'
          AND conrelid = 'app_settings'::regclass
    ) THEN
        ALTER TABLE app_settings
            ADD CONSTRAINT uq_app_settings_scope_key UNIQUE (scope, key);
    END IF;
END;
$$;

-- Step 6: Recreate the single-column lookup index on the renamed column.
CREATE INDEX IF NOT EXISTS idx_app_setting_key ON app_settings (key);

COMMIT;
