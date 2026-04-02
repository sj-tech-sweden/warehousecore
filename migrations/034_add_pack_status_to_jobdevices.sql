-- Add pack_status and pack_ts columns to the shared jobdevices table.
-- These columns are used by WarehouseCore to track device scanning progress
-- for job outtake workflows, without modifying RentalCore-owned data.

ALTER TABLE jobdevices
  ADD COLUMN IF NOT EXISTS pack_status VARCHAR(50) NULL,
  ADD COLUMN IF NOT EXISTS pack_ts TIMESTAMP NULL;
