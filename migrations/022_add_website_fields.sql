-- Add website visibility and image selection for products
ALTER TABLE products
  ADD COLUMN IF NOT EXISTS website_visible BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS website_thumbnail VARCHAR(255) NULL,
  ADD COLUMN IF NOT EXISTS website_images_json JSONB NULL;

-- Add website visibility for packages (product packages table)
ALTER TABLE product_packages
  ADD COLUMN IF NOT EXISTS website_visible BOOLEAN NOT NULL DEFAULT FALSE;
