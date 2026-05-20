CREATE TABLE IF NOT EXISTS "storage_zones" (
  zone_id SERIAL PRIMARY KEY,
  code VARCHAR(50) NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL,
  type TEXT NOT NULL DEFAULT 'other',
  description TEXT,
  parent_zone_id INT NULL,
  capacity INT NULL,
  location VARCHAR(255) NULL,
  metadata JSON NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes converted from inline MySQL syntax
CREATE INDEX IF NOT EXISTS idx_zone_type ON storage_zones(type);
CREATE INDEX IF NOT EXISTS idx_zone_active ON storage_zones(is_active);
CREATE INDEX IF NOT EXISTS idx_zone_parent ON storage_zones(parent_zone_id);

-- Add zone reference to existing cases table (if column doesn't exist)
ALTER TABLE cases
  ADD COLUMN IF NOT EXISTS zone_id INT NULL;

ALTER TABLE cases
  ADD COLUMN IF NOT EXISTS barcode VARCHAR(255) NULL;

ALTER TABLE cases
  ADD COLUMN IF NOT EXISTS rfid_tag VARCHAR(255) NULL;

CREATE INDEX IF NOT EXISTS idx_case_zone ON cases(zone_id);
