-- Storage Zones: Logical warehouse areas (shelves, racks, vehicles, etc.)
CREATE TABLE IF NOT EXISTS `storage_zones` (
  `zone_id` INT AUTO_INCREMENT PRIMARY KEY,
  `code` VARCHAR(50) NOT NULL UNIQUE COMMENT 'Short code: SHELF-A1, RACK-B2, VAN-01',
  `name` VARCHAR(100) NOT NULL,
  `type` ENUM('shelf', 'rack', 'case', 'vehicle', 'stage', 'warehouse', 'other') NOT NULL DEFAULT 'other',
  `description` TEXT,
  `parent_zone_id` INT NULL COMMENT 'For hierarchical zones',
  `capacity` INT NULL COMMENT 'Maximum items this zone can hold',
  `location` VARCHAR(255) NULL COMMENT 'Physical location description',
  `metadata` JSON NULL COMMENT 'Flexible attributes',
  `is_active` BOOLEAN NOT NULL DEFAULT TRUE,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_zone_type (`type`),
  INDEX idx_zone_active (`is_active`),
  INDEX idx_zone_parent (`parent_zone_id`),
  FOREIGN KEY (`parent_zone_id`) REFERENCES `storage_zones`(`zone_id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Add zone reference to devices table if not exists
SET @dbname = DATABASE();
SET @tablename = 'devices';
SET @columnname = 'zone_id';
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  'SELECT 1',
  CONCAT('ALTER TABLE ', @tablename, ' ADD COLUMN ', @columnname, ' INT NULL AFTER current_location, ADD INDEX idx_device_zone (zone_id)')
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- Add zone reference to cases table if not exists
SET @tablename = 'cases';
SET @columnname = 'zone_id';
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  'SELECT 1',
  CONCAT('ALTER TABLE ', @tablename, ' ADD COLUMN zone_id INT NULL AFTER status, ADD COLUMN barcode VARCHAR(255) NULL AFTER zone_id, ADD COLUMN rfid_tag VARCHAR(255) NULL AFTER barcode, ADD INDEX idx_case_zone (zone_id)')
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;
