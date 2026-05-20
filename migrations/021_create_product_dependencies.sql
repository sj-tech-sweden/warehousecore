-- Migration: Create product_dependencies table
-- This table stores relationships between products and their optional dependencies
-- (e.g., a fog machine might suggest fog fluid as a dependency)

CREATE TABLE IF NOT EXISTS product_dependencies (
        id SERIAL PRIMARY KEY,
        product_id INT NOT NULL,
        dependency_product_id INT NOT NULL,
        is_optional BOOLEAN DEFAULT TRUE,
        default_quantity DECIMAL(10,2) DEFAULT 1.0,
        notes VARCHAR(500),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        WHERE c.conname = 'fk_pd_product' AND t.relname = 'product_dependencies'
    ) THEN
        EXECUTE 'ALTER TABLE product_dependencies ADD CONSTRAINT fk_pd_product FOREIGN KEY (product_id) REFERENCES products(productID) ON DELETE CASCADE';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        WHERE c.conname = 'fk_pd_dependency' AND t.relname = 'product_dependencies'
    ) THEN
        EXECUTE 'ALTER TABLE product_dependencies ADD CONSTRAINT fk_pd_dependency FOREIGN KEY (dependency_product_id) REFERENCES products(productID) ON DELETE CASCADE';
    END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS unique_dependency ON product_dependencies(product_id, dependency_product_id);
CREATE INDEX IF NOT EXISTS idx_product_id ON product_dependencies(product_id);
CREATE INDEX IF NOT EXISTS idx_dependency_product_id ON product_dependencies(dependency_product_id);

COMMENT ON TABLE product_dependencies IS 'Stores product dependencies for job assignment suggestions';
