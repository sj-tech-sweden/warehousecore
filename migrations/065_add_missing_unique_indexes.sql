-- Migration 065: add missing UNIQUE indexes referenced by ON CONFLICT
-- Adds explicit unique indexes where ON CONFLICT(column) is used but no
-- matching unique constraint/index exists in earlier migrations. This keeps
-- INSERT ... ON CONFLICT(column) valid across DBs.

BEGIN;

-- roles(name)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE schemaname = current_schema() AND tablename = 'roles' AND indexname = 'ux_roles_name'
  ) THEN
    CREATE UNIQUE INDEX ux_roles_name ON roles (name);
  END IF;
END$$;

-- zone_types(key)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE schemaname = current_schema() AND tablename = 'zone_types' AND indexname = 'ux_zone_types_key'
  ) THEN
    CREATE UNIQUE INDEX ux_zone_types_key ON zone_types (key);
  END IF;
END$$;

-- product_field_definitions(name)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE schemaname = current_schema() AND tablename = 'product_field_definitions' AND indexname = 'ux_pfd_name'
  ) THEN
    CREATE UNIQUE INDEX ux_pfd_name ON product_field_definitions (name);
  END IF;
END$$;

COMMIT;
