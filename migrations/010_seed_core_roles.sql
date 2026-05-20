-- Migration 010: Ensure core roles exist (admin, manager, worker, viewer)

-- Convert MySQL JSON_ARRAY and SELECT-UNION seed into Postgres-compatible INSERT ... SELECT with JSON arrays
INSERT INTO roles (name, display_name, description, permissions, is_system_role, is_active)
SELECT name, display_name, description, permissions::jsonb, is_system_role, is_active
FROM (
  VALUES
    ('admin','Admin','Full access', '["admin.*"]'::jsonb, true, true),
    ('manager','Manager','Manage operations', '["manage.*"]'::jsonb, true, true),
    ('worker','Worker','Operational tasks', '["warehouse.scan"]'::jsonb, true, true),
    ('viewer','Viewer','Read-only','["view.*"]'::jsonb, true, true)
) AS r(name, display_name, description, permissions, is_system_role, is_active)
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE roles.name = r.name);

