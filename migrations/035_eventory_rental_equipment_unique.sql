-- Add a unique index on (product_name, supplier_name) in rental_equipment.
-- This enables INSERT ... ON CONFLICT DO UPDATE (atomic upsert) for the
-- Eventory sync, and also prevents accidental duplicate rows.
--
-- Both steps run inside a single transaction with an explicit table lock so
-- that no concurrent INSERT/UPDATE can create a new duplicate row between the
-- DELETE and the CREATE UNIQUE INDEX. The lock blocks writes briefly; on a
-- small table this is negligible. If the table is large and write availability
-- is critical, run during a maintenance window.
BEGIN;

-- Lock the table for the duration of this migration to prevent concurrent
-- writes from inserting a new duplicate row between the DELETE and the index
-- build.  SHARE ROW EXCLUSIVE blocks INSERT, UPDATE, and DELETE from other
-- sessions while this transaction is open.
LOCK TABLE rental_equipment IN SHARE ROW EXCLUSIVE MODE;

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
