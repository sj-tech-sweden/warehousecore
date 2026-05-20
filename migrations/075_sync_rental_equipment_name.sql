-- 075_sync_rental_equipment_name.sql
-- Backfill and sync rental_equipment.name from product_name so Eventory upserts don't violate NOT NULL
-- Idempotent and safe to re-run

BEGIN;

-- Ensure `name` column exists
ALTER TABLE rental_equipment ADD COLUMN IF NOT EXISTS name TEXT;

-- Backfill existing rows
UPDATE rental_equipment SET name = COALESCE(product_name, name) WHERE name IS NULL;

-- Make `name` column nullable (drop NOT NULL if present)
DO $$
BEGIN
  BEGIN
    ALTER TABLE rental_equipment ALTER COLUMN name DROP NOT NULL;
  EXCEPTION WHEN undefined_column THEN
    -- column missing, nothing to do
    NULL;
  END;
END$$;

-- Trigger function to populate name from product_name on insert/update
CREATE OR REPLACE FUNCTION trg_sync_rental_equipment_name()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.product_name IS NOT NULL AND (NEW.name IS NULL OR NEW.name = '') THEN
    NEW.name := NEW.product_name;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger if not exists
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_sync_rental_equipment_name') THEN
    CREATE TRIGGER trg_sync_rental_equipment_name
    BEFORE INSERT OR UPDATE ON rental_equipment
    FOR EACH ROW EXECUTE FUNCTION trg_sync_rental_equipment_name();
  END IF;
END$$;

COMMIT;
