-- Migration 012: Ensure super_admin role exists (shared roles table)

INSERT INTO roles (name, display_name, description, permissions, is_system_role, is_active)
SELECT * FROM (
  SELECT 'super_admin' AS name,
         'Super Admin' AS display_name,
         'Global superuser with full access' AS description,
         JSON_ARRAY('super_admin.*','admin.*') AS permissions,
         1 AS is_system_role,
         1 AS is_active
) AS r
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE roles.name = r.name);
