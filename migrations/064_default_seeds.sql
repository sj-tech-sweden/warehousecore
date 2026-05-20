-- Default storage zones
INSERT INTO storage_zones (code, name, type, description, is_active) VALUES
('MAIN-WH', 'Main warehouse', 'warehouse', 'Primary storage location', TRUE),
('STAGE', 'Staging area', 'stage', 'Job preparation area', TRUE)
ON CONFLICT DO NOTHING;

-- Default label template: avoid referencing `template_type` if the column
-- doesn't exist (older schemas). Use a conditional DO block so this seed
-- is safe to run on partially-updated databases.
DO $seed$
DECLARE
    has_template_json BOOL := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'label_templates' AND column_name = 'template_json');
BEGIN
    -- Prefer newer columns `width_mm`/`height_mm` when present.
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'label_templates' AND column_name = 'width_mm'
    ) THEN
        -- If legacy `width` exists and is NOT NULL, also set it to avoid
        -- violating NOT NULL constraints when both columns are present.
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'label_templates' AND column_name = 'width'
        ) THEN
            IF EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_name = 'label_templates' AND column_name = 'template_type'
            ) THEN
                IF has_template_json THEN
                    INSERT INTO label_templates (name, description, template_type, width_mm, height_mm, width, height, template_json, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, 62, 29, '{}'::jsonb, TRUE)
                    ON CONFLICT DO NOTHING;
                ELSE
                    INSERT INTO label_templates (name, description, template_type, width_mm, height_mm, width, height, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, 62, 29, TRUE)
                    ON CONFLICT DO NOTHING;
                END IF;
            ELSE
                IF has_template_json THEN
                    INSERT INTO label_templates (name, description, width_mm, height_mm, width, height, template_json, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, 62, 29, '{}'::jsonb, TRUE)
                    ON CONFLICT DO NOTHING;
                ELSE
                    INSERT INTO label_templates (name, description, width_mm, height_mm, width, height, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, 62, 29, TRUE)
                    ON CONFLICT DO NOTHING;
                END IF;
            END IF;
        ELSE
            IF EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_name = 'label_templates' AND column_name = 'template_type'
            ) THEN
                IF has_template_json THEN
                    INSERT INTO label_templates (name, description, template_type, width_mm, height_mm, template_json, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, '{}'::jsonb, TRUE)
                    ON CONFLICT DO NOTHING;
                ELSE
                    INSERT INTO label_templates (name, description, template_type, width_mm, height_mm, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, TRUE)
                    ON CONFLICT DO NOTHING;
                END IF;
            ELSE
                IF has_template_json THEN
                    INSERT INTO label_templates (name, description, width_mm, height_mm, template_json, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, '{}'::jsonb, TRUE)
                    ON CONFLICT DO NOTHING;
                ELSE
                    INSERT INTO label_templates (name, description, width_mm, height_mm, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, TRUE)
                    ON CONFLICT DO NOTHING;
                END IF;
            END IF;
        END IF;

    -- Fallback to older `width`/`height` columns if present (non-mm units expected).
    ELSIF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'label_templates' AND column_name = 'width'
    ) THEN
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'label_templates' AND column_name = 'template_type'
        ) THEN
            IF has_template_json THEN
                INSERT INTO label_templates (name, description, template_type, width, height, template_json, is_default)
                VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, '{}'::jsonb, TRUE)
                ON CONFLICT DO NOTHING;
            ELSE
                INSERT INTO label_templates (name, description, template_type, width, height, is_default)
                VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, TRUE)
                ON CONFLICT DO NOTHING;
            END IF;
        ELSE
            IF has_template_json THEN
                INSERT INTO label_templates (name, description, width, height, template_json, is_default)
                VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, '{}'::jsonb, TRUE)
                ON CONFLICT DO NOTHING;
            ELSE
                INSERT INTO label_templates (name, description, width, height, is_default)
                VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, TRUE)
                ON CONFLICT DO NOTHING;
            END IF;
        END IF;

    -- Last resort: insert only name+description if neither width column exists.
    ELSE
        INSERT INTO label_templates (name, description) VALUES
        ('Standard Equipment Label', 'Standard Equipment Label 62x29mm')
        ON CONFLICT DO NOTHING;
    END IF;
END
$seed$;

-- Default count types for accessories/consumables
DO $seed$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'count_types' AND column_name = 'is_decimal') THEN
        INSERT INTO count_types (name, abbreviation, is_decimal) VALUES
        ('Piece', 'Pcs', FALSE),
        ('Kilogram', 'kg', TRUE),
        ('Liter', 'L', TRUE),
        ('Meter', 'm', TRUE),
        ('Square meter', 'm²', TRUE)
        ON CONFLICT DO NOTHING;
    ELSE
        INSERT INTO count_types (name, abbreviation) VALUES
        ('Piece', 'Pcs'),
        ('Kilogram', 'kg'),
        ('Liter', 'L'),
        ('Meter', 'm'),
        ('Square meter', 'm²')
        ON CONFLICT DO NOTHING;
    END IF;
