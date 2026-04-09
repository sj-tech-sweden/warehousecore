-- Migration 037: Replace the plain devices(deviceID) index with a
-- varchar_pattern_ops index for efficient LIKE 'prefix%' queries under
-- non-C database collations. This index is used by AllocateDeviceCounter in
-- internal/services/device_id.go to find the next available device ID counter.
--
-- The plain index idx_devices_deviceid_pattern (created in migration 030) is
-- dropped here to avoid maintaining two redundant indexes on the same column.
--
-- CREATE INDEX CONCURRENTLY is used so the index build does not take a
-- table-level lock that would block concurrent writes on a live system.
-- NOTE: CONCURRENTLY cannot run inside an explicit transaction block; apply
-- this migration outside of a BEGIN/COMMIT wrapper.
DROP INDEX IF EXISTS idx_devices_deviceid_pattern;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_devices_deviceid_pattern_ops
    ON devices(deviceID varchar_pattern_ops);
