-- Add a unique index on (product_name, supplier_name) in rental_equipment.
-- This enables INSERT ... ON CONFLICT DO UPDATE (atomic upsert) for the
-- Eventory sync, and also prevents accidental duplicate rows.
--
-- Both steps run inside a single transaction with an explicit table lock so
-- that no concurrent INSERT/UPDATE can create a new duplicate row between the
-- DELETE and the CREATE UNIQUE INDEX. The lock blocks writes briefly; on a
-- small table this is negligible. If the table is large and write availability
-- is critical, run during a maintenance window.
-- Only perform cleanup/index creation if the table exists in this DB. Use a
-- PL/pgSQL DO block so the check is safe and this file is a no-op when the
-- relation is absent.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'rental_equipment') THEN
    -- Lock, dedupe, and create index using EXECUTE so statements run even inside
    -- this anonymous block.
    EXECUTE 'LOCK TABLE rental_equipment IN SHARE ROW EXCLUSIVE MODE';

    EXECUTE 'DELETE FROM rental_equipment WHERE equipment_id IN (
      SELECT equipment_id FROM (
        SELECT equipment_id, ROW_NUMBER() OVER (
          PARTITION BY product_name, supplier_name ORDER BY equipment_id DESC
        ) AS rn FROM rental_equipment
      ) t WHERE rn > 1
    )';

    EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS uq_rental_equipment_name_supplier ON rental_equipment (product_name, supplier_name)';
  ELSE
    RAISE NOTICE 'Skipping migration 035: relation rental_equipment does not exist in this database.';
  END IF;
END$$;
