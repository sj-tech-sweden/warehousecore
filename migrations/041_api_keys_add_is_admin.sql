-- Add is_admin flag to api_keys so that admin endpoints can be accessed
-- via X-API-Key / Authorization: Bearer <key> headers.
BEGIN;

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;
