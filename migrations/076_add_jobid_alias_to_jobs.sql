-- 076_add_jobid_alias_to_jobs.sql
-- Add a `jobid` column that mirrors `id` to maintain compatibility
-- Idempotent: safe to run multiple times

BEGIN;

-- If the jobs table exists but doesn't have jobid, add a generated column that mirrors id
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'jobs') THEN
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_name='jobs' AND column_name='jobid'
    ) THEN
      ALTER TABLE jobs ADD COLUMN jobid INT GENERATED ALWAYS AS (id) STORED;
    END IF;
  END IF;
END$$;

COMMIT;
