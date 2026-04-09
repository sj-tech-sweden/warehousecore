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
-- The jobdevices table is shared with RentalCore; we use IF NOT EXISTS guards so
-- the migration is safe to re-run and does not break existing data.
BEGIN;

-- Lock the table for the duration of this migration to prevent concurrent
-- writes from inserting a new duplicate row between the DELETE and the
-- constraint addition. SHARE ROW EXCLUSIVE blocks INSERT, UPDATE, and DELETE
-- from other sessions while this transaction is open.
LOCK TABLE jobdevices IN SHARE ROW EXCLUSIVE MODE;

-- Step 1: Remove any duplicate (deviceID, jobID) pairs that would violate the
-- constraint, keeping the row with the newest pack_ts and using ctid only as a
-- deterministic tie-breaker when pack_ts values are equal or NULL.
DELETE FROM jobdevices
WHERE ctid IN (
  SELECT ctid
  FROM (
    SELECT ctid,
           ROW_NUMBER() OVER (
             PARTITION BY deviceID, jobID
             ORDER BY (pack_ts IS NULL), pack_ts DESC, ctid DESC
           ) AS rn
    FROM jobdevices
  ) ranked
  WHERE rn > 1
);

-- Step 2: Add the unique constraint (idempotent via IF NOT EXISTS guard on pg_constraint).
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM   pg_constraint
    WHERE  conname = 'uq_jobdevices_device_job'
      AND  conrelid = 'jobdevices'::regclass
  ) THEN
    ALTER TABLE jobdevices
      ADD CONSTRAINT uq_jobdevices_device_job UNIQUE (deviceID, jobID);
  END IF;
END;
$$;

COMMIT;
