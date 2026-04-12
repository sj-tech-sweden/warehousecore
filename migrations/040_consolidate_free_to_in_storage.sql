-- Migration 040: Consolidate 'free' device status into 'in_storage'
-- The 'free' status was used when devices were first created or after repair completion.
-- The 'in_storage' status was used when devices were scanned back into the warehouse.
-- Both statuses mean the device is available in the warehouse, so we consolidate to 'in_storage'.

BEGIN;

UPDATE devices SET status = 'in_storage' WHERE LOWER(TRIM(status)) = 'free';

COMMIT;
