-- Migration 062: Add `notes` column to `devices` (idempotent)

BEGIN;

ALTER TABLE devices ADD COLUMN IF NOT EXISTS notes TEXT NULL;

COMMIT;
