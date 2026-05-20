-- Migration 019: Create Product Packages Support
-- Similar to cases, but virtual packages of products for job assignment

CREATE TABLE IF NOT EXISTS product_packages (
    package_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_name ON product_packages(name);

CREATE TABLE IF NOT EXISTS product_package_items (
    package_item_id SERIAL PRIMARY KEY,
    package_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- If product_packages exists but uses `id` as PK (legacy), add a compatibility `package_id` generated column
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_packages')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'package_id')
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'id') THEN
        -- Add a generated column mirroring id so older migrations referencing package_id work
        EXECUTE 'ALTER TABLE product_packages ADD COLUMN IF NOT EXISTS package_id INT GENERATED ALWAYS AS (id) STORED';
        -- ensure an index exists for FK referencing
        EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS uq_product_packages_package_id ON product_packages(package_id)';
    END IF;
END$$;

DO $$
DECLARE
    ref_col text := NULL;
BEGIN
    -- Choose correct primary key column name on product_packages (package_id or id)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'package_id') THEN
        ref_col := 'package_id';
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'id') THEN
        ref_col := 'id';
    END IF;

    IF ref_col IS NOT NULL AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_package_items') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON c.conrelid = t.oid
            WHERE c.conname = 'fk_ppi_package' AND t.relname = 'product_package_items'
        ) AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_package_items' AND column_name = 'package_id') THEN
            EXECUTE format('ALTER TABLE product_package_items ADD CONSTRAINT fk_ppi_package FOREIGN KEY (package_id) REFERENCES product_packages(%I) ON DELETE CASCADE', ref_col);
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON c.conrelid = t.oid
            WHERE c.conname = 'fk_ppi_product' AND t.relname = 'product_package_items'
        ) AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_package_items' AND column_name = 'product_id') THEN
            EXECUTE 'ALTER TABLE product_package_items ADD CONSTRAINT fk_ppi_product FOREIGN KEY (product_id) REFERENCES products(productID) ON DELETE CASCADE';
        END IF;
    END IF;
END$$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_package_items' AND column_name = 'package_id') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_index i
            JOIN pg_class c ON i.indrelid = c.oid
            JOIN pg_class ic ON i.indexrelid = ic.oid
            WHERE c.relname = 'product_package_items' AND ic.relname = 'unique_package_product'
        ) THEN
            EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS unique_package_product ON product_package_items(package_id, product_id)';
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM pg_index i
            JOIN pg_class c ON i.indrelid = c.oid
            JOIN pg_class ic ON i.indexrelid = ic.oid
            WHERE c.relname = 'product_package_items' AND ic.relname = 'idx_package_id'
        ) THEN
            EXECUTE 'CREATE INDEX IF NOT EXISTS idx_package_id ON product_package_items(package_id)';
        END IF;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_package_items' AND column_name = 'product_id') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_index i
            JOIN pg_class c ON i.indrelid = c.oid
            JOIN pg_class ic ON i.indexrelid = ic.oid
            WHERE c.relname = 'product_package_items' AND ic.relname = 'idx_product_id'
        ) THEN
            EXECUTE 'CREATE INDEX IF NOT EXISTS idx_product_id ON product_package_items(product_id)';
        END IF;
    END IF;
END$$;
