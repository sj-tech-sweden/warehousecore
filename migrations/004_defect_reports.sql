-- Defect Reports: Detailed defect tracking beyond basic maintenance logs
CREATE TABLE IF NOT EXISTS `defect_reports` (
  `defect_id` BIGINT AUTO_INCREMENT PRIMARY KEY,
  `device_id` VARCHAR(50) NOT NULL,
  `severity` ENUM('low', 'medium', 'high', 'critical') NOT NULL DEFAULT 'medium',
  `status` ENUM('open', 'in_progress', 'repaired', 'closed') NOT NULL DEFAULT 'open',
  `title` VARCHAR(200) NOT NULL,
  `description` TEXT NOT NULL,
  `reported_by` BIGINT NULL COMMENT 'User who reported',
  `reported_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `assigned_to` BIGINT NULL COMMENT 'Technician assigned',
  `repaired_by` BIGINT NULL COMMENT 'Technician who repaired',
  `repaired_at` TIMESTAMP NULL,
  `repair_cost` DECIMAL(10,2) NULL,
  `repair_notes` TEXT NULL,
  `closed_at` TIMESTAMP NULL,
  `images` JSON NULL COMMENT 'Array of image URLs',
  `metadata` JSON NULL,
  INDEX idx_defect_device (`device_id`),
  INDEX idx_defect_status (`status`),
  INDEX idx_defect_severity (`severity`),
  INDEX idx_defect_reported (`reported_at`),
  FOREIGN KEY (`device_id`) REFERENCES `devices`(`deviceID`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Inspection Schedules: Periodic inspection requirements
CREATE TABLE IF NOT EXISTS `inspection_schedules` (
  `schedule_id` BIGINT AUTO_INCREMENT PRIMARY KEY,
  `device_id` VARCHAR(50) NULL COMMENT 'Specific device, if NULL applies to product type',
  `product_id` INT NULL COMMENT 'Product type, applies to all devices of this type',
  `inspection_type` VARCHAR(100) NOT NULL COMMENT 'Safety, electrical, visual, etc.',
  `interval_days` INT NOT NULL COMMENT 'Days between inspections',
  `last_inspection` TIMESTAMP NULL,
  `next_inspection` TIMESTAMP NULL,
  `is_active` BOOLEAN NOT NULL DEFAULT TRUE,
  `notes` TEXT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_inspection_device (`device_id`),
  INDEX idx_inspection_product (`product_id`),
  INDEX idx_inspection_next (`next_inspection`),
  INDEX idx_inspection_active (`is_active`),
  FOREIGN KEY (`device_id`) REFERENCES `devices`(`deviceID`) ON DELETE CASCADE,
  FOREIGN KEY (`product_id`) REFERENCES `products`(`productID`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
