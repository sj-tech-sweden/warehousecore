-- Extend LED controller metadata with network and status fields

ALTER TABLE led_controllers
  ADD COLUMN ip_address VARCHAR(64) DEFAULT NULL AFTER last_seen,
  ADD COLUMN hostname VARCHAR(255) DEFAULT NULL AFTER ip_address,
  ADD COLUMN firmware_version VARCHAR(64) DEFAULT NULL AFTER hostname,
  ADD COLUMN mac_address VARCHAR(64) DEFAULT NULL AFTER firmware_version,
  ADD COLUMN status_data JSON DEFAULT NULL AFTER metadata,
  ADD INDEX idx_led_controllers_last_seen (last_seen);
