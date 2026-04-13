-- Migration 040: Assign admin and warehouse_admin roles to N. Thielmann
--
-- This is a corrected follow-up to migration 008.  Migration 008 is kept
-- immutable so that environments that have already applied it are not affected
-- by edits to that file.
--
-- Improvements over 008:
--   1. A single PL/pgSQL block counts users matching an exact username first
--      and aborts with RAISE EXCEPTION if more than one matches, so roles are
--      never granted to the wrong account.
--   2. Raises a NOTICE when no user is found (safe no-op).
--   3. Returns early with a WARNING when neither required role exists.
--   4. Reports the actual number of inserted/updated rows (GET DIAGNOSTICS ROW_COUNT).
--   5. Idempotent: ON CONFLICT DO UPDATE ensures inactive role assignments are
--      re-activated and assigned_at is refreshed.
--
-- Safe to re-run on any environment where the target username is unique.

DO $$
DECLARE
  target_username TEXT   := 'ntielmann';
  target_userid   BIGINT;
  user_count      INT;
  role_count      INT;
BEGIN
  -- Match the intended account by exact (case-insensitive) username to avoid
  -- granting roles to an unrelated user whose profile merely contains a
  -- similar substring.
  SELECT COUNT(*) INTO user_count
  FROM users u
  WHERE LOWER(u.username) = LOWER(target_username);

  IF user_count = 0 THEN
    RAISE NOTICE 'Migration 040: no user found for username %; skipping role grants.', target_username;
    RETURN;
  END IF;

  IF user_count <> 1 THEN
    RAISE EXCEPTION 'Migration 040: expected exactly 1 user for username %, found %; '
                    'aborting to avoid granting roles to the wrong account.',
                    target_username, user_count;
  END IF;

  SELECT u.userid INTO target_userid
  FROM users u
  WHERE LOWER(u.username) = LOWER(target_username);

  -- Verify that both required roles exist before inserting.
  SELECT COUNT(*) INTO role_count
  FROM roles
  WHERE name IN ('admin', 'warehouse_admin');

  IF role_count = 0 THEN
    RAISE WARNING 'Migration 040: neither admin nor warehouse_admin role found; skipping grants.';
    RETURN;
  END IF;

  IF role_count < 2 THEN
    RAISE WARNING 'Migration 040: expected 2 roles (admin, warehouse_admin) but found %; '
                  'grant may be incomplete for userid %.', role_count, target_userid;
  END IF;

  -- Upsert both roles: re-activate any existing but inactive assignment and
  -- refresh assigned_at so the audit trail reflects this run.
  INSERT INTO user_roles (userid, roleid, assigned_at, is_active)
  SELECT target_userid, r.roleid, NOW(), TRUE
  FROM roles r
  WHERE r.name IN ('admin', 'warehouse_admin')
  ON CONFLICT (userid, roleid) DO UPDATE
    SET assigned_at = EXCLUDED.assigned_at,
        is_active   = EXCLUDED.is_active;

  GET DIAGNOSTICS role_count = ROW_COUNT;
  RAISE NOTICE 'Migration 040: % role(s) granted/confirmed for userid %.', role_count, target_userid;
END;
$$;
