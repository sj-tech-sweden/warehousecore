-- 071_add_missing_case_columns.sql
-- Add commonly-missing columns used by handlers (cases.status, cases.created_at, cases.updated_at)
-- Idempotent and safe to run multiple times

BEGIN;

ALTER TABLE cases
  ADD COLUMN IF NOT EXISTS status VARCHAR(64) DEFAULT 'open',
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Ensure indexes used by handlers
CREATE INDEX IF NOT EXISTS idx_cases_status ON cases(status);

COMMIT;
