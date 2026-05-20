-- Migration 078: Ensure device_movements has movement_type and created_at
-- Idempotent: safe to run multiple times

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='device_movements' AND column_name='movement_type'
    ) THEN
        ALTER TABLE device_movements ADD COLUMN movement_type VARCHAR(50) NOT NULL DEFAULT 'transfer';
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='device_movements' AND column_name='created_at'
    ) THEN
        ALTER TABLE device_movements ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes WHERE tablename='device_movements' AND indexname='idx_movement_type'
    ) THEN
        CREATE INDEX idx_movement_type ON device_movements(movement_type);
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes WHERE tablename='device_movements' AND indexname='idx_movement_created'
    ) THEN
        CREATE INDEX idx_movement_created ON device_movements(created_at);
    END IF;
END$$;

COMMIT;
