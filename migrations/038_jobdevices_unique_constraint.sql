-- Add a unique constraint on (deviceID, jobID) to the jobdevices table.
-- This is required so that the INSERT ... ON CONFLICT (deviceID, jobID) DO UPDATE
-- query used during outtake scanning works correctly in PostgreSQL.
--
-- The jobdevices table is shared with RentalCore; we use IF NOT EXISTS guards so
-- the migration is safe to re-run and does not break existing data.

-- Step 1: Remove any duplicate (deviceID, jobID) pairs that would violate the
-- constraint, keeping the row with the highest rowid / ctid (most recent insert).
DELETE FROM jobdevices a
USING jobdevices b
WHERE a.ctid < b.ctid
  AND a.deviceID = b.deviceID
  AND a.jobID    = b.jobID;

-- Step 2: Add the unique constraint (idempotent via DO NOTHING on duplicate name).
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
