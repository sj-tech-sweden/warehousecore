-- Migration 079: Add subcategory column to rental_equipment
ALTER TABLE rental_equipment
    ADD COLUMN IF NOT EXISTS subcategory VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_rental_equipment_category ON rental_equipment(category);
CREATE INDEX IF NOT EXISTS idx_rental_equipment_subcategory ON rental_equipment(subcategory);
