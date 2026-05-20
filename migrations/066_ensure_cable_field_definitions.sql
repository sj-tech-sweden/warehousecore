-- Ensure cable-related product field definitions exist (idempotent)
DO $$
BEGIN
    -- Create definitions if missing
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_field_definitions') THEN
        RAISE NOTICE 'product_field_definitions table not found; skipping cable field definitions';
        RETURN;
    END IF;

    INSERT INTO product_field_definitions (name, label, field_type, options, unit, sort_order, is_required)
    SELECT v.name, v.label, v.field_type, v.options, v.unit, v.sort_order, v.is_required
    FROM (VALUES
        ('connector_1','Connector 1','select',NULL,NULL,1,FALSE),
        ('connector_2','Connector 2','select',NULL,NULL,2,FALSE),
        ('cable_type','Cable Type','select',NULL,NULL,3,FALSE),
        ('cable_length','Length','number',NULL,'m',4,FALSE),
        ('cable_mm2','Cross-section','number',NULL,'mm²',5,FALSE)
    ) AS v(name,label,field_type,options,unit,sort_order,is_required)
    WHERE NOT EXISTS (SELECT 1 FROM product_field_definitions pfd WHERE pfd.name = v.name)
    ;

    -- Populate select options from cable_connectors and cable_types if available
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'cable_connectors') THEN
        UPDATE product_field_definitions
        SET options = (SELECT COALESCE(json_agg(name ORDER BY name), '[]'::json)::text FROM cable_connectors)
        WHERE name IN ('connector_1','connector_2');
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'cable_types') THEN
        UPDATE product_field_definitions
        SET options = (SELECT COALESCE(json_agg(name ORDER BY name), '[]'::json)::text FROM cable_types)
        WHERE name = 'cable_type';
    END IF;

END$$;
