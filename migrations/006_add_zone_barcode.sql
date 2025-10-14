-- Add barcode field to storage_zones for Fächer
-- Version 1.8 - 2025-10-14

ALTER TABLE storage_zones
ADD COLUMN barcode VARCHAR(255) NULL AFTER code,
ADD INDEX idx_zone_barcode (barcode);

-- Generate barcodes for existing zones
UPDATE storage_zones
SET barcode = CONCAT('ZONE-', LPAD(zone_id, 8, '0'))
WHERE barcode IS NULL AND type = 'shelf';
