-- Migration 036: Add label_path column to storage_zones table
ALTER TABLE storage_zones ADD COLUMN IF NOT EXISTS label_path VARCHAR(512) DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_storage_zones_label_path ON storage_zones(label_path);
