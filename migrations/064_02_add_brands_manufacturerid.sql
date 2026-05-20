-- Add manufacturerid column to brands table if missing
-- Idempotent migration to satisfy seeds referencing manufacturerid

BEGIN;

ALTER TABLE brands ADD COLUMN IF NOT EXISTS manufacturerid INT NULL;

-- Add FK constraint only if manufacturer table exists and constraint missing
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'manufacturer') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint WHERE conrelid = 'brands'::regclass AND conname = 'fk_brands_manufacturerid'
        ) THEN
            BEGIN
                EXECUTE 'ALTER TABLE brands ADD CONSTRAINT fk_brands_manufacturerid FOREIGN KEY (manufacturerid) REFERENCES manufacturer(manufacturerID) ON DELETE SET NULL';
            EXCEPTION WHEN duplicate_object THEN
                -- ignore
            END;
        END IF;
    END IF;
END$$;

COMMIT;
