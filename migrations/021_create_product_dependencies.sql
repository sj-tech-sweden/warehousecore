-- Migration: Create product_dependencies table
-- This table stores relationships between products and their optional dependencies
-- (e.g., a fog machine might suggest fog fluid as a dependency)

CREATE TABLE IF NOT EXISTS product_dependencies (
    id INT AUTO_INCREMENT PRIMARY KEY,
    product_id INT NOT NULL COMMENT 'Main product that has the dependency',
    dependency_product_id INT NOT NULL COMMENT 'The dependent product (accessory/consumable)',
    is_optional BOOLEAN DEFAULT TRUE COMMENT 'Whether the dependency is optional (shows as suggestion)',
    default_quantity DECIMAL(10,2) DEFAULT 1.0 COMMENT 'Suggested quantity for this dependency',
    notes VARCHAR(500) COMMENT 'Optional notes about why this dependency is needed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (product_id) REFERENCES products(productID) ON DELETE CASCADE,
    FOREIGN KEY (dependency_product_id) REFERENCES products(productID) ON DELETE CASCADE,

    UNIQUE KEY unique_dependency (product_id, dependency_product_id),
    INDEX idx_product_id (product_id),
    INDEX idx_dependency_product_id (dependency_product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Stores product dependencies for job assignment suggestions';
