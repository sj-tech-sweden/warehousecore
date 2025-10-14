-- Device Movements: Audit trail of all physical device movements
CREATE TABLE IF NOT EXISTS `device_movements` (
  `movement_id` BIGINT AUTO_INCREMENT PRIMARY KEY,
  `device_id` VARCHAR(50) NOT NULL,
  `action` ENUM('intake', 'outtake', 'transfer', 'return', 'move') NOT NULL,
  `from_zone_id` INT NULL COMMENT 'Origin zone',
  `to_zone_id` INT NULL COMMENT 'Destination zone',
  `from_job_id` BIGINT NULL COMMENT 'Job device came from',
  `to_job_id` BIGINT NULL COMMENT 'Job device went to',
  `barcode` VARCHAR(255) NULL COMMENT 'Scanned barcode/QR code',
  `user_id` BIGINT NULL COMMENT 'User who performed the movement',
  `notes` TEXT NULL,
  `metadata` JSON NULL COMMENT 'Additional context',
  `timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_movement_device (`device_id`),
  INDEX idx_movement_action (`action`),
  INDEX idx_movement_timestamp (`timestamp`),
  INDEX idx_movement_from_zone (`from_zone_id`),
  INDEX idx_movement_to_zone (`to_zone_id`),
  INDEX idx_movement_job (`to_job_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
