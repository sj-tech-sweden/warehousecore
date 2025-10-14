-- Scan Events: Complete log of all barcode/QR scans
CREATE TABLE IF NOT EXISTS `scan_events` (
  `scan_id` BIGINT AUTO_INCREMENT PRIMARY KEY,
  `scan_code` VARCHAR(255) NOT NULL COMMENT 'The scanned barcode/QR code',
  `scan_type` ENUM('barcode', 'qr_code', 'rfid') NOT NULL DEFAULT 'barcode',
  `device_id` VARCHAR(50) NULL COMMENT 'Resolved device ID',
  `action` ENUM('intake', 'outtake', 'check', 'transfer') NULL,
  `job_id` BIGINT NULL COMMENT 'Associated job',
  `zone_id` INT NULL COMMENT 'Associated zone',
  `user_id` BIGINT NULL COMMENT 'User who scanned',
  `success` BOOLEAN NOT NULL DEFAULT TRUE,
  `error_message` TEXT NULL COMMENT 'Error if scan failed',
  `metadata` JSON NULL COMMENT 'Additional data',
  `ip_address` VARCHAR(45) NULL,
  `user_agent` TEXT NULL,
  `timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_scan_code (`scan_code`),
  INDEX idx_scan_device (`device_id`),
  INDEX idx_scan_job (`job_id`),
  INDEX idx_scan_timestamp (`timestamp`),
  INDEX idx_scan_success (`success`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
