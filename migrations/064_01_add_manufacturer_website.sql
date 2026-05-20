-- Add website column to manufacturer if missing
-- Idempotent migration to satisfy seeds that insert website

BEGIN;

ALTER TABLE manufacturer ADD COLUMN IF NOT EXISTS website VARCHAR(255);

COMMIT;
