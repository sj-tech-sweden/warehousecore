-- Migration: 043_product_custom_fields.sql
-- Description: Add product custom field definitions and values tables, then migrate
--              existing cable data (cables, cable_connectors, cable_types) into the
--              generic products + field-values model and clean up the cable-specific tables.
-- Date: 2026-04-13

BEGIN;

-- ---------------------------------------------------------------------------
-- 1. product_field_definitions
--    Stores the schema for each dynamic attribute that can be attached to a
--    product (e.g. "connector_1", "cable_length").
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS product_field_definitions (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100)  NOT NULL UNIQUE,   -- machine key, e.g. "connector_1"
    label       VARCHAR(200)  NOT NULL,           -- human label,  e.g. "Connector 1"
    field_type  VARCHAR(50)   NOT NULL,           -- 'text' | 'number' | 'integer' | 'select' | 'boolean'
    options     TEXT,                             -- JSON array of strings for 'select' fields, NULL otherwise
    unit        VARCHAR(50),                      -- optional unit suffix, e.g. "m", "mm²"
    sort_order  INTEGER       NOT NULL DEFAULT 0,
    is_required BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- 2. product_field_values
--    Stores the actual value for each (product, field_definition) pair.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS product_field_values (
    id                  SERIAL PRIMARY KEY,
    product_id          INTEGER NOT NULL REFERENCES products(productID) ON DELETE CASCADE,
    field_definition_id INTEGER NOT NULL REFERENCES product_field_definitions(id) ON DELETE CASCADE,
    value               TEXT    NOT NULL,
    UNIQUE (product_id, field_definition_id)
);

CREATE INDEX IF NOT EXISTS idx_pfv_product_id          ON product_field_values (product_id);
CREATE INDEX IF NOT EXISTS idx_pfv_field_definition_id ON product_field_values (field_definition_id);

