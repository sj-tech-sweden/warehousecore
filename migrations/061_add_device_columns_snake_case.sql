-- Migration 061: Add snake_case device columns to support alternate schema variants
-- Adds `purchase_date`, `last_maintenance`, `next_maintenance` and aliases if missing.

BEGIN;

-- Add snake_case purchase_date if missing
ALTER TABLE devices ADD COLUMN IF NOT EXISTS purchase_date TIMESTAMP NULL;

-- Add snake_case retire_date and warranty_end_date (may already exist)
ALTER TABLE devices ADD COLUMN IF NOT EXISTS retire_date TIMESTAMP NULL;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS warranty_end_date TIMESTAMP NULL;

-- Add snake_case last_maintenance / next_maintenance if missing
ALTER TABLE devices ADD COLUMN IF NOT EXISTS last_maintenance TIMESTAMP NULL;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS next_maintenance TIMESTAMP NULL;

-- Also ensure lowercase unquoted variants exist (some code uses purchasedate/lastmaintenance)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='devices' AND column_name='purchasedate') THEN
    EXECUTE 'ALTER TABLE devices ADD COLUMN purchasedate TIMESTAMP NULL';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='devices' AND column_name='lastmaintenance') THEN
    EXECUTE 'ALTER TABLE devices ADD COLUMN lastmaintenance TIMESTAMP NULL';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='devices' AND column_name='nextmaintenance') THEN
    EXECUTE 'ALTER TABLE devices ADD COLUMN nextmaintenance TIMESTAMP NULL';
  END IF;
END$$;

COMMIT;
