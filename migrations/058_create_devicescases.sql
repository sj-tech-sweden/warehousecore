-- Migration 058: Create devicescases linking table used by cases/devices handlers
BEGIN;

-- Create devicescases table if it doesn't exist. Some deployments store this
-- join table externally, so keep this migration idempotent and avoid hard
-- foreign key constraints to maintain compatibility.
CREATE TABLE IF NOT EXISTS devicescases (
  deviceID TEXT NOT NULL,
  caseID INTEGER NOT NULL,
  created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (deviceID, caseID)
);

-- Index by caseID for efficient lookups
CREATE INDEX IF NOT EXISTS idx_devicescases_caseid ON devicescases(caseID);

COMMIT;
