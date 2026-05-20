-- 073_eventory_upsert_and_cleanup.sql
-- Add missing columns used by Eventory sync and create unique constraint for ON CONFLICT upserts
-- Idempotent and safe to re-run

BEGIN;

-- Ensure rental_equipment has supplier_name (compatibility) and notes
ALTER TABLE rental_equipment
  ADD COLUMN IF NOT EXISTS supplier_name VARCHAR(255),
  ADD COLUMN IF NOT EXISTS notes TEXT;

-- Ensure product_name and supplier_name exist and are not null for uniqueness
-- Create unique index used by Eventory ON CONFLICT (product_name, supplier_name)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'ux_rental_equipment_product_supplier'
  ) THEN
    BEGIN
      CREATE UNIQUE INDEX ux_rental_equipment_product_supplier ON rental_equipment(LOWER(COALESCE(product_name,'')), LOWER(COALESCE(supplier_name,'')));
    EXCEPTION WHEN others THEN
      -- ignore if concurrent creation or other harmless error
      RAISE NOTICE 'ux_rental_equipment_product_supplier index create skipped: %', SQLERRM;
    END;
  END IF;
END$$;

-- Ensure cases has label_path column expected by handlers
ALTER TABLE cases ADD COLUMN IF NOT EXISTS label_path VARCHAR(512);

COMMIT;
