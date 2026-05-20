-- Add barcode field to storage_zones for Fächer
-- Version 1.8 - 2025-10-14

ALTER TABLE storage_zones
ADD COLUMN IF NOT EXISTS barcode VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_zone_barcode ON storage_zones(barcode);

-- Generate barcodes for existing zones
UPDATE storage_zones
SET barcode = 'ZONE-' || LPAD(zone_id::text, 8, '0')
WHERE barcode IS NULL AND type = 'shelf';
