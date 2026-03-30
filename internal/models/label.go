package models

import (
	"time"
)

// LabelTemplate represents a saved label design
type LabelTemplate struct {
	ID           int       `json:"id" gorm:"primaryKey;column:id"`
	Name         string    `json:"name" gorm:"column:name;size:255;not null"`
	Description  string    `json:"description" gorm:"column:description;type:text"`
	Width        float64   `json:"width" gorm:"column:width_mm;not null"`                               // in mm (DB column: width_mm)
	Height       float64   `json:"height" gorm:"column:height_mm;not null"`                             // in mm (DB column: height_mm)
	TemplateJSON string    `json:"template_json" gorm:"column:template_content;type:longtext;not null"` // JSON with elements (DB column: template_content)
	IsDefault    bool      `json:"is_default" gorm:"column:is_default;default:false"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (LabelTemplate) TableName() string {
	return "label_templates"
}

// LabelElement represents an element in a label design (stored in TemplateJSON)
type LabelElement struct {
	Type      string            `json:"type"` // "barcode", "qrcode", "text", "image"
	X         float64           `json:"x"`
	Y         float64           `json:"y"`
	Width     float64           `json:"width"`
	Height    float64           `json:"height"`
	Rotation  float64           `json:"rotation"`
	Content   string            `json:"content"` // field name or static text
	Style     LabelElementStyle `json:"style"`
	ImageData string            `json:"image_data,omitempty"` // For static images
}

// LabelElementStyle defines styling for label elements
type LabelElementStyle struct {
	FontSize   int    `json:"font_size"`
	FontWeight string `json:"font_weight"`
	FontFamily string `json:"font_family"`
	Color      string `json:"color"`
	Alignment  string `json:"alignment"`
	Format     string `json:"format"` // For barcodes: "code128", "qr", "ean13"
}
