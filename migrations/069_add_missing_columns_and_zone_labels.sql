-- 069_add_missing_columns_and_zone_labels.sql
-- Add missing columns expected by WarehouseCore handlers and normalize zone type labels to English
-- Idempotent changes only

BEGIN;

-- Ensure product_packages has created_at and updated_at
ALTER TABLE product_packages
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Ensure rental_equipment has rental_price, customer_price and timestamps
ALTER TABLE rental_equipment
  ADD COLUMN IF NOT EXISTS rental_price DECIMAL(10,2) DEFAULT 0.00,
  ADD COLUMN IF NOT EXISTS customer_price DECIMAL(10,2) DEFAULT 0.00,
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Update zone_types labels/descriptions to English equivalents (idempotent)
INSERT INTO zone_types ("key", label, description, default_led_pattern, default_led_color, default_intensity)
VALUES
  ('shelf','Shelf','Standard warehouse shelf','breathe','#FF4500',255),
  ('bin','Bin','Storage bin or compartment','breathe','#FF4500',255),
  ('eurobox','Eurobox','European standard box','breathe','#00FF00',220),
  ('gitterbox','Wire Mesh Container','Wire mesh container','solid','#0088FF',200),
  ('flightcase','Flight Case','Transport flight case','solid','#FF00FF',200),
  ('rack','Rack','Equipment rack','breathe','#FFAA00',240),
  ('stage','Stage','Stage area','blink','#FF0000',180),
  ('vehicle','Vehicle','Transport vehicle','solid','#00FFFF',160)
ON CONFLICT ("key") DO UPDATE SET
  label = EXCLUDED.label,
  description = EXCLUDED.description,
  default_led_pattern = EXCLUDED.default_led_pattern,
  default_led_color = EXCLUDED.default_led_color,
  default_intensity = EXCLUDED.default_intensity,
  updated_at = NOW();

COMMIT;
