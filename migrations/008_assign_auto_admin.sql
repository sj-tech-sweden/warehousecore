-- Migration 008: Auto-assign Admin Role to N. Thielmann
-- This migration automatically grants admin privileges to the user "N. Thielmann"

-- Find user "N. Thielmann" and assign admin role
-- Using CONCAT to build full name from first_name and last_name
-- Also trying username and email fields as fallback

DO $$
DECLARE
  admin_role_id INT;
  wh_admin_role_id INT;
  thielmann_user_id INT;
BEGIN
  SELECT roleid INTO admin_role_id FROM roles WHERE name = 'admin' LIMIT 1;
  SELECT roleid INTO wh_admin_role_id FROM roles WHERE name = 'warehouse_admin' LIMIT 1;
  SELECT userid INTO thielmann_user_id FROM users
    WHERE (COALESCE(first_name,'') || ' ' || COALESCE(last_name,'')) ILIKE '%Thielmann%'
       OR username ILIKE '%thielmann%'
       OR email ILIKE '%thielmann%'
    LIMIT 1;

  IF thielmann_user_id IS NOT NULL AND admin_role_id IS NOT NULL THEN
    INSERT INTO user_roles_wh (user_id, role_id, created_at)
    VALUES (thielmann_user_id, admin_role_id, NOW())
    ON CONFLICT DO NOTHING;
  END IF;

  IF thielmann_user_id IS NOT NULL AND wh_admin_role_id IS NOT NULL THEN
    INSERT INTO user_roles_wh (user_id, role_id, created_at)
    VALUES (thielmann_user_id, wh_admin_role_id, NOW())
    ON CONFLICT DO NOTHING;
  END IF;

  IF thielmann_user_id IS NULL THEN
    RAISE NOTICE 'WARNING: User "N. Thielmann" not found. Admin role not assigned.';
  ELSIF admin_role_id IS NULL THEN
    RAISE NOTICE 'WARNING: Admin role not found in roles table.';
  ELSE
    RAISE NOTICE 'SUCCESS: Admin role assigned to user ID %', thielmann_user_id;
  END IF;
END$$;
