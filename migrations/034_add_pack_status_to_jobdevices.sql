-- Use ALTER TABLE IF EXISTS so this migration is a no-op in DBs that
-- don't host the shared `jobdevices` relation (it lives in RentalCore).

ALTER TABLE IF EXISTS jobdevices
  ADD COLUMN IF NOT EXISTS pack_status VARCHAR(50) NULL,
  ADD COLUMN IF NOT EXISTS pack_ts TIMESTAMP NULL;
