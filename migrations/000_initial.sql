-- Initial consolidated schema for WarehouseCore (minimal, idempotent)
-- This migration creates core tables required for a fresh WarehouseCore deployment.

BEGIN;

-- Roles (shared core)
CREATE TABLE IF NOT EXISTS roles (
  roleid SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL UNIQUE,
  display_name VARCHAR(150),
  description TEXT,
  scope VARCHAR(50) DEFAULT 'warehousecore',
  is_system_role BOOLEAN DEFAULT FALSE,
  is_active BOOLEAN DEFAULT TRUE,
  permissions JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Users (auth)
CREATE TABLE IF NOT EXISTS users (
  userid SERIAL PRIMARY KEY,
  username VARCHAR(100) NOT NULL UNIQUE,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API keys for service access
CREATE TABLE IF NOT EXISTS api_keys (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  api_key_hash CHAR(64) NOT NULL,
  is_active BOOLEAN DEFAULT TRUE,
  is_admin BOOLEAN DEFAULT FALSE,
  last_used_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products
CREATE TABLE IF NOT EXISTS products (
  productID SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  categoryID INT,
  manufacturerID INT,
  description TEXT,
  website_visible BOOLEAN DEFAULT FALSE
);

-- Devices (warehouse inventory)
CREATE TABLE IF NOT EXISTS devices (
  deviceID VARCHAR(255) PRIMARY KEY,
  productID INT REFERENCES products(productID),
  serialnumber TEXT,
  status VARCHAR(50) DEFAULT 'free',
  zone_id INT,
  condition_rating NUMERIC DEFAULT 5.0,
  usage_hours NUMERIC DEFAULT 0.0,
  barcode TEXT
);

-- Zones (storage locations)
CREATE TABLE IF NOT EXISTS zones (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  barcode VARCHAR(255),
  description TEXT
);

-- Cases
CREATE TABLE IF NOT EXISTS cases (
  caseID SERIAL PRIMARY KEY,
  name VARCHAR(255),
  description TEXT
);

-- Product packages
CREATE TABLE IF NOT EXISTS product_packages (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT
);

COMMIT;

-- Additional indexes, api_key hash sync, and helpers
BEGIN;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
CREATE INDEX IF NOT EXISTS idx_devices_product ON devices(productID);
CREATE INDEX IF NOT EXISTS idx_devices_zone ON devices(zone_id);

-- API key hashes are generated in application code (HMAC-SHA256 via HashAPIKey).

COMMIT;
