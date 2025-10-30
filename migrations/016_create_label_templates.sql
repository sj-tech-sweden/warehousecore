-- Migration 016: Create label_templates table for barcode/QR label designer

CREATE TABLE IF NOT EXISTS label_templates (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    width DECIMAL(10,2) NOT NULL COMMENT 'Width in millimeters',
    height DECIMAL(10,2) NOT NULL COMMENT 'Height in millimeters',
    template_json LONGTEXT NOT NULL COMMENT 'JSON array of label elements',
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_is_default (is_default),
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default template for device labels
INSERT INTO label_templates (name, description, width, height, template_json, is_default) VALUES
(
    'Device Label - Standard 62x29mm',
    'Standard device label with QR code and product name',
    62,
    29,
    '[
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
    ]',
    TRUE
),
(
    'Device Label - Large 100x50mm',
    'Large device label with barcode and extended information',
    100,
    50,
    '[
        {
            "type": "barcode",
            "x": 5,
            "y": 5,
            "width": 90,
            "height": 20,
            "rotation": 0,
            "content": "device_id",
            "style": {
                "format": "code128"
            }
        },
        {
            "type": "text",
            "x": 5,
            "y": 28,
            "width": 90,
            "height": 8,
            "rotation": 0,
            "content": "product",
            "style": {
                "font_size": 14,
                "font_weight": "bold",
                "font_family": "Arial",
                "color": "#000000",
                "alignment": "center"
            }
        },
        {
            "type": "text",
            "x": 5,
            "y": 38,
            "width": 90,
            "height": 6,
            "rotation": 0,
            "content": "category",
            "style": {
                "font_size": 10,
                "font_weight": "normal",
                "font_family": "Arial",
                "color": "#666666",
                "alignment": "center"
            }
        }
    ]',
    FALSE
);
