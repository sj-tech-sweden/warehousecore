-- Migration 060: Add frequently-used device columns missing in some schema variants
-- Adds idempotent columns expected by server SQL (purchaseDate, lastmaintenance, nextmaintenance, current_location, retire_date, warranty_end_date)

BEGIN;

-- Add purchaseDate column (unquoted identifiers are lowercased by Postgres; this ensures the column purchasedate exists)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='devices' AND column_name='purchasedate') THEN
    EXECUTE 'ALTER TABLE devices ADD COLUMN purchasedate TIMESTAMP NULL';
  END IF;
END$$;

-- Add retire_date and warranty_end_date (snake_case variants used by multiple queries)
ALTER TABLE devices ADD COLUMN IF NOT EXISTS retire_date TIMESTAMP NULL;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS warranty_end_date TIMESTAMP NULL;

-- Add lastmaintenance / nextmaintenance (handler code uses these exact identifiers)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='devices' AND column_name='lastmaintenance') THEN
    EXECUTE 'ALTER TABLE devices ADD COLUMN lastmaintenance TIMESTAMP NULL';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='devices' AND column_name='nextmaintenance') THEN
    EXECUTE 'ALTER TABLE devices ADD COLUMN nextmaintenance TIMESTAMP NULL';
  END IF;
END$$;

-- Add current_location (commonly used)
ALTER TABLE devices ADD COLUMN IF NOT EXISTS current_location TEXT NULL;

COMMIT;
