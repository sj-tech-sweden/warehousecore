-- Adjust default zone type labels so that "shelf" corresponds to Fächer and "rack" to Regale
UPDATE zone_types
SET label = 'Fach', description = 'Einzele Fach / Lagerfach'
WHERE `key` = 'shelf';

UPDATE zone_types
SET label = 'Regal', description = 'Regal oder Gestell'
WHERE `key` = 'rack';
