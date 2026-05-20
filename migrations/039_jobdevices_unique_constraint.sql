-- Add a unique constraint on (deviceID, jobID) to the jobdevices table.
-- This is required so that the INSERT ... ON CONFLICT (deviceID, jobID) DO UPDATE
-- query used during outtake scanning works correctly in PostgreSQL.
--
-- Both steps run inside a single transaction with an explicit table lock so
-- that no concurrent INSERT/UPDATE can create a new duplicate row between the
-- DELETE and the constraint addition. The lock blocks writes briefly; on a
-- small table this is negligible. If the table is large and write availability
-- is critical, run during a maintenance window.
--
-- The jobdevices table is shared with RentalCore; the constraint addition is
-- guarded by a column-based check (any existing UNIQUE constraint or UNIQUE
-- index on (deviceID, jobID), regardless of name) so the migration is safe
-- to re-run even if another system has already enforced uniqueness.
-- Run the whole migration inside a DO block so it's a no-op when `jobdevices`
-- doesn't exist, and so we can use PL/pgSQL control flow safely.
DO $$
DECLARE
  devcol text;
  jobcol text;
  delsql text;
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'jobdevices') THEN
    -- Lock the table for the duration of this operation
    EXECUTE 'LOCK TABLE jobdevices IN SHARE ROW EXCLUSIVE MODE';

    -- Find the actual column names for device and job (case-insensitive variants)
    SELECT a.attname INTO devcol
    FROM pg_attribute a
    WHERE a.attrelid = 'jobdevices'::regclass
      AND a.attnum > 0
      AND lower(a.attname) IN ('deviceid','device_id')
    LIMIT 1;

    SELECT a.attname INTO jobcol
    FROM pg_attribute a
    WHERE a.attrelid = 'jobdevices'::regclass
      AND a.attnum > 0
      AND lower(a.attname) IN ('jobid','job_id')
    LIMIT 1;

    IF devcol IS NULL OR jobcol IS NULL THEN
      RAISE NOTICE 'Skipping duplicate cleanup/constraint: jobdevices missing expected columns (device=% job=%)', devcol, jobcol;
      RETURN;
    END IF;

    -- Remove duplicate pairs using dynamic SQL that quotes the actual column names
    delsql := format(
      $q$DELETE FROM jobdevices WHERE ctid IN (
        SELECT ctid FROM (
          SELECT ctid, ROW_NUMBER() OVER (PARTITION BY %I, %I ORDER BY (pack_ts IS NULL), pack_ts DESC, ctid DESC) AS rn FROM jobdevices
        ) ranked WHERE rn > 1
      );$q$,
      devcol, jobcol
    );
    EXECUTE delsql;

    -- Add unique constraint if no existing unique constraint/index covers the same columns
    IF NOT EXISTS (
      SELECT 1 FROM pg_constraint c
      WHERE c.contype = 'u'
        AND c.conrelid = 'jobdevices'::regclass
        AND (
          SELECT array_agg(lower(a.attname) ORDER BY lower(a.attname))
          FROM pg_attribute a
          WHERE a.attrelid = c.conrelid
            AND a.attnum = ANY(c.conkey)
        ) = ARRAY[lower(devcol), lower(jobcol)]
      UNION ALL
      SELECT 1 FROM pg_index i
      WHERE i.indrelid = 'jobdevices'::regclass
        AND i.indisunique = true
        AND (
          SELECT array_agg(lower(a.attname) ORDER BY lower(a.attname))
          FROM pg_attribute a
          WHERE a.attrelid = i.indrelid
            AND a.attnum = ANY(i.indkey)
            AND a.attnum > 0
        ) = ARRAY[lower(devcol), lower(jobcol)]
    ) THEN
      EXECUTE format('ALTER TABLE jobdevices ADD CONSTRAINT uq_jobdevices_device_job UNIQUE (%I, %I);', devcol, jobcol);
    END IF;
  ELSE
    RAISE NOTICE 'Skipping migration 039: relation jobdevices does not exist in this database.';
  END IF;
END$$;
