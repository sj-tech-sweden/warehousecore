-- Migration 059: Bridge missing jobdevices table and device QR code column
-- Adds minimal structures if they are missing so handlers work across schema variants.

BEGIN;

-- Add qr_code column to devices if missing
ALTER TABLE devices ADD COLUMN IF NOT EXISTS qr_code TEXT;

-- Create minimal jobdevices table if it does not exist. This table may be managed by RentalCore
-- in some deployments; create a minimal compatible version to avoid handler failures.
CREATE TABLE IF NOT EXISTS jobdevices (
  deviceID TEXT NOT NULL,
  jobID INT NOT NULL,
  pack_status TEXT,
  pack_ts TIMESTAMP NULL,
  PRIMARY KEY (deviceID, jobID)
);

-- Ensure product_dependencies has canonical column names ('product_id', 'dependency_product_id')
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_dependencies') THEN
    -- add product_id if missing and try to populate from common variants
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='product_dependencies' AND column_name='product_id') THEN
      EXECUTE 'ALTER TABLE product_dependencies ADD COLUMN product_id INT NULL';
      IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='product_dependencies' AND column_name='productID') THEN
        EXECUTE 'UPDATE product_dependencies SET product_id = productID WHERE product_id IS NULL';
      END IF;
    END IF;

    -- add dependency_product_id if missing
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='product_dependencies' AND column_name='dependency_product_id') THEN
      EXECUTE 'ALTER TABLE product_dependencies ADD COLUMN dependency_product_id INT NULL';
      IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='product_dependencies' AND column_name='dependencyProductID') THEN
        EXECUTE 'UPDATE product_dependencies SET dependency_product_id = dependencyProductID WHERE dependency_product_id IS NULL';
      END IF;
    END IF;

    -- Create index for product_id if not exists
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind = 'i' AND c.relname = 'idx_product_dependencies_product_id') THEN
      EXECUTE 'CREATE INDEX IF NOT EXISTS idx_product_dependencies_product_id ON product_dependencies(product_id)';
    END IF;
  END IF;
END$$;

COMMIT;
