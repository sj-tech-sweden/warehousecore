-- Migration 063: Add optional `product_id` column to `product_packages` for legacy variants
-- If some deployments store a single product directly on product_packages, create the column and try to populate from product_package_items.

BEGIN;

-- Add product_id column if missing
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='product_packages' AND column_name='product_id') THEN
    EXECUTE 'ALTER TABLE product_packages ADD COLUMN product_id INT NULL';
  END IF;
END$$;

-- If product_package_items exists, try to backfill product_id for packages that only have a single item
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='product_package_items') THEN
    -- For each package_id with exactly one product, copy product_id
    EXECUTE $sql$
      UPDATE product_packages pp
      SET product_id = sub.product_id
      FROM (
        SELECT package_id, MIN(product_id) AS product_id
        FROM product_package_items
        GROUP BY package_id
        HAVING COUNT(*) = 1
      ) AS sub
      WHERE pp.package_id = sub.package_id AND pp.product_id IS NULL;
    $sql$;
  END IF;
END$$;

-- Create index on product_id for lookups
CREATE INDEX IF NOT EXISTS idx_product_packages_product_id ON product_packages(product_id);

COMMIT;
