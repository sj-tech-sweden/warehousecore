-- Add a unique index on (product_name, supplier_name) in rental_equipment.
-- This enables INSERT ... ON CONFLICT DO UPDATE (atomic upsert) for the
-- Eventory sync, and also prevents accidental duplicate rows.
CREATE UNIQUE INDEX IF NOT EXISTS uq_rental_equipment_name_supplier
    ON rental_equipment (product_name, supplier_name);
