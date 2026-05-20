-- Migration 077: Add missing columns used by recent handlers
-- Idempotent: safe to run multiple times

BEGIN;

-- Add last_login to users if missing
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='users' AND column_name='last_login'
    ) THEN
        ALTER TABLE users ADD COLUMN last_login TIMESTAMP;
    END IF;
END$$;

-- Add moved_by to device_movements if missing
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='device_movements' AND column_name='moved_by'
    ) THEN
        ALTER TABLE device_movements ADD COLUMN moved_by INT NULL;
    END IF;
END$$;

-- Add foreign key constraint for moved_by -> users(userid) if missing
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_device_movements_moved_by'
    ) THEN
        ALTER TABLE device_movements ADD CONSTRAINT fk_device_movements_moved_by FOREIGN KEY (moved_by) REFERENCES users(userid) ON DELETE SET NULL;
    END IF;
END$$;

COMMIT;
