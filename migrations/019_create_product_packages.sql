-- Migration 019: Create Product Packages Support
-- Similar to cases, but virtual packages of products for job assignment

CREATE TABLE IF NOT EXISTS product_packages (
    package_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS product_package_items (
    package_item_id INT AUTO_INCREMENT PRIMARY KEY,
    package_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (package_id) REFERENCES product_packages(package_id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(productID) ON DELETE CASCADE,
    UNIQUE KEY unique_package_product (package_id, product_id),
    INDEX idx_package_id (package_id),
    INDEX idx_product_id (product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
