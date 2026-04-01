-- Add missing columns to defect_reports table to align with handler expectations.
-- The original schema lacked title, assigned_to, repair_cost, repair_notes,
-- repaired_at, closed_at, and reported_at columns.

-- Step 1: Add columns (reported_at added as nullable initially to allow backfill)
ALTER TABLE defect_reports
    ADD COLUMN IF NOT EXISTS title VARCHAR(200) NOT NULL DEFAULT 'Untitled Defect Report',
    ADD COLUMN IF NOT EXISTS assigned_to INT NULL REFERENCES users(userid) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS repair_cost DECIMAL(10,2) NULL,
    ADD COLUMN IF NOT EXISTS repair_notes TEXT NULL,
    ADD COLUMN IF NOT EXISTS repaired_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS closed_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS reported_at TIMESTAMP NULL;

-- Step 2: Backfill reported_at using the current timestamp for any existing rows
UPDATE defect_reports SET reported_at = CURRENT_TIMESTAMP WHERE reported_at IS NULL;

-- Step 3: Apply NOT NULL constraint and default for future rows
ALTER TABLE defect_reports
    ALTER COLUMN reported_at SET NOT NULL,
    ALTER COLUMN reported_at SET DEFAULT CURRENT_TIMESTAMP;