END$seed$;

-- Default cable connectors
INSERT INTO cable_connectors (name, abbreviation, gender)
SELECT name, abbreviation, gender FROM (VALUES
    ('Schuko', 'SCH', 'male'),
    ('Schuko coupling', 'SCH', 'female'),
    ('CEE 16A blue', 'CEE16', 'male'),
    ('CEE 16A blue coupling', 'CEE16', 'female'),
    ('CEE 16A red', 'CEE16R', 'male'),
    ('CEE 16A red coupling', 'CEE16R', 'female'),
    ('CEE 32A red', 'CEE32', 'male'),
    ('CEE 32A red coupling', 'CEE32', 'female'),
    ('CEE 63A red', 'CEE63', 'male'),
    ('CEE 63A red coupling', 'CEE63', 'female')
) AS v(name, abbreviation, gender)
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'cable_connectors')
ON CONFLICT DO NOTHING;
DO $seed$
DECLARE
    has_width_mm BOOL := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'label_templates' AND column_name = 'width_mm');
    has_width BOOL := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'label_templates' AND column_name = 'width');
    has_template_type BOOL := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'label_templates' AND column_name = 'template_type');
    has_template_json BOOL := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'label_templates' AND column_name = 'template_json');
    default_json JSONB := $json$[
        {"type":"qrcode","x":2,"y":2,"width":25,"height":25,"rotation":0,"content":"device_id","style":{"format":"qr"}},
        {"type":"text","x":30,"y":5,"width":30,"height":8,"rotation":0,"content":"product","style":{"font_size":12,"font_weight":"bold","font_family":"Arial","color":"#000000","alignment":"left"}},
        {"type":"text","x":30,"y":15,"width":30,"height":6,"rotation":0,"content":"device_id","style":{"font_size":10,"font_weight":"normal","font_family":"Arial","color":"#666666","alignment":"left"}}
    ]$json$::jsonb;
BEGIN
    -- Build appropriate INSERT per available columns. We always prefer to
    -- populate both legacy (`width`/`height`) and new (`width_mm`/`height_mm`)
    -- where possible to satisfy NOT NULL constraints.
    IF has_width_mm THEN
        IF has_width THEN
            IF has_template_type THEN
                IF has_template_json THEN
                    INSERT INTO label_templates (name, description, template_type, width_mm, height_mm, width, height, template_json, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, 62, 29, default_json, TRUE)
                    ON CONFLICT DO NOTHING;
                ELSE
                    INSERT INTO label_templates (name, description, template_type, width_mm, height_mm, width, height, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, 62, 29, TRUE)
                    ON CONFLICT DO NOTHING;
                END IF;
            ELSE
                IF has_template_json THEN
                    INSERT INTO label_templates (name, description, width_mm, height_mm, width, height, template_json, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, 62, 29, default_json, TRUE)
                    ON CONFLICT DO NOTHING;
                ELSE
                    INSERT INTO label_templates (name, description, width_mm, height_mm, width, height, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, 62, 29, TRUE)
                    ON CONFLICT DO NOTHING;
                END IF;
            END IF;
        ELSE
            IF has_template_type THEN
                IF has_template_json THEN
                    INSERT INTO label_templates (name, description, template_type, width_mm, height_mm, template_json, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, default_json, TRUE)
                    ON CONFLICT DO NOTHING;
                ELSE
                    INSERT INTO label_templates (name, description, template_type, width_mm, height_mm, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, TRUE)
                    ON CONFLICT DO NOTHING;
                END IF;
            ELSE
                IF has_template_json THEN
                    INSERT INTO label_templates (name, description, width_mm, height_mm, template_json, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, default_json, TRUE)
                    ON CONFLICT DO NOTHING;
                ELSE
                    INSERT INTO label_templates (name, description, width_mm, height_mm, is_default)
                    VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, TRUE)
                    ON CONFLICT DO NOTHING;
                END IF;
            END IF;
        END IF;

    ELSIF has_width THEN
        IF has_template_type THEN
            IF has_template_json THEN
                INSERT INTO label_templates (name, description, template_type, width, height, template_json, is_default)
                VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, default_json, TRUE)
                ON CONFLICT DO NOTHING;
            ELSE
                INSERT INTO label_templates (name, description, template_type, width, height, is_default)
                VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 'device', 62, 29, TRUE)
                ON CONFLICT DO NOTHING;
            END IF;
        ELSE
            IF has_template_json THEN
                INSERT INTO label_templates (name, description, width, height, template_json, is_default)
                VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, default_json, TRUE)
                ON CONFLICT DO NOTHING;
            ELSE
                INSERT INTO label_templates (name, description, width, height, is_default)
                VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', 62, 29, TRUE)
                ON CONFLICT DO NOTHING;
            END IF;
        END IF;

    ELSE
        -- No width columns present; insert minimal record but include
        -- template_json if available to satisfy NOT NULL constraint.
        IF has_template_json THEN
            INSERT INTO label_templates (name, description, template_json, is_default)
            VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', default_json, TRUE)
            ON CONFLICT DO NOTHING;
        ELSE
            INSERT INTO label_templates (name, description, is_default)
            VALUES ('Standard Equipment Label', 'Standard Equipment Label 62x29mm', TRUE)
            ON CONFLICT DO NOTHING;
        END IF;
    END IF;
