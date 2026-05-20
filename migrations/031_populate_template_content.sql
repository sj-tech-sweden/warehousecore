-- Migration 031: Populate template_content for label_templates from default JSON
-- Idempotent: only updates rows where template_content is NULL/empty

BEGIN;

-- Update any templates that lack template_content with the default device label JSON
-- Ensure template_content column exists (JSONB) before updating
ALTER TABLE IF EXISTS label_templates ADD COLUMN IF NOT EXISTS template_content JSONB;

UPDATE label_templates
SET template_content = $$[
        {
            "type": "qrcode",
            "x": 2,
            "y": 2,
            "width": 25,
            "height": 25,
            "rotation": 0,
            "content": "device_id",
            "style": {
                "format": "qr"
            }
        },
        {
            "type": "text",
            "x": 30,
            "y": 5,
            "width": 30,
            "height": 8,
            "rotation": 0,
            "content": "product",
            "style": {
                "font_size": 12,
                "font_weight": "bold",
                "font_family": "Arial",
                "color": "#000000",
                "alignment": "left"
            }
        },
        {
            "type": "text",
            "x": 30,
            "y": 15,
            "width": 30,
            "height": 6,
            "rotation": 0,
            "content": "device_id",
            "style": {
                "font_size": 10,
                "font_weight": "normal",
                "font_family": "Arial",
                "color": "#666666",
                "alignment": "left"
            }
        }
    ]$$::jsonb
    WHERE template_content IS NULL OR TRIM(template_content::text) = '';

-- Ensure width_mm/height_mm are populated from legacy width/height columns if present
ALTER TABLE label_templates ADD COLUMN IF NOT EXISTS width_mm numeric;
ALTER TABLE label_templates ADD COLUMN IF NOT EXISTS height_mm numeric;
UPDATE label_templates SET width_mm = width WHERE (width_mm IS NULL OR width_mm = 0) AND (width IS NOT NULL);
UPDATE label_templates SET height_mm = height WHERE (height_mm IS NULL OR height_mm = 0) AND (height IS NOT NULL);

COMMIT;