-- ---------------------------------------------------------------------------
-- 3. Migrate cable data — only executed when the cables table still exists.
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'cables'
    ) THEN
        RAISE NOTICE 'cables table not found – skipping cable data migration.';
        RETURN;
    END IF;

    -- -----------------------------------------------------------------------
    -- 3a. Ensure the "Cables" category exists.
    -- -----------------------------------------------------------------------
    INSERT INTO categories (name)
    SELECT 'Cables'
    WHERE NOT EXISTS (
        SELECT 1 FROM categories WHERE LOWER(name) = 'cables'
    );

    -- -----------------------------------------------------------------------
    -- 3b. Insert field definitions for cable attributes.
    -- -----------------------------------------------------------------------
    INSERT INTO product_field_definitions (name, label, field_type, options, unit, sort_order)
    VALUES
        ('connector_1', 'Connector 1',    'select', NULL, NULL,   1),
        ('connector_2', 'Connector 2',    'select', NULL, NULL,   2),
        ('cable_type',  'Cable Type',     'select', NULL, NULL,   3),
        ('cable_length','Length',         'number', NULL, 'm',    4),
        ('cable_mm2',   'Cross-section',  'number', NULL, 'mm²',  5)
    ON CONFLICT (name) DO NOTHING;

    -- Populate select options from the lookup tables.
    UPDATE product_field_definitions
    SET options = (
        SELECT COALESCE(json_agg(name ORDER BY name), '[]'::json)::text FROM cable_connectors
    )
    WHERE name IN ('connector_1', 'connector_2');

    UPDATE product_field_definitions
    SET options = (
        SELECT COALESCE(json_agg(name ORDER BY name), '[]'::json)::text FROM cable_types
    )
    WHERE name = 'cable_type';

    -- -----------------------------------------------------------------------
    -- 3c. For each cable, create a product and insert its field values.
    -- -----------------------------------------------------------------------
    DECLARE
        v_cable_category_id INTEGER;
        v_cable             RECORD;
        v_product_id        INTEGER;
        v_fd_conn1          INTEGER;
        v_fd_conn2          INTEGER;
        v_fd_type           INTEGER;
        v_fd_length         INTEGER;
        v_fd_mm2            INTEGER;
        v_dual_count        INTEGER;
    BEGIN
        SELECT categoryID INTO v_cable_category_id
        FROM categories WHERE LOWER(name) = 'cables' LIMIT 1;

        SELECT id INTO v_fd_conn1  FROM product_field_definitions WHERE name = 'connector_1';
        SELECT id INTO v_fd_conn2  FROM product_field_definitions WHERE name = 'connector_2';
        SELECT id INTO v_fd_type   FROM product_field_definitions WHERE name = 'cable_type';
        SELECT id INTO v_fd_length FROM product_field_definitions WHERE name = 'cable_length';
        SELECT id INTO v_fd_mm2    FROM product_field_definitions WHERE name = 'cable_mm2';

        FOR v_cable IN
            SELECT c.*,
                   cc1.name AS c1name,
                   cc2.name AS c2name,
                   ct.name  AS ctname
            FROM cables c
            LEFT JOIN cable_connectors cc1 ON c.connector1 = cc1.cable_connectorsID
            LEFT JOIN cable_connectors cc2 ON c.connector2 = cc2.cable_connectorsID
            LEFT JOIN cable_types      ct  ON c.typ        = ct.cable_typesID
        LOOP
            -- Create a product for this cable.
            INSERT INTO products (name, categoryID, description)
            VALUES (
                COALESCE(v_cable.name, 'Cable #' || v_cable.cableID),
                v_cable_category_id,
                NULL
            )
            RETURNING productID INTO v_product_id;

            -- Report and migrate ALL devices associated with this cable.
            -- Devices that already had a productID would silently lose their cable
            -- association when cable_id is dropped; warn before overwriting.
            -- Guard this section so the migration remains idempotent if a prior
            -- cleanup already removed devices.cable_id in a partially-migrated
            -- environment.
            IF EXISTS (
                SELECT 1
                FROM information_schema.columns
                WHERE table_schema = ANY (current_schemas(FALSE))
                  AND table_name = 'devices'
                  AND column_name = 'cable_id'
            ) THEN
                SELECT COUNT(*) INTO v_dual_count
                FROM devices
                WHERE cable_id = v_cable.cableID AND productID IS NOT NULL;

                IF v_dual_count > 0 THEN
                    RAISE NOTICE 'Cable "%" (cableID=%): % device(s) had both productID and cable_id set. '
                                 'Their productID is being updated to the new cable-product (%).',
                        v_cable.name, v_cable.cableID, v_dual_count, v_product_id;
                END IF;

                UPDATE devices
                SET productID = v_product_id
                WHERE cable_id = v_cable.cableID;
            END IF;
            -- Insert field values, skipping NULLs silently.
            IF v_cable.c1name IS NOT NULL THEN
                INSERT INTO product_field_values (product_id, field_definition_id, value)
                VALUES (v_product_id, v_fd_conn1, v_cable.c1name)
                ON CONFLICT (product_id, field_definition_id) DO NOTHING;
            END IF;

            IF v_cable.c2name IS NOT NULL THEN
                INSERT INTO product_field_values (product_id, field_definition_id, value)
                VALUES (v_product_id, v_fd_conn2, v_cable.c2name)
                ON CONFLICT (product_id, field_definition_id) DO NOTHING;
            END IF;

            IF v_cable.ctname IS NOT NULL THEN
                INSERT INTO product_field_values (product_id, field_definition_id, value)
                VALUES (v_product_id, v_fd_type, v_cable.ctname)
                ON CONFLICT (product_id, field_definition_id) DO NOTHING;
            END IF;

            INSERT INTO product_field_values (product_id, field_definition_id, value)
            VALUES (v_product_id, v_fd_length, v_cable.length::text)
            ON CONFLICT (product_id, field_definition_id) DO NOTHING;

            IF v_cable.mm2 IS NOT NULL THEN
                INSERT INTO product_field_values (product_id, field_definition_id, value)
                VALUES (v_product_id, v_fd_mm2, v_cable.mm2::text)
                ON CONFLICT (product_id, field_definition_id) DO NOTHING;
            END IF;

        END LOOP;
    END;

END $$;

-- ---------------------------------------------------------------------------
-- 3d. Clean up cable-specific columns and tables.
--     These DDL statements always run (guarded by IF EXISTS) so that a
--     partially-migrated environment (cables already gone, column still
--     present) is also correctly cleaned up.
-- ---------------------------------------------------------------------------

-- Remove FK constraint and index added by migration 042, then drop the column.
ALTER TABLE devices DROP CONSTRAINT IF EXISTS fk_devices_cable_id;
DROP  INDEX  IF EXISTS idx_devices_cable_id;
ALTER TABLE devices DROP COLUMN IF EXISTS cable_id;

-- Drop cable tables (FK-safe order: dependent first).
DROP TABLE IF EXISTS cables;
DROP TABLE IF EXISTS cable_connectors;
DROP TABLE IF EXISTS cable_types;

COMMIT;
