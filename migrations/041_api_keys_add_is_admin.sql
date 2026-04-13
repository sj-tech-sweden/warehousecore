-- Add is_admin flag to api_keys so that admin endpoints can be accessed
-- via X-API-Key / Authorization: Bearer <key> headers.
--
-- NOTE: This release also switches API-key hashing from plain SHA-256 to
-- HMAC-SHA256 with an application-level pepper (API_KEY_PEPPER env var).
-- Existing api_key_hash values are incompatible with the new algorithm;
-- all API keys must be regenerated after applying this migration.
BEGIN;

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

-- Invalidate existing key hashes (new algorithm makes them unmatchable anyway)
-- without deleting API-key records, so audit/history such as names and usage
-- timestamps remain available. Operators should create fresh keys via the
-- admin UI after deploying. Plan replacement keys before running this
-- migration if API keys are actively in use.
-- Use a per-row invalidated value so the update remains valid if
-- api_key_hash is protected by a UNIQUE constraint.
UPDATE api_keys
SET api_key_hash = '__invalidated__:' || id::text,
    is_active = FALSE;

COMMIT;
