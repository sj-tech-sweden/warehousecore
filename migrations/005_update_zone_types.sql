-- Update zone types to support only Lager, Regal, and Gitterbox
-- Version 1.7 - 2025-10-14

-- Convert MySQL ENUM ALTER to Postgres-compatible ALTER: change column to TEXT/VARCHAR and enforce via CHECK
ALTER TABLE storage_zones
	ALTER COLUMN type TYPE VARCHAR(50) USING type::text;
ALTER TABLE storage_zones
	ALTER COLUMN type SET NOT NULL;
ALTER TABLE storage_zones
	ALTER COLUMN type SET DEFAULT 'other';

-- Optional: add a CHECK constraint to limit allowed values
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'chk_storage_zones_type_allowed'
	) THEN
		ALTER TABLE storage_zones
			ADD CONSTRAINT chk_storage_zones_type_allowed CHECK (type IN ('warehouse','rack','gitterbox','shelf','vehicle','stage','case','other'));
	END IF;
END$$;

-- Note: 'warehouse' = Lager, 'rack' = Regal, 'gitterbox' = Gitterbox
-- Other types (shelf, vehicle, stage, case) kept for backward compatibility but should not be used
