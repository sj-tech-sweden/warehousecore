-- Migration 037: Add a varchar_pattern_ops index on devices(deviceID) for
-- efficient LIKE 'prefix%' queries under non-C database collations. This index
-- is used by AllocateDeviceCounter in internal/services/device_id.go to find
-- the next available device ID counter.
--
-- The existing plain btree index idx_devices_deviceid_pattern (created in
-- migration 030) is kept alongside this index. The plain index is needed for
-- equality lookups, ORDER BY, and collation-aware comparisons; the
-- varchar_pattern_ops index is additive and only optimises LIKE prefix scans.
--
-- IMPORTANT: CREATE INDEX CONCURRENTLY cannot run inside a transaction block.
-- Apply this file outside of BEGIN/COMMIT (e.g. psql -f 037_...sql), NOT via
-- a migration runner that wraps every file in a transaction. If your runner
-- always uses transactions, replace CONCURRENTLY with a plain CREATE INDEX
-- (which takes a stronger lock but runs inside a transaction).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_devices_deviceid_pattern_ops
    ON devices(deviceID varchar_pattern_ops);