END
$seed$;
-- Insert specific categories without `code`/`parentid` (older schemas)
INSERT INTO categories (name, abbreviation) VALUES
('Routers & Access Points','RAP'),
('Servers & Compute',      'SRV')
ON CONFLICT DO NOTHING;

-- Default manufacturers
INSERT INTO manufacturer (name, website)
SELECT v.name, v.website
FROM (
    VALUES
    ('Shure',                  'https://www.shure.com'),
    ('Sennheiser',              'https://www.sennheiser.com'),
    ('d&b audiotechnik',        'https://www.dbaudio.com'),
    ('L-Acoustics',             'https://www.l-acoustics.com'),
    ('JBL Professional',        'https://jblpro.com'),
    ('Yamaha',                  'https://usa.yamaha.com'),
    ('QSC',                     'https://www.qsc.com'),
    ('Crown International',     'https://www.crownaudio.com'),
    ('Martin by Harman',        'https://www.martin.com'),
    ('Robe Lighting',           'https://www.robe.cz'),
    ('Chauvet Professional',    'https://www.chauvetprofessional.com'),
    ('ETC',                     'https://www.etcconnect.com'),
    ('GLP',                     'https://www.glp.de'),
    ('Ayrton',                  'https://www.ayrton.eu'),
    ('Claypaky',                'https://www.claypaky.it'),
    ('Elation Professional',    'https://www.elationlighting.com'),
    ('DiGiCo',                  'https://www.digico.biz'),
    ('Midas',                   'https://www.midasconsoles.com'),
    ('Allen & Heath',           'https://www.allen-heath.com'),
    ('Roland',                  'https://www.roland.com'),
    ('Blackmagic Design',       'https://www.blackmagicdesign.com'),
    ('Panasonic',               'https://www.panasonic.com'),
    ('Sony',                    'https://pro.sony'),
    ('Christie',                'https://www.christiedigital.com'),
    ('Barco',                   'https://www.barco.com'),
    ('Prolyte Group',           'https://www.prolyte.com'),
    ('Global Truss',            'https://www.global-truss.com'),
    ('Neutrik',                 'https://www.neutrik.com'),
    ('Amphenol',                'https://www.amphenol.com'),
    ('Obsidian Control Systems','https://www.obsidiancontrol.com'),
    ('MA Lighting',             'https://www.malighting.com'),
    ('Avolites',                'https://www.avolites.com')
) AS v(name, website)
WHERE NOT EXISTS (
    SELECT 1 FROM manufacturer m WHERE m.name = v.name
);

-- Default brands
INSERT INTO brands (name, manufacturerid) VALUES
('Shure',              (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Shure')),
('Sennheiser',         (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Sennheiser')),
('Neumann',            (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Sennheiser')),
('d&b audiotechnik',   (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'd&b audiotechnik')),
('L-Acoustics',        (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'L-Acoustics')),
('JBL Professional',   (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'JBL Professional')),
('JBL',                (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'JBL Professional')),
('Yamaha',             (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Yamaha')),
('Nexo',               (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Yamaha')),
('QSC',                (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'QSC')),
('Crown',              (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Crown International')),
('Martin',             (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Martin by Harman')),
('Robe',               (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Robe Lighting')),
('Chauvet Professional',(SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Chauvet Professional')),
('Chauvet DJ',         (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Chauvet Professional')),
('ETC',                (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'ETC')),
('GLP',                (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'GLP')),
('Ayrton',             (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Ayrton')),
('Claypaky',           (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Claypaky')),
('Elation',            (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Elation Professional')),
('DiGiCo',             (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'DiGiCo')),
('Midas',              (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Midas')),
('Allen & Heath',      (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Allen & Heath')),
('Roland',             (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Roland')),
('BOSS',               (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Roland')),
('Blackmagic Design',  (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Blackmagic Design')),
('ATEM',               (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Blackmagic Design')),
('Panasonic',          (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Panasonic')),
('Sony',               (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Sony')),
('Christie',           (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Christie')),
('Barco',              (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Barco')),
('Prolyte',            (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Prolyte Group')),
('Global Truss',       (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Global Truss')),
('Neutrik',            (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Neutrik')),
('Amphenol',           (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Amphenol')),
('Onyx',               (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Obsidian Control Systems')),
('grandMA',            (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'MA Lighting')),
('Avolites',           (SELECT MIN(manufacturerid) FROM manufacturer WHERE name = 'Avolites'))
ON CONFLICT DO NOTHING;
