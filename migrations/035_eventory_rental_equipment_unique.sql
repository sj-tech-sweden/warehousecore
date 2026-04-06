-- Add a unique index on (product_name, supplier_name) in rental_equipment.
-- This enables INSERT ... ON CONFLICT DO UPDATE (atomic upsert) for the
-- Eventory sync, and also prevents accidental duplicate rows.
--
-- Both steps run inside a single transaction so that no duplicate row can slip
-- in between the DELETE and the CREATE UNIQUE INDEX. The index build will hold
-- a brief write lock on rental_equipment; on a small table this is negligible.
-- If the table is very large and write availability is critical, run during a
-- maintenance window.
BEGIN;

-- Step 1: Remove duplicate rows, keeping the row with the highest equipment_id
-- (i.e. the most recently inserted).
DELETE FROM rental_equipment
WHERE equipment_id IN (
    SELECT equipment_id
    FROM (
        SELECT equipment_id,
               ROW_NUMBER() OVER (
                   PARTITION BY product_name, supplier_name
                   ORDER BY equipment_id DESC
               ) AS rn
        FROM rental_equipment
    ) t
    WHERE rn > 1
);

-- Step 2: Add the unique index. Running without CONCURRENTLY allows the index
-- creation to be wrapped in the same transaction as the duplicate-row removal,
-- eliminating the race window that would otherwise exist between the two steps.
CREATE UNIQUE INDEX IF NOT EXISTS uq_rental_equipment_name_supplier
    ON rental_equipment (product_name, supplier_name);

COMMIT;
