-- Device Movements: Audit trail of all physical device movements
CREATE TABLE IF NOT EXISTS device_movements (
  movement_id BIGSERIAL PRIMARY KEY,
  device_id VARCHAR(50) NOT NULL,
  action VARCHAR(50) NOT NULL,
  from_zone_id INT NULL,
  to_zone_id INT NULL,
  from_job_id BIGINT NULL,
  to_job_id BIGINT NULL,
  barcode VARCHAR(255) NULL,
  user_id BIGINT NULL,
  notes TEXT NULL,
  metadata JSON NULL,
  timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_movement_device ON device_movements(device_id);
CREATE INDEX IF NOT EXISTS idx_movement_action ON device_movements(action);
CREATE INDEX IF NOT EXISTS idx_movement_timestamp ON device_movements(timestamp);
CREATE INDEX IF NOT EXISTS idx_movement_from_zone ON device_movements(from_zone_id);
CREATE INDEX IF NOT EXISTS idx_movement_to_zone ON device_movements(to_zone_id);
CREATE INDEX IF NOT EXISTS idx_movement_job ON device_movements(to_job_id);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_dm_device' AND t.relname = 'device_movements'
  ) THEN
    EXECUTE 'ALTER TABLE device_movements ADD CONSTRAINT fk_dm_device FOREIGN KEY (device_id) REFERENCES devices(deviceID) ON DELETE CASCADE';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_dm_from_zone' AND t.relname = 'device_movements'
  ) THEN
    EXECUTE 'ALTER TABLE device_movements ADD CONSTRAINT fk_dm_from_zone FOREIGN KEY (from_zone_id) REFERENCES storage_zones(zone_id) ON DELETE SET NULL';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_dm_to_zone' AND t.relname = 'device_movements'
  ) THEN
    EXECUTE 'ALTER TABLE device_movements ADD CONSTRAINT fk_dm_to_zone FOREIGN KEY (to_zone_id) REFERENCES storage_zones(zone_id) ON DELETE SET NULL';
  END IF;
END$$;
-- Conditionally add FK to jobs only if jobs table exists and constraint missing
DO $$
DECLARE
  jobs_fk_column text;
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = current_schema() AND table_name = 'jobs'
  ) THEN
    IF EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = current_schema() AND table_name = 'jobs' AND lower(column_name) = 'jobid'
    ) THEN
      jobs_fk_column := 'jobid';
    ELSIF EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = current_schema() AND table_name = 'jobs' AND lower(column_name) = 'id'
    ) THEN
      jobs_fk_column := 'id';
    END IF;

    IF NOT EXISTS (
      SELECT 1 FROM pg_constraint c
      JOIN pg_class t ON c.conrelid = t.oid
      WHERE c.conname = 'fk_dm_to_job' AND t.relname = 'device_movements'
    ) AND jobs_fk_column IS NOT NULL THEN
      EXECUTE format(
        'ALTER TABLE device_movements ADD CONSTRAINT fk_dm_to_job FOREIGN KEY (to_job_id) REFERENCES jobs(%I) ON DELETE SET NULL',
        jobs_fk_column
      );
    ELSIF jobs_fk_column IS NULL THEN
      RAISE NOTICE 'Skipping fk_dm_to_job: no compatible jobs id column found';
    END IF;
  END IF;
END$$;

-- Add zone reference to devices table if not exists
ALTER TABLE devices
  ADD COLUMN IF NOT EXISTS zone_id INT NULL;
CREATE INDEX IF NOT EXISTS idx_device_zone ON devices(zone_id);
