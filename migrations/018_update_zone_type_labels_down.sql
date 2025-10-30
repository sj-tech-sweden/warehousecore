-- Revert zone type label adjustments
UPDATE zone_types
SET label = 'Regal', description = 'Standard warehouse shelf'
WHERE `key` = 'shelf';

UPDATE zone_types
SET label = 'Rack', description = 'Equipment rack'
WHERE `key` = 'rack';
