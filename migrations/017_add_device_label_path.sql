-- Add label_path column to devices table
ALTER TABLE devices ADD COLUMN IF NOT EXISTS label_path VARCHAR(512) DEFAULT NULL;

-- Add index for faster lookups
CREATE INDEX IF NOT EXISTS idx_devices_label_path ON devices(label_path);
