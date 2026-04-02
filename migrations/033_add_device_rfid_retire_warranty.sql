-- Migration: 033_add_device_rfid_retire_warranty.sql
-- Description: Add RFID, retire_date, and warranty_end_date columns to devices table.
-- Date: 2026-04-01

ALTER TABLE devices
  ADD COLUMN IF NOT EXISTS rfid VARCHAR(255) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS retire_date DATE DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS warranty_end_date DATE DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_devices_rfid ON devices(rfid);
CREATE INDEX IF NOT EXISTS idx_devices_serialnumber ON devices(serialnumber);
