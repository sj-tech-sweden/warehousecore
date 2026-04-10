-- Migration 040: Assign admin and warehouse_admin roles to N. Thielmann
--
-- This is a corrected follow-up to migration 008.  Migration 008 is kept
-- immutable so that environments that have already applied it are not affected
-- by edits to that file.
--
-- Improvements over 008:
--   1. A single PL/pgSQL block selects exactly one matching user (ORDER BY
--      userid LIMIT 1), so both role grants always apply to the same userid.
--      (008 used two independent queries that could resolve to different users.)
--   2. Raises a NOTICE if neither role is found, so the operator knows the
--      grant was a no-op.
--   3. Idempotent: ON CONFLICT (userid, roleid) DO NOTHING.
--
-- The ILIKE '%thielmann%' pattern is intentionally broad to match variations
-- (first/last name split, email domain, etc.).  If the target environment has
-- multiple users matching the pattern, ops should verify the granted userid
-- in the Postgres log (RAISE NOTICE output).
--
-- Safe to re-run on any environment.

DO $$
DECLARE
  target_userid BIGINT;
  role_count     INT;
BEGIN
  -- Pick exactly one user whose full name, username, or email contains
  -- 'thielmann' (case-insensitive).  ORDER BY userid gives deterministic
  -- selection when multiple rows match.
  SELECT u.userid INTO target_userid
  FROM users u
  WHERE (
    (COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) ILIKE '%thielmann%'
    OR u.username ILIKE '%thielmann%'
    OR u.email    ILIKE '%thielmann%'
  )
  ORDER BY u.userid
  LIMIT 1;

  IF target_userid IS NULL THEN
    RAISE NOTICE 'Migration 040: no user matching thielmann found; skipping role grants.';
    RETURN;
  END IF;

  -- Verify that both required roles exist before inserting.
  SELECT COUNT(*) INTO role_count
  FROM roles
  WHERE name IN ('admin', 'warehouse_admin');

  IF role_count < 2 THEN
    RAISE WARNING 'Migration 040: expected 2 roles (admin, warehouse_admin) but found %; '
                  'grant may be incomplete.', role_count;
  END IF;

  INSERT INTO user_roles (userid, roleid, assigned_at, is_active)
  SELECT target_userid, r.roleid, NOW(), TRUE
  FROM roles r
  WHERE r.name IN ('admin', 'warehouse_admin')
  ON CONFLICT (userid, roleid) DO NOTHING;

  RAISE NOTICE 'Migration 040: granted admin and warehouse_admin to userid %.', target_userid;
END;
$$;
