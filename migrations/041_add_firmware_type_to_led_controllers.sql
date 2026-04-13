-- Add firmware_type column to led_controllers to distinguish Arduino vs ESPHome controllers

ALTER TABLE led_controllers
  ADD COLUMN IF NOT EXISTS firmware_type VARCHAR(32) NOT NULL DEFAULT 'arduino';
