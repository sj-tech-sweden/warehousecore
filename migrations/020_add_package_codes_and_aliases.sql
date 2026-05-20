-- Migration 020: Add package codes and OCR alias mapping for product packages

-- 1) Add package_code column (temporary nullable)
ALTER TABLE product_packages
    ADD COLUMN IF NOT EXISTS package_code VARCHAR(32);

-- Backfill deterministically using the actual primary key column name (package_id or id)
DO $$
DECLARE
    ref_col text := NULL;
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'package_id') THEN
        ref_col := 'package_id';
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'id') THEN
        ref_col := 'id';
    END IF;

    IF ref_col IS NOT NULL THEN
        EXECUTE format('UPDATE product_packages SET package_code = ''PKG-'' || LPAD((%I)::text, 6, ''0'') WHERE package_code IS NULL OR package_code = '''';', ref_col);
    END IF;
END$$;

-- 3) Enforce NOT NULL + uniqueness
ALTER TABLE product_packages
    ALTER COLUMN package_code TYPE VARCHAR(32);
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_packages')
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'package_code')
       AND NOT EXISTS (SELECT 1 FROM product_packages WHERE package_code IS NULL OR package_code = '') THEN
        EXECUTE 'ALTER TABLE product_packages ALTER COLUMN package_code SET NOT NULL';
    ELSIF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_packages')
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'package_code') THEN
        RAISE WARNING 'Migration 020: package_code contains NULL/empty values; NOT NULL constraint was not applied';
    END IF;
END$$;
CREATE UNIQUE INDEX IF NOT EXISTS uq_product_package_code ON product_packages(package_code);

-- 4) Create aliases table for OCR mapping
CREATE TABLE IF NOT EXISTS product_package_aliases (
    alias_id SERIAL PRIMARY KEY,
    package_id INT NOT NULL,
    alias VARCHAR(191) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_package_alias ON product_package_aliases(package_id, alias);
CREATE INDEX IF NOT EXISTS idx_alias ON product_package_aliases(alias);

DO $$
DECLARE
    ref_col text := NULL;
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'package_id') THEN
        ref_col := 'package_id';
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'id') THEN
        ref_col := 'id';
    END IF;

    IF ref_col IS NOT NULL AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_package_aliases') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON c.conrelid = t.oid
            WHERE c.conname = 'fk_package_alias_package' AND t.relname = 'product_package_aliases'
        ) AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_package_aliases' AND column_name = 'package_id') THEN
            EXECUTE format('ALTER TABLE product_package_aliases ADD CONSTRAINT fk_package_alias_package FOREIGN KEY (package_id) REFERENCES product_packages(%I) ON DELETE CASCADE', ref_col);
        END IF;
    END IF;
END$$;
