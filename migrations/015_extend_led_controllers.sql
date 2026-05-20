-- Extend LED controller metadata with network and status fields

ALTER TABLE led_controllers
  ADD COLUMN IF NOT EXISTS ip_address VARCHAR(64) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS hostname VARCHAR(255) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS firmware_version VARCHAR(64) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS mac_address VARCHAR(64) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS status_data JSON DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_led_controllers_last_seen ON led_controllers(last_seen);
