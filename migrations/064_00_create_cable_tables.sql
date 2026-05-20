-- Ensure cable lookup tables exist for seeds and migrations
-- This migration is idempotent and safe to run multiple times.

BEGIN;

CREATE TABLE IF NOT EXISTS cable_connectors (
    cable_connectorsid SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    abbreviation VARCHAR(20),
    gender VARCHAR(10)
);

CREATE TABLE IF NOT EXISTS cable_types (
    cable_typesid SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS cables (
    cableid SERIAL PRIMARY KEY,
    connector1 INT NOT NULL REFERENCES cable_connectors(cable_connectorsid) ON DELETE RESTRICT,
    connector2 INT NOT NULL REFERENCES cable_connectors(cable_connectorsid) ON DELETE RESTRICT,
    typ INT NOT NULL REFERENCES cable_types(cable_typesid) ON DELETE RESTRICT,
    length DECIMAL(10,2) NOT NULL,
    mm2 DECIMAL(10,2),
    name VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_cables_connector1 ON cables(connector1);
CREATE INDEX IF NOT EXISTS idx_cables_connector2 ON cables(connector2);
CREATE INDEX IF NOT EXISTS idx_cables_type ON cables(typ);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_cable_connectors
    ON cable_connectors(name, abbreviation, gender);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_cable_types ON cable_types(name);

COMMIT;
