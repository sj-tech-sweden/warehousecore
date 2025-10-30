-- Rollback 009: Restore previous default LED values

-- Restore prior app_settings value (best-effort)
UPDATE app_settings
SET v = JSON_OBJECT('color', '#FF4500', 'pattern', 'breathe', 'intensity', 255), updated_at = NOW()
WHERE scope = 'warehousecore' AND k = 'led.single_bin.default';

-- Restore previous defaults on zone_types
ALTER TABLE zone_types
  MODIFY COLUMN default_led_pattern ENUM('solid','breathe','blink') DEFAULT 'breathe',
  MODIFY COLUMN default_led_color VARCHAR(9) DEFAULT '#FF4500',
  MODIFY COLUMN default_intensity TINYINT UNSIGNED DEFAULT 255;

