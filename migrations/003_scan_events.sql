-- Scan Events: Complete log of all barcode/QR scans
CREATE TABLE IF NOT EXISTS scan_events (
  scan_id BIGSERIAL PRIMARY KEY,
  scan_code VARCHAR(255) NOT NULL,
  scan_type VARCHAR(50) NOT NULL DEFAULT 'barcode',
  device_id VARCHAR(50) NULL,
  action VARCHAR(50) NULL,
  job_id BIGINT NULL,
  zone_id INT NULL,
  user_id BIGINT NULL,
  success BOOLEAN NOT NULL DEFAULT TRUE,
  error_message TEXT NULL,
  metadata JSON NULL,
  ip_address VARCHAR(45) NULL,
  user_agent TEXT NULL,
  timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes converted from inline MySQL definitions
CREATE INDEX IF NOT EXISTS idx_scan_code ON scan_events(scan_code);
CREATE INDEX IF NOT EXISTS idx_scan_device ON scan_events(device_id);
CREATE INDEX IF NOT EXISTS idx_scan_job ON scan_events(job_id);
CREATE INDEX IF NOT EXISTS idx_scan_timestamp ON scan_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_scan_success ON scan_events(success);

-- Foreign keys
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_scan_device' AND t.relname = 'scan_events'
  ) THEN
    EXECUTE 'ALTER TABLE scan_events ADD CONSTRAINT fk_scan_device FOREIGN KEY (device_id) REFERENCES devices(deviceID) ON DELETE CASCADE';
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
      WHERE c.conname = 'fk_scan_job' AND t.relname = 'scan_events'
    ) AND jobs_fk_column IS NOT NULL THEN
      EXECUTE format(
        'ALTER TABLE scan_events ADD CONSTRAINT fk_scan_job FOREIGN KEY (job_id) REFERENCES jobs(%I) ON DELETE SET NULL',
        jobs_fk_column
      );
    ELSIF jobs_fk_column IS NULL THEN
      RAISE NOTICE 'Skipping fk_scan_job: no compatible jobs id column found';
    END IF;
  END IF;
END$$;
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_scan_zone' AND t.relname = 'scan_events'
  ) THEN
    EXECUTE 'ALTER TABLE scan_events ADD CONSTRAINT fk_scan_zone FOREIGN KEY (zone_id) REFERENCES storage_zones(zone_id) ON DELETE SET NULL';
  END IF;
END$$;
