-- Migration 011: Sync user_roles_wh assignments into shared user_roles table

-- Create user_roles if it does not exist (following RentalCore schema shape)
CREATE TABLE IF NOT EXISTS `user_roles` (
  `userID` bigint UNSIGNED NOT NULL,
  `roleID` int NOT NULL,
  `assigned_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `assigned_by` bigint UNSIGNED DEFAULT NULL,
  `expires_at` timestamp NULL DEFAULT NULL,
  `is_active` tinyint(1) DEFAULT '1',
  UNIQUE KEY `uniq_user_role` (`userID`,`roleID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- If warehouse-specific mapping table exists, copy assignments across
SET @wh_exists = (
  SELECT COUNT(*) FROM information_schema.TABLES
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_roles_wh'
);

SET @sql := IF(@wh_exists > 0,
  'INSERT IGNORE INTO user_roles (userID, roleID, assigned_at, assigned_by, is_active)\n   SELECT user_id, role_id, NOW(), NULL, 1 FROM user_roles_wh',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

