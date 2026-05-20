-- Migration 056: Create sessions and company_settings tables
-- Idempotent: safe to run on existing databases

BEGIN;

-- Sessions table for server-side sessions used by auth
CREATE TABLE IF NOT EXISTS sessions (
    session_id   TEXT PRIMARY KEY,
    user_id      INTEGER NOT NULL,
    expires_at   TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    created_at   TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Company settings used for branding
CREATE TABLE IF NOT EXISTS company_settings (
    id           SERIAL PRIMARY KEY,
    company_name VARCHAR(255) NOT NULL,
    created_at   TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

COMMIT;
