-- Update zone types to support only Lager, Regal, and Gitterbox
-- Version 1.7 - 2025-10-14

ALTER TABLE storage_zones
MODIFY type ENUM('warehouse', 'rack', 'gitterbox', 'shelf', 'vehicle', 'stage', 'case', 'other')
NOT NULL DEFAULT 'other';

-- Note: 'warehouse' = Lager, 'rack' = Regal, 'gitterbox' = Gitterbox
-- Other types (shelf, vehicle, stage, case) kept for backward compatibility but should not be used
