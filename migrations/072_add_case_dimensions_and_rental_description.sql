-- 072_add_case_dimensions_and_rental_description.sql
-- Add case dimension columns and ensure rental_equipment.description exists
-- Idempotent and safe to run multiple times

BEGIN;

-- Add dimensions and weight to cases
ALTER TABLE cases
  ADD COLUMN IF NOT EXISTS width NUMERIC DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS height NUMERIC DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS depth NUMERIC DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS weight NUMERIC DEFAULT NULL;

-- Ensure rental_equipment has a description column
ALTER TABLE rental_equipment
  ADD COLUMN IF NOT EXISTS description TEXT;

COMMIT;
