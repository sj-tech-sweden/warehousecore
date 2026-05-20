-- Defect Reports: Detailed defect tracking beyond basic maintenance logs
CREATE TABLE IF NOT EXISTS defect_reports (
  defect_id BIGSERIAL PRIMARY KEY,
  device_id VARCHAR(50) NOT NULL,
  severity VARCHAR(50) NOT NULL DEFAULT 'medium',
  status VARCHAR(50) NOT NULL DEFAULT 'open',
  title VARCHAR(200) NOT NULL,
  description TEXT NOT NULL,
  reported_by BIGINT NULL,
  reported_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  assigned_to BIGINT NULL,
  repaired_by BIGINT NULL,
  repaired_at TIMESTAMP NULL,
  repair_cost DECIMAL(10,2) NULL,
  repair_notes TEXT NULL,
  closed_at TIMESTAMP NULL,
  images JSON NULL,
  metadata JSON NULL
);

CREATE INDEX IF NOT EXISTS idx_defect_device ON defect_reports(device_id);
CREATE INDEX IF NOT EXISTS idx_defect_status ON defect_reports(status);
CREATE INDEX IF NOT EXISTS idx_defect_severity ON defect_reports(severity);
CREATE INDEX IF NOT EXISTS idx_defect_reported ON defect_reports(reported_at);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_defect_device' AND t.relname = 'defect_reports'
  ) THEN
    EXECUTE 'ALTER TABLE defect_reports ADD CONSTRAINT fk_defect_device FOREIGN KEY (device_id) REFERENCES devices(deviceID) ON DELETE CASCADE';
  END IF;
END$$;

-- Inspection Schedules: Periodic inspection requirements
CREATE TABLE IF NOT EXISTS inspection_schedules (
  schedule_id BIGSERIAL PRIMARY KEY,
  device_id VARCHAR(50) NULL,
  product_id INT NULL,
  inspection_type VARCHAR(100) NOT NULL,
  interval_days INT NOT NULL,
  last_inspection TIMESTAMP NULL,
  next_inspection TIMESTAMP NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  notes TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inspection_device ON inspection_schedules(device_id);
CREATE INDEX IF NOT EXISTS idx_inspection_product ON inspection_schedules(product_id);
CREATE INDEX IF NOT EXISTS idx_inspection_next ON inspection_schedules(next_inspection);
CREATE INDEX IF NOT EXISTS idx_inspection_active ON inspection_schedules(is_active);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_inspection_device' AND t.relname = 'inspection_schedules'
  ) THEN
    EXECUTE 'ALTER TABLE inspection_schedules ADD CONSTRAINT fk_inspection_device FOREIGN KEY (device_id) REFERENCES devices(deviceID) ON DELETE CASCADE';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE c.conname = 'fk_inspection_product' AND t.relname = 'inspection_schedules'
  ) THEN
    EXECUTE 'ALTER TABLE inspection_schedules ADD CONSTRAINT fk_inspection_product FOREIGN KEY (product_id) REFERENCES products(productID) ON DELETE CASCADE';
  END IF;
END$$;
