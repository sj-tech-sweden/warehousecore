-- Migration 008: Auto-assign Admin Role to N. Thielmann
-- This migration automatically grants admin privileges to the user "N. Thielmann"

-- Find user "N. Thielmann" and assign admin role
-- Using CONCAT to build full name from first_name and last_name
-- Also trying username and email fields as fallback
-- Postgres-compatible: insert admin and warehouse_admin roles for matching user
INSERT INTO user_roles (userid, roleid, assigned_at, is_active)
SELECT u.userid, r.roleid, NOW(), TRUE
FROM users u
JOIN roles r ON r.name = 'admin'
WHERE (
  (COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) ILIKE '%thielmann%'
  OR u.username ILIKE '%thielmann%'
  OR u.email ILIKE '%thielmann%'
)
ON CONFLICT (userid, roleid) DO NOTHING;

INSERT INTO user_roles (userid, roleid, assigned_at, is_active)
SELECT u.userid, r.roleid, NOW(), TRUE
FROM users u
JOIN roles r ON r.name = 'warehouse_admin'
WHERE (
  (COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) ILIKE '%thielmann%'
  OR u.username ILIKE '%thielmann%'
  OR u.email ILIKE '%thielmann%'
)
ON CONFLICT (userid, roleid) DO NOTHING;

-- This migration is idempotent and safe to run across environments.
