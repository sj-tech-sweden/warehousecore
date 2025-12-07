-- Add website visibility and image selection for products
ALTER TABLE products
  ADD COLUMN website_visible TINYINT(1) NOT NULL DEFAULT 0 AFTER price_per_unit,
  ADD COLUMN website_thumbnail VARCHAR(255) NULL AFTER website_visible,
  ADD COLUMN website_images_json JSON NULL AFTER website_thumbnail;

-- Add website visibility for packages (product packages table)
ALTER TABLE product_packages
  ADD COLUMN website_visible TINYINT(1) NOT NULL DEFAULT 0 AFTER description;
