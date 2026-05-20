-- Migration 057: Create admin tables (categories, subcategories, brands, manufacturers, count_types)
-- and ensure api_keys has expected columns (api_key_hash, last_used_at, is_admin)

BEGIN;

-- Ensure api_keys has the newer columns used by the app
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS api_key_hash CHAR(64);
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMP NULL;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

-- Ensure a canonical index exists for api_key_hash when present
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'api_keys' AND column_name = 'api_key_hash') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE c.relkind = 'i' AND c.relname = 'idx_api_keys_api_key_hash'
        ) THEN
            EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_api_key_hash ON api_keys(api_key_hash)';
        END IF;
    END IF;
END$$;

-- Categories (top-level)
CREATE TABLE IF NOT EXISTS categories (
  categoryID SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  abbreviation VARCHAR(50)
);

-- Create sequence for string IDs used in subcategory/subbiercategory to preserve string-type IDs
CREATE SEQUENCE IF NOT EXISTS subcategories_seq;
CREATE SEQUENCE IF NOT EXISTS subbiercategories_seq;

-- Subcategories (second-level). Use TEXT primary key defaulting to sequence cast to text so handlers expecting strings continue to work.
CREATE TABLE IF NOT EXISTS subcategories (
  subcategoryID TEXT PRIMARY KEY DEFAULT nextval('subcategories_seq')::text,
  name TEXT NOT NULL,
  abbreviation VARCHAR(50),
  categoryID INT REFERENCES categories(categoryID) ON DELETE SET NULL
);

-- Sub-subcategories (third-level)
CREATE TABLE IF NOT EXISTS subbiercategories (
  subbiercategoryID TEXT PRIMARY KEY DEFAULT nextval('subbiercategories_seq')::text,
  name TEXT NOT NULL,
  abbreviation VARCHAR(50),
  subcategoryID TEXT REFERENCES subcategories(subcategoryID) ON DELETE SET NULL
);

-- Brands and manufacturers
CREATE TABLE IF NOT EXISTS brands (
  brandID SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  abbreviation VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS manufacturer (
  manufacturerID SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  abbreviation VARCHAR(50)
);

-- Count types / measurement units
CREATE TABLE IF NOT EXISTS count_types (
  count_type_id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  abbreviation VARCHAR(20),
  is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Ensure products table has the expected columns used by handlers
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'subcategoryid') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN subcategoryID TEXT NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'subbiercategoryid') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN subbiercategoryID TEXT NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'brandid') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN brandID INT NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'maintenanceinterval') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN maintenanceInterval INT NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'itemcostperday') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN itemcostperday NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'weight') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN weight NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'height') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN height NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'width') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN width NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'depth') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN depth NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'powerconsumption') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN powerconsumption NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'pos_in_category') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN pos_in_category INT NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'is_accessory') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN is_accessory BOOLEAN DEFAULT FALSE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'is_consumable') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN is_consumable BOOLEAN DEFAULT FALSE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'count_type_id') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN count_type_id INT NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'stock_quantity') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN stock_quantity NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'min_stock_level') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN min_stock_level NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'generic_barcode') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN generic_barcode TEXT NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'price_per_unit') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN price_per_unit NUMERIC NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'website_visible') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN website_visible BOOLEAN DEFAULT FALSE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'website_thumbnail') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN website_thumbnail TEXT NULL';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'products' AND lower(column_name) = 'website_images_json') THEN
        EXECUTE 'ALTER TABLE products ADD COLUMN website_images_json TEXT NULL';
    END IF;
END$$;

COMMIT;
