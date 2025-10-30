-- Add LED controller registry tables

CREATE TABLE IF NOT EXISTS led_controllers (
  id INT AUTO_INCREMENT PRIMARY KEY,
  controller_id VARCHAR(128) NOT NULL UNIQUE,
  display_name VARCHAR(255) NOT NULL,
  topic_suffix VARCHAR(255) NOT NULL DEFAULT '',
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  last_seen DATETIME NULL DEFAULT NULL,
  metadata JSON NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS led_controller_zone_types (
  controller_id INT NOT NULL,
  zone_type_id INT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (controller_id, zone_type_id),
  CONSTRAINT fk_led_controller_zone_types_controller
    FOREIGN KEY (controller_id) REFERENCES led_controllers(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_led_controller_zone_types_zone_type
    FOREIGN KEY (zone_type_id) REFERENCES zone_types(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
