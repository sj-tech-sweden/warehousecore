-- Add label_path column to devices table
ALTER TABLE devices ADD COLUMN label_path VARCHAR(512) DEFAULT NULL;

-- Add index for faster lookups
CREATE INDEX idx_devices_label_path ON devices(label_path);
