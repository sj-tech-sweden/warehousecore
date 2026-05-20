-- Migration 009: Update LED defaults to warehouse standard (orange + breathe, intensity 180)

-- Update existing app setting if present; otherwise insert
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'app_settings') THEN
    CREATE TABLE app_settings (
      id SERIAL PRIMARY KEY,
      scope VARCHAR(50) NOT NULL DEFAULT 'warehousecore',
      k VARCHAR(128) NOT NULL,
      v JSONB NOT NULL,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    CREATE UNIQUE INDEX unique_scope_key ON app_settings(scope, k);
    CREATE INDEX idx_setting_key ON app_settings(k);
  END IF;
END$$;

INSERT INTO app_settings (scope, k, v)
VALUES ('warehousecore', 'led.single_bin.default', '{"color":"#FF7A00","pattern":"breathe","intensity":180}'::jsonb)
ON CONFLICT (scope,k) DO UPDATE SET v = EXCLUDED.v, updated_at = NOW();

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'zone_types') THEN
    CREATE TABLE zone_types (
      id SERIAL PRIMARY KEY,
      "key" VARCHAR(64) NOT NULL UNIQUE,
      label VARCHAR(128) NOT NULL,
      description TEXT NULL,
      default_led_pattern VARCHAR(50) DEFAULT 'breathe',
      default_led_color VARCHAR(9) DEFAULT '#FF4500',
      default_intensity SMALLINT DEFAULT 255,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    CREATE INDEX IF NOT EXISTS idx_zone_type_key ON zone_types("key");
  END IF;
END$$;

ALTER TABLE zone_types
  ALTER COLUMN default_led_pattern TYPE VARCHAR(50) USING default_led_pattern::text,
  ALTER COLUMN default_led_pattern SET DEFAULT 'breathe',
  ALTER COLUMN default_led_color TYPE VARCHAR(9) USING default_led_color::text,
  ALTER COLUMN default_led_color SET DEFAULT '#FF7A00',
  ALTER COLUMN default_intensity TYPE SMALLINT USING default_intensity::smallint,
  ALTER COLUMN default_intensity SET DEFAULT 180;
