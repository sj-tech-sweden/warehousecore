-- Migration 011: Sync user_roles_wh assignments into shared user_roles table

CREATE TABLE IF NOT EXISTS user_roles (
  userID BIGINT NOT NULL,
  roleID INTEGER NOT NULL,
  assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  assigned_by BIGINT DEFAULT NULL,
  expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
  is_active BOOLEAN DEFAULT TRUE,
  UNIQUE (userID, roleID)
);

-- If a warehouse-specific `user_roles_wh` table exists, copy assignments safely.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = current_schema() AND table_name = 'user_roles_wh'
  ) THEN
    INSERT INTO user_roles (userID, roleID, assigned_at, assigned_by, is_active)
    SELECT user_id, role_id, NOW(), NULL, TRUE FROM user_roles_wh
    ON CONFLICT DO NOTHING;
  END IF;
END;
$$;

