-- 074_update_eventory_index_and_columns.sql
-- Create a concrete unique index matching ON CONFLICT (product_name, supplier_name)
-- and ensure commonly-referenced columns exist (created_by)
-- Idempotent and safe to re-run

BEGIN;

-- Ensure created_by exists on rental_equipment (used by handlers)
ALTER TABLE rental_equipment
  ADD COLUMN IF NOT EXISTS created_by INT;

-- Drop any existing index with the same name (could be expression index)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname = 'ux_rental_equipment_product_supplier') THEN
    BEGIN
      EXECUTE 'DROP INDEX IF EXISTS ux_rental_equipment_product_supplier';
    EXCEPTION WHEN others THEN
      RAISE NOTICE 'Could not drop ux_rental_equipment_product_supplier: %', SQLERRM;
    END;
  END IF;
END$$;

-- Create concrete unique index on the two columns so ON CONFLICT works
CREATE UNIQUE INDEX IF NOT EXISTS ux_rental_equipment_product_supplier ON rental_equipment(product_name, supplier_name);

COMMIT;
