-- Migration 007: RBAC System and Admin Features
-- Creates tables for Role-Based Access Control, Zone Types, App Settings, and User Profiles

-- =======================
-- 1. ROLES TABLE (Use existing RentalCore roles table)
-- =======================
-- Note: roles table already exists in RentalCore with roleID as PK
-- We'll use the existing table and add WarehouseCore-specific roles if needed

-- Add WarehouseCore roles to existing table if they don't exist
-- Insert warehouse-specific roles if missing (convert JSON_ARRAY to JSONB and use ON CONFLICT)
INSERT INTO roles (name, display_name, description, permissions, is_system_role, is_active)
  SELECT name, display_name, description, permissions::jsonb, is_system_role, is_active
  FROM (
    VALUES
      ('warehouse_admin','Warehouse Admin','Full warehouse management access','["warehouse.*"]'::jsonb, true, true),
      ('warehouse_manager','Warehouse Manager','Warehouse operations management','["warehouse.manage","warehouse.reports"]'::jsonb, true, true),
      ('warehouse_worker','Warehouse Worker','Warehouse tasks and scanning','["warehouse.scan","warehouse.view"]'::jsonb, true, true),
      ('warehouse_viewer','Warehouse Viewer','Read-only warehouse access','["warehouse.view"]'::jsonb, true, true)
  ) AS v(name, display_name, description, permissions, is_system_role, is_active)
ON CONFLICT (name) DO NOTHING;

-- =======================
-- 2. USER_ROLES TABLE
-- =======================
-- Note: Using roleID from existing roles table
CREATE TABLE IF NOT EXISTS user_roles_wh (
  id SERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  role_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_user_role_wh ON user_roles_wh(user_id, role_id);
CREATE INDEX IF NOT EXISTS idx_user_id_wh ON user_roles_wh(user_id);
CREATE INDEX IF NOT EXISTS idx_role_id_wh ON user_roles_wh(role_id);

ALTER TABLE user_roles_wh ADD CONSTRAINT fk_user_roles_wh_user FOREIGN KEY (user_id) REFERENCES users(userID) ON DELETE CASCADE;
ALTER TABLE user_roles_wh ADD CONSTRAINT fk_user_roles_wh_role FOREIGN KEY (role_id) REFERENCES roles(roleID) ON DELETE CASCADE;

-- =======================
-- 3. ZONE_TYPES TABLE
-- =======================
CREATE TABLE IF NOT EXISTS zone_types (
  id SERIAL PRIMARY KEY,
  key VARCHAR(64) NOT NULL UNIQUE,
  label VARCHAR(128) NOT NULL,
  description TEXT NULL,
  default_led_pattern VARCHAR(50) DEFAULT 'breathe',
  default_led_color VARCHAR(9) DEFAULT '#FF4500',
  default_intensity SMALLINT DEFAULT 255,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_zone_type_key ON zone_types(key);

-- Optional CHECK for default_led_pattern values
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_zone_types_led_pattern') THEN
    ALTER TABLE zone_types ADD CONSTRAINT chk_zone_types_led_pattern CHECK (default_led_pattern IN ('solid','breathe','blink'));
  END IF;
END$$;

-- =======================
-- 4. APP_SETTINGS TABLE
-- =======================
CREATE TABLE IF NOT EXISTS app_settings (
  id SERIAL PRIMARY KEY,
  scope VARCHAR(50) NOT NULL DEFAULT 'warehousecore',
  k VARCHAR(128) NOT NULL,
  v JSONB NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_scope_key ON app_settings(scope, k);
CREATE INDEX IF NOT EXISTS idx_setting_key ON app_settings(k);

-- =======================
-- 5. USER_PROFILES TABLE
-- =======================
CREATE TABLE IF NOT EXISTS user_profiles (
  id SERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL UNIQUE,
  display_name VARCHAR(128) NULL,
  avatar_url VARCHAR(512) NULL,
  prefs JSONB NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_profile_user_id ON user_profiles(user_id);
ALTER TABLE user_profiles ADD CONSTRAINT fk_user_profiles_user FOREIGN KEY (user_id) REFERENCES users(userID) ON DELETE CASCADE;

-- =======================
-- SEED DATA
-- =======================

-- Insert default zone types
INSERT INTO zone_types (key, label, description, default_led_pattern, default_led_color, default_intensity)
VALUES
  ('shelf','Regal','Standard warehouse shelf','breathe','#FF4500',255),
  ('bin','Fach','Storage bin or compartment','breathe','#FF4500',255),
  ('eurobox','Eurobox','European standard box','breathe','#00FF00',220),
  ('gitterbox','Gitterbox','Wire mesh container','solid','#0088FF',200),
  ('flightcase','Flight Case','Transport flight case','solid','#FF00FF',200),
  ('rack','Rack','Equipment rack','breathe','#FFAA00',240),
  ('stage','Bühne','Stage area','blink','#FF0000',180),
  ('vehicle','Fahrzeug','Transport vehicle','solid','#00FFFF',160)
ON CONFLICT (key) DO NOTHING;

-- Insert default LED settings for single bin highlight
INSERT INTO app_settings (scope, k, v)
VALUES
  ('warehousecore','led.single_bin.default','{"color":"#FF4500","pattern":"breathe","intensity":255}'::jsonb),
  ('warehousecore','ui.theme','{"darkMode":true,"accentColor":"#EF4444"}'::jsonb)
ON CONFLICT (scope,k) DO NOTHING;
