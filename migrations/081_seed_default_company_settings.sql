-- Migration 081: Ensure a default company_settings row exists
-- Idempotent: inserts only when company_settings is empty

BEGIN;

INSERT INTO company_settings (company_name)
SELECT 'WarehouseCore'
WHERE NOT EXISTS (
    SELECT 1
    FROM company_settings
);

COMMIT;
