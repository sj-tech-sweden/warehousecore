-- Migration 067: Cleanup duplicates and ensure product_packages.price + rental_equipment
-- Idempotent: safe to run multiple times on partially-migrated DBs.

DO $$
BEGIN
    -- 1) Ensure product_packages.price exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_packages') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_packages' AND column_name = 'price') THEN
            EXECUTE 'ALTER TABLE product_packages ADD COLUMN IF NOT EXISTS price DECIMAL(10,2) DEFAULT 0.00';
        END IF;
    END IF;

    -- 2) Create a minimal rental_equipment table if missing (compatibility)
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'rental_equipment') THEN
        EXECUTE '
            CREATE TABLE rental_equipment (
                equipment_id SERIAL PRIMARY KEY,
                name TEXT NOT NULL,
                supplier INT NULL,
                supplier_name VARCHAR(255),
                product_name VARCHAR(255),
                is_active BOOLEAN DEFAULT TRUE,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_rental_equipment_supplier ON rental_equipment(supplier)';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_rental_equipment_active ON rental_equipment(is_active)';
    END IF;

    -- 3) Deduplicate brands (case-insensitive): update referencing products then remove duplicate brand rows
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'brands') THEN
        -- Update products to point to canonical brand
        UPDATE products p
        SET brandID = m.keep_id
        FROM (
            SELECT lower(name) AS nl, min(brandID) AS keep_id
            FROM brands
            GROUP BY lower(name)
            HAVING count(*) > 1
        ) m
        JOIN brands b ON lower(b.name) = m.nl AND b.brandID <> m.keep_id
        WHERE p.brandID = b.brandID;

        -- Delete duplicate brand rows keeping the smallest brandID per lowercase(name)
        DELETE FROM brands b
        USING (
            SELECT lower(name) AS nl, min(brandID) AS keep_id
            FROM brands
            GROUP BY lower(name)
            HAVING count(*) > 1
        ) m
        WHERE lower(b.name) = m.nl AND b.brandID <> m.keep_id;

        -- Ensure unique index on lower(name)
        IF NOT EXISTS (
            SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE c.relkind = 'i' AND c.relname = 'uq_brands_name_lower'
        ) THEN
            EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS uq_brands_name_lower ON brands ((lower(name)))';
        END IF;
    END IF;

    -- 4) Deduplicate categories similarly
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'categories') THEN
        UPDATE products p
        SET categoryID = m.keep_id
        FROM (
            SELECT lower(name) AS nl, min(categoryID) AS keep_id
            FROM categories
            GROUP BY lower(name)
            HAVING count(*) > 1
        ) m
        JOIN categories c ON lower(c.name) = m.nl AND c.categoryID <> m.keep_id
        WHERE p.categoryID = c.categoryID;

        DELETE FROM categories c
        USING (
            SELECT lower(name) AS nl, min(categoryID) AS keep_id
            FROM categories
            GROUP BY lower(name)
            HAVING count(*) > 1
        ) m
        WHERE lower(c.name) = m.nl AND c.categoryID <> m.keep_id;

        IF NOT EXISTS (
            SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE c.relkind = 'i' AND c.relname = 'uq_categories_name_lower'
        ) THEN
            EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS uq_categories_name_lower ON categories ((lower(name)))';
        END IF;
    END IF;

    -- 5) Rebuild cable-related product_field_definitions.options from current cable tables
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_field_definitions') THEN
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'cable_connectors') THEN
            UPDATE product_field_definitions
            SET options = (SELECT COALESCE(json_agg(name ORDER BY name), '[]'::json)::text FROM cable_connectors)
            WHERE name IN ('connector_1','connector_2');
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'cable_types') THEN
            UPDATE product_field_definitions
            SET options = (SELECT COALESCE(json_agg(name ORDER BY name), '[]'::json)::text FROM cable_types)
            WHERE name = 'cable_type';
        END IF;
    END IF;

END$$;

-- Done.
