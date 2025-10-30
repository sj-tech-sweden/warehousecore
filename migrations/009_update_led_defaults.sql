-- Migration 009: Update LED defaults to Tsunami standard (orange + breathe, intensity 180)

-- Update existing app setting if present; otherwise insert
INSERT INTO app_settings (scope, k, v)
VALUES ('warehousecore', 'led.single_bin.default', JSON_OBJECT('color', '#FF7A00', 'pattern', 'breathe', 'intensity', 180))
ON DUPLICATE KEY UPDATE v = VALUES(v), updated_at = NOW();

-- Align zone_types default values for new rows
ALTER TABLE zone_types
  MODIFY COLUMN default_led_pattern ENUM('solid','breathe','blink') DEFAULT 'breathe',
  MODIFY COLUMN default_led_color VARCHAR(9) DEFAULT '#FF7A00',
  MODIFY COLUMN default_intensity TINYINT UNSIGNED DEFAULT 180;

