-- 070_add_missing_users_jobs_rental.sql
-- Add commonly-missing columns and minimal jobs table required by handlers
-- Idempotent and safe to run multiple times

BEGIN;

-- Ensure users have is_admin and force_password_change flags
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS force_password_change BOOLEAN DEFAULT FALSE;

-- Ensure product_packages timestamps (again, idempotent)
ALTER TABLE product_packages
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Ensure rental_equipment has category column and timestamps
ALTER TABLE rental_equipment
  ADD COLUMN IF NOT EXISTS category VARCHAR(100),
  ADD COLUMN IF NOT EXISTS rental_price DECIMAL(10,2) DEFAULT 0.00,
  ADD COLUMN IF NOT EXISTS customer_price DECIMAL(10,2) DEFAULT 0.00,
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Minimal jobs table used by movement/job queries
CREATE TABLE IF NOT EXISTS jobs (
  id SERIAL PRIMARY KEY,
  job_code VARCHAR(128),
  status VARCHAR(64) DEFAULT 'open',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMIT;
