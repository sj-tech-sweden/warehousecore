-- Migration: 042_add_cable_id_to_devices.sql
-- Description: Add cable_id column to devices table so devices can be associated with cables
-- Date: 2026-04-13

BEGIN;

-- Add cable_id column (nullable – a device may belong to a product, a cable, or both)
ALTER TABLE devices ADD COLUMN IF NOT EXISTS cable_id INTEGER;

-- Add foreign key constraint referencing cables(cableID)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_devices_cable_id'
          AND table_name = 'devices'
    ) THEN
        ALTER TABLE devices
            ADD CONSTRAINT fk_devices_cable_id
            FOREIGN KEY (cable_id) REFERENCES cables(cableID)
            ON DELETE SET NULL;
    END IF;
END $$;

-- Index for efficient lookup
CREATE INDEX IF NOT EXISTS idx_devices_cable_id ON devices(cable_id) WHERE cable_id IS NOT NULL;

COMMIT;
