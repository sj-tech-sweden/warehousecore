-- Migration 020: Add package codes and OCR alias mapping for product packages

-- 1) Add package_code column (temporary nullable)
ALTER TABLE product_packages
    ADD COLUMN package_code VARCHAR(32) NULL AFTER package_id;

-- 2) Backfill existing rows with deterministic codes
UPDATE product_packages
SET package_code = CONCAT('PKG-', LPAD(package_id, 6, '0'))
WHERE package_code IS NULL OR package_code = '';

-- 3) Enforce NOT NULL + uniqueness
ALTER TABLE product_packages
    MODIFY COLUMN package_code VARCHAR(32) NOT NULL,
    ADD UNIQUE KEY uq_product_package_code (package_code);

-- 4) Create aliases table for OCR mapping
CREATE TABLE IF NOT EXISTS product_package_aliases (
    alias_id INT AUTO_INCREMENT PRIMARY KEY,
    package_id INT NOT NULL,
    alias VARCHAR(191) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_package_alias (package_id, alias),
    INDEX idx_alias (alias),
    CONSTRAINT fk_package_alias_package
        FOREIGN KEY (package_id) REFERENCES product_packages(package_id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
