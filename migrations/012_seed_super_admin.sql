-- Migration 012: Ensure super_admin role exists (shared roles table)

INSERT INTO roles (name, display_name, description, permissions, is_system_role, is_active)
SELECT name, display_name, description, permissions::jsonb, is_system_role, is_active
FROM (
  VALUES
    ('super_admin','Super Admin','Global superuser with full access','["super_admin.*","admin.*"]'::jsonb, true, true)
) AS r(name, display_name, description, permissions, is_system_role, is_active)
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE roles.name = r.name);
