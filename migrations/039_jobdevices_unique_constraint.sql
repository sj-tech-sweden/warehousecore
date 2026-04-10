-- Add a unique constraint on (deviceid, jobid) to the job_devices table.
-- This is required so that the INSERT ... ON CONFLICT (jobid, deviceid) DO UPDATE
-- query used by the jobdevices view's INSTEAD OF trigger works correctly in PostgreSQL.
--
-- Both steps run inside a single transaction with an explicit table lock so
-- that no concurrent INSERT/UPDATE can create a new duplicate row between the
-- DELETE and the constraint addition. The lock blocks writes briefly; on a
-- small table this is negligible. If the table is large and write availability
-- is critical, run during a maintenance window.
--
-- The underlying table is `job_devices` (RentalCore canonical table). This
-- migration operates on that table and is idempotent.
BEGIN;

-- Lock the table for the duration of this migration to prevent concurrent
-- writes from inserting a new duplicate row between the DELETE and the
-- constraint addition.
LOCK TABLE job_devices IN SHARE ROW EXCLUSIVE MODE;

-- Step 1: Remove any duplicate (jobid, deviceid) pairs that would violate the
-- constraint, keeping the row with the newest pack_ts and using ctid only as a
-- deterministic tie-breaker when pack_ts values are equal or NULL.
DELETE FROM job_devices
WHERE ctid IN (
  SELECT ctid
  FROM (
    SELECT ctid,
           ROW_NUMBER() OVER (
             PARTITION BY jobid, deviceid
             ORDER BY (pack_ts IS NULL), pack_ts DESC, ctid DESC
           ) AS rn
    FROM job_devices
  ) ranked
  WHERE rn > 1
);

-- Step 2: Add the unique constraint (idempotent: skip if any unique constraint
-- OR unique index already covers exactly (jobid, deviceid) on job_devices,
-- regardless of name — covers both ADD CONSTRAINT and CREATE UNIQUE INDEX paths).
DO $$
BEGIN
    IF NOT EXISTS (
    -- Check for a named UNIQUE constraint on exactly (jobid, deviceid)
    SELECT 1
    FROM   pg_constraint c
    WHERE  c.contype  = 'u'
      AND  c.conrelid = 'job_devices'::regclass
      AND  (
          SELECT array_agg(a.attname::text ORDER BY a.attname)
          FROM   pg_attribute a
          WHERE  a.attrelid = c.conrelid
            AND  a.attnum   = ANY(c.conkey)
        ) = ARRAY['jobid', 'deviceid']
    UNION ALL
    -- Check for a standalone UNIQUE index on exactly (jobid, deviceid)
    SELECT 1
    FROM   pg_index i
    WHERE  i.indrelid  = 'job_devices'::regclass
      AND  i.indisunique = true
      AND  (
        SELECT array_agg(a.attname::text ORDER BY a.attname)
        FROM   pg_attribute a
        WHERE  a.attrelid = i.indrelid
          AND  a.attnum   = ANY(i.indkey)
          AND  a.attnum   > 0
      ) = ARRAY['jobid', 'deviceid']
  ) THEN
    ALTER TABLE job_devices
      ADD CONSTRAINT uq_job_devices_jobid_deviceid UNIQUE (jobid, deviceid);
  END IF;
END;
$$;

COMMIT;
