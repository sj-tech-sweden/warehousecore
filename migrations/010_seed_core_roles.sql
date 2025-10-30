-- Migration 010: Ensure core roles exist (admin, manager, worker, viewer)

INSERT INTO roles (name, display_name, description, permissions, is_system_role, is_active)
SELECT * FROM (
  SELECT 'admin'   AS name, 'Admin'   AS display_name, 'Full access',      JSON_ARRAY('admin.*')         AS permissions, 1 AS is_system_role, 1 AS is_active UNION ALL
  SELECT 'manager' AS name, 'Manager' AS display_name, 'Manage operations', JSON_ARRAY('manage.*')        AS permissions, 1, 1 UNION ALL
  SELECT 'worker'  AS name, 'Worker'  AS display_name, 'Operational tasks', JSON_ARRAY('warehouse.scan')  AS permissions, 1, 1 UNION ALL
  SELECT 'viewer'  AS name, 'Viewer'  AS display_name, 'Read-only',         JSON_ARRAY('view.*')          AS permissions, 1, 1
) AS r
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE roles.name = r.name);

