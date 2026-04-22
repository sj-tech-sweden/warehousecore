package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/chromedp/chromedp"
	"github.com/skip2/go-qrcode"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

type LabelService struct {
	// LabelsDir overrides the default labels directory. When empty, defaults to
	// "./web/dist/labels". Exposed for testing.
	LabelsDir string
}

func NewLabelService() *LabelService {
	log.Printf("[LABEL INIT] Label service initialized (using headless browser rendering)")
	return &LabelService{}
}

// GenerateQRCode generates a QR code and returns it as base64-encoded PNG
func (s *LabelService) GenerateQRCode(content string, size int) (string, error) {
	if size == 0 {
		size = 256 // default size
	}

	// Generate QR code
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Convert to PNG bytes
	pngBytes, err := qr.PNG(size)
	if err != nil {
		return "", fmt.Errorf("failed to convert QR code to PNG: %w", err)
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(pngBytes)
	return fmt.Sprintf("data:image/png;base64,%s", encoded), nil
}

// GenerateBarcode generates a Code128 barcode and returns it as base64-encoded PNG
func (s *LabelService) GenerateBarcode(content string, width, height int) (string, error) {
	if width == 0 {
		width = 300
	}
	if height == 0 {
		height = 100
	}

	// Ensure minimum dimensions for barcode library (needs at least 123px width)
	if width < 123 {
		width = 123
	}
	if height < 1 {
		height = 1
	}

	// Generate Code128 barcode
	bc, err := code128.Encode(content)
	if err != nil {
		return "", fmt.Errorf("failed to generate barcode: %w", err)
	}

	// Scale barcode
	scaled, err := barcode.Scale(bc, width, height)
	if err != nil {
		return "", fmt.Errorf("failed to scale barcode: %w", err)
	}

	// Convert to PNG
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, scaled); err != nil {
		return "", fmt.Errorf("failed to encode barcode to PNG: %w", err)
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:image/png;base64,%s", encoded), nil
}

// GetAllTemplates retrieves all label templates
func (s *LabelService) GetAllTemplates() ([]models.LabelTemplate, error) {
	db := repository.GetDB()
	var templates []models.LabelTemplate

	if err := db.Order("is_default DESC, name ASC").Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch templates: %w", err)
	}

	return templates, nil
}

// GetTemplateByID retrieves a specific template
func (s *LabelService) GetTemplateByID(id int) (*models.LabelTemplate, error) {
	db := repository.GetDB()
	var template models.LabelTemplate

	if err := db.Where("id = ?", id).First(&template).Error; err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	return &template, nil
}

// CreateTemplate creates a new label template
func (s *LabelService) CreateTemplate(template *models.LabelTemplate) error {
	db := repository.GetDB()

	// Validate template JSON
	var elements []models.LabelElement
	if err := json.Unmarshal([]byte(template.TemplateJSON), &elements); err != nil {
		return fmt.Errorf("invalid template JSON: %w", err)
	}

	// If this is set as default, unset other defaults
	if template.IsDefault {
		if err := db.Model(&models.LabelTemplate{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	if err := db.Create(template).Error; err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// UpdateTemplate updates an existing label template
func (s *LabelService) UpdateTemplate(id int, updates map[string]interface{}) error {
	db := repository.GetDB()

	// Validate template JSON if provided (API uses key "template_json")
	if templateJSON, ok := updates["template_json"].(string); ok {
		var elements []models.LabelElement
		if err := json.Unmarshal([]byte(templateJSON), &elements); err != nil {
			return fmt.Errorf("invalid template JSON: %w", err)
		}
		// Translate API key to DB column name expected by the schema
		updates["template_content"] = templateJSON
		delete(updates, "template_json")
	}

	// Translate width/height keys from API to DB column names if present
	if w, ok := updates["width"]; ok {
		updates["width_mm"] = w
		delete(updates, "width")
	}
	if h, ok := updates["height"]; ok {
		updates["height_mm"] = h
		delete(updates, "height")
	}

	// If setting as default, unset other defaults
	if isDefault, ok := updates["is_default"].(bool); ok && isDefault {
		if err := db.Model(&models.LabelTemplate{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	if err := db.Model(&models.LabelTemplate{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	return nil
}

// DeleteTemplate deletes a label template
func (s *LabelService) DeleteTemplate(id int) error {
	db := repository.GetDB()

	result := db.Where("id = ?", id).Delete(&models.LabelTemplate{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete template: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("template not found")
	}

	return nil
}

// GenerateLabelForDevice generates a complete label for a device
func (s *LabelService) GenerateLabelForDevice(deviceID string, templateID int) (map[string]interface{}, error) {
	// Get template
	template, err := s.GetTemplateByID(templateID)
	if err != nil {
		return nil, err
	}

	// Parse template elements
	var elements []models.LabelElement
	if strings.TrimSpace(template.TemplateJSON) == "" {
		return nil, fmt.Errorf("template JSON is empty for template ID %d", template.ID)
	}
	if err := json.Unmarshal([]byte(template.TemplateJSON), &elements); err != nil {
		return nil, fmt.Errorf("invalid template JSON for template ID %d: %w", template.ID, err)
	}

	// Get device data with all related fields
	db := repository.GetDB()
	var device struct {
		DeviceID            string  `json:"device_id"`
		SerialNumber        string  `json:"serial_number"`
		Barcode             string  `json:"barcode"`
		RFID                string  `json:"rfid"`
		QRCode              string  `json:"qr_code"`
		Status              string  `json:"status"`
		ConditionRating     float64 `json:"condition_rating"`
		UsageHours          float64 `json:"usage_hours"`
		PurchaseDate        string  `json:"purchase_date"`
		Notes               string  `json:"notes"`
		ZoneName            string  `json:"zone_name"`
		ZoneCode            string  `json:"zone_code"`
		CaseName            string  `json:"case_name"`
		ProductName         string  `json:"product_name"`
		ProductDescription  string  `json:"product_description"`
		Subcategory         string  `json:"subcategory"`
		Category            string  `json:"category"`
		ManufacturerName    string  `json:"manufacturer_name"`
		BrandName           string  `json:"brand_name"`
		ProductWeight       float64 `json:"product_weight"`
		ProductWidth        float64 `json:"product_width"`
		ProductHeight       float64 `json:"product_height"`
		ProductDepth        float64 `json:"product_depth"`
		MaintenanceInterval int     `json:"maintenance_interval"`
		PowerConsumption    float64 `json:"power_consumption"`
	}

	query := `
		SELECT
			d.deviceID as device_id,
			COALESCE(d.serialnumber, '') as serial_number,
			COALESCE(d.barcode, '') as barcode,
			COALESCE(d.rfid, '') as rfid,
			COALESCE(d.qr_code, '') as qr_code,
			COALESCE(d.status, '') as status,
			COALESCE(d.condition_rating, 0) as condition_rating,
			COALESCE(d.usage_hours, 0) as usage_hours,
			COALESCE(TO_CHAR(d.purchaseDate, 'YYYY-MM-DD'), '') as purchase_date,
			COALESCE(d.notes, '') as notes,
			COALESCE(z.name, '') as zone_name,
			COALESCE(z.code, '') as zone_code,
			COALESCE(ca.name, '') as case_name,
			COALESCE(p.name, '') as product_name,
			COALESCE(p.description, '') as product_description,
			COALESCE(sb.name, '') as subcategory,
			COALESCE(c.name, '') as category,
			COALESCE(m.name, '') as manufacturer_name,
			COALESCE(b.name, '') as brand_name,
			COALESCE(p.weight, 0) as product_weight,
			COALESCE(p.width, 0) as product_width,
			COALESCE(p.height, 0) as product_height,
			COALESCE(p.depth, 0) as product_depth,
			COALESCE(p.maintenanceInterval, 0) as maintenance_interval,
			COALESCE(p.powerconsumption, 0) as power_consumption
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN subbiercategories sb ON p.subbiercategoryID = sb.subbiercategoryID
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		LEFT JOIN manufacturer m ON p.manufacturerID = m.manufacturerID
		LEFT JOIN brands b ON p.brandID = b.brandID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		LEFT JOIN devicescases dc ON d.deviceID = dc.deviceID
		LEFT JOIN cases ca ON dc.caseID = ca.caseID
		WHERE d.deviceID = $1
	`

	if err := db.Raw(query, deviceID).Scan(&device).Error; err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// Build a field map for easy lookup
	fields := map[string]string{
		"device_id":           device.DeviceID,
		"serial_number":       device.SerialNumber,
		"barcode":             device.Barcode,
		"rfid":                device.RFID,
		"qr_code":             device.QRCode,
		"status":              device.Status,
		"zone_name":           device.ZoneName,
		"zone_code":           device.ZoneCode,
		"case_name":           device.CaseName,
		"notes":               device.Notes,
		"purchase_date":       device.PurchaseDate,
		"product_name":        device.ProductName,
		"product_description": device.ProductDescription,
		"subcategory":         device.Subcategory,
		"category":            device.Category,
		"manufacturer":        device.ManufacturerName,
		"manufacturer_name":   device.ManufacturerName,
		"brand":               device.BrandName,
		"brand_name":          device.BrandName,
	}
	if device.ConditionRating > 0 {
		fields["condition_rating"] = fmt.Sprintf("%.1f", device.ConditionRating)
	} else {
		fields["condition_rating"] = ""
	}
	if device.UsageHours > 0 {
		fields["usage_hours"] = fmt.Sprintf("%.0f h", device.UsageHours)
	} else {
		fields["usage_hours"] = ""
	}
	if device.ProductWeight > 0 {
		fields["product_weight"] = fmt.Sprintf("%.2f kg", device.ProductWeight)
	} else {
		fields["product_weight"] = ""
	}
	if device.ProductWidth > 0 && device.ProductHeight > 0 && device.ProductDepth > 0 {
		fields["product_dimensions"] = fmt.Sprintf("%.1fx%.1fx%.1f cm", device.ProductWidth, device.ProductHeight, device.ProductDepth)
	} else {
		fields["product_dimensions"] = ""
	}
	if device.MaintenanceInterval > 0 {
		fields["maintenance_interval"] = fmt.Sprintf("%d days", device.MaintenanceInterval)
	} else {
		fields["maintenance_interval"] = ""
	}
	if device.PowerConsumption > 0 {
		fields["power_consumption"] = fmt.Sprintf("%.0f W", device.PowerConsumption)
	} else {
		fields["power_consumption"] = ""
	}

	// Support legacy aliases
	fields["device_name"] = device.DeviceID
	fields["name"] = device.ProductName
	fields["product"] = device.Subcategory

	// Process elements and generate barcodes/QR codes
	processedElements := make([]map[string]interface{}, 0, len(elements))
	for _, elem := range elements {
		processed := map[string]interface{}{
			"type":     elem.Type,
			"x":        elem.X,
			"y":        elem.Y,
			"width":    elem.Width,
			"height":   elem.Height,
			"rotation": elem.Rotation,
			"style":    elem.Style,
		}

		// Resolve content from field map (supports all fields)
		content := elem.Content
		if resolved, ok := fields[elem.Content]; ok {
			content = resolved
		}

		processed["content"] = content

		// Generate barcode/QR code if needed, or copy static image data
		if elem.Type == "qrcode" {
			// Convert mm to pixels at 300 DPI (300 pixels per inch, 25.4mm per inch)
			// pixels = mm * 300 / 25.4 ≈ mm * 11.8
			sizePixels := int(elem.Width * 11.8)
			if sizePixels < 100 {
				sizePixels = 100
			}
			qrData, err := s.GenerateQRCode(content, sizePixels)
			if err != nil {
				return nil, err
			}
			processed["image_data"] = qrData
		} else if elem.Type == "barcode" {
			// Convert mm to pixels at 300 DPI
			widthPixels := int(elem.Width * 11.8)
			heightPixels := int(elem.Height * 11.8)
			if widthPixels < 123 {
				widthPixels = 123 // Minimum for Code128
			}
			if heightPixels < 50 {
				heightPixels = 50
			}
			barcodeData, err := s.GenerateBarcode(content, widthPixels, heightPixels)
			if err != nil {
				return nil, err
			}
			processed["image_data"] = barcodeData
		} else if elem.Type == "image" && elem.ImageData != "" {
			// Copy static image data from template
			processed["image_data"] = elem.ImageData
		}

		processedElements = append(processedElements, processed)
	}

	return map[string]interface{}{
		"template": template,
		"elements": processedElements,
		"device":   device,
	}, nil
}

// GenerateLabelForCase generates a complete label for a case
func (s *LabelService) GenerateLabelForCase(caseID int, templateID int) (map[string]interface{}, error) {
	// Get template
	template, err := s.GetTemplateByID(templateID)
	if err != nil {
		return nil, err
	}

	// Parse template elements
	var elements []models.LabelElement
	if err := json.Unmarshal([]byte(template.TemplateJSON), &elements); err != nil {
		return nil, fmt.Errorf("invalid template JSON: %w", err)
	}

	// Get case data with zone information
	db := repository.GetDB()
	var caseData struct {
		CaseID      int     `json:"case_id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Barcode     string  `json:"barcode"`
		RFIDTag     string  `json:"rfid_tag"`
		Width       float64 `json:"width"`
		Height      float64 `json:"height"`
		Depth       float64 `json:"depth"`
		Weight      float64 `json:"weight"`
		Status      string  `json:"status"`
		ZoneName    string  `json:"zone_name"`
		ZoneCode    string  `json:"zone_code"`
	}

	query := `
		SELECT
			c.caseID as case_id,
			c.name,
			COALESCE(c.description, '') as description,
			COALESCE(c.barcode, '') as barcode,
			COALESCE(c.rfid_tag, '') as rfid_tag,
			COALESCE(c.width, 0) as width,
			COALESCE(c.height, 0) as height,
			COALESCE(c.depth, 0) as depth,
			COALESCE(c.weight, 0) as weight,
			c.status,
			COALESCE(z.name, '') as zone_name,
			COALESCE(z.code, '') as zone_code
		FROM cases c
		LEFT JOIN storage_zones z ON c.zone_id = z.zone_id
		WHERE c.caseID = $1
	`

	if err := db.Raw(query, caseID).Scan(&caseData).Error; err != nil {
		return nil, fmt.Errorf("case not found: %w", err)
	}

	caseIDStr := fmt.Sprintf("CASE-%d", caseData.CaseID)
	barcodeVal := caseData.Barcode
	if barcodeVal == "" {
		barcodeVal = caseIDStr
	}

	// Build field map for easy lookup
	fields := map[string]string{
		"case_id":      caseIDStr,
		"device_id":    caseIDStr, // compatibility alias
		"name":         caseData.Name,
		"product_name": caseData.Name, // compatibility alias
		"description":  caseData.Description,
		"barcode":      barcodeVal,
		"rfid_tag":     caseData.RFIDTag,
		"status":       caseData.Status,
		"zone_name":    caseData.ZoneName,
		"zone_code":    caseData.ZoneCode,
	}
	if caseData.Width > 0 && caseData.Height > 0 && caseData.Depth > 0 {
		fields["dimensions"] = fmt.Sprintf("%.1fx%.1fx%.1f cm", caseData.Width, caseData.Height, caseData.Depth)
	} else {
		fields["dimensions"] = ""
	}
	if caseData.Weight > 0 {
		fields["weight"] = fmt.Sprintf("%.1f kg", caseData.Weight)
	} else {
		fields["weight"] = ""
	}

	// Process elements and generate barcodes/QR codes
	processedElements := make([]map[string]interface{}, 0, len(elements))
	for _, elem := range elements {
		processed := map[string]interface{}{
			"type":     elem.Type,
			"x":        elem.X,
			"y":        elem.Y,
			"width":    elem.Width,
			"height":   elem.Height,
			"rotation": elem.Rotation,
			"style":    elem.Style,
		}

		// Resolve content from field map (supports all fields)
		content := elem.Content
		if resolved, ok := fields[elem.Content]; ok {
			content = resolved
		}

		processed["content"] = content

		// Generate barcode/QR code if needed, or copy static image data
		if elem.Type == "qrcode" {
			// Convert mm to pixels at 300 DPI (300 pixels per inch, 25.4mm per inch)
			// pixels = mm * 300 / 25.4 ≈ mm * 11.8
			sizePixels := int(elem.Width * 11.8)
			if sizePixels < 100 {
				sizePixels = 100
			}
			qrData, err := s.GenerateQRCode(content, sizePixels)
			if err != nil {
				return nil, err
			}
			processed["image_data"] = qrData
		} else if elem.Type == "barcode" {
			// Convert mm to pixels at 300 DPI
			widthPixels := int(elem.Width * 11.8)
			heightPixels := int(elem.Height * 11.8)
			if widthPixels < 123 {
				widthPixels = 123 // Minimum for Code128
			}
			if heightPixels < 50 {
				heightPixels = 50
			}
			barcodeData, err := s.GenerateBarcode(content, widthPixels, heightPixels)
			if err != nil {
				return nil, err
			}
			processed["image_data"] = barcodeData
		} else if elem.Type == "image" && elem.ImageData != "" {
			// Copy static image data from template
			processed["image_data"] = elem.ImageData
		}

		processedElements = append(processedElements, processed)
	}

	return map[string]interface{}{
		"template": template,
		"elements": processedElements,
		"case":     caseData,
	}, nil
}

// GenerateLabelForZone generates a complete label for a zone
func (s *LabelService) GenerateLabelForZone(zoneID int64, templateID int) (map[string]interface{}, error) {
	// Get template
	template, err := s.GetTemplateByID(templateID)
	if err != nil {
		return nil, err
	}

	// Parse template elements
	var elements []models.LabelElement
	if err := json.Unmarshal([]byte(template.TemplateJSON), &elements); err != nil {
		return nil, fmt.Errorf("invalid template JSON: %w", err)
	}

	// Get zone data with parent zone info
	db := repository.GetDB()
	var zoneData struct {
		ZoneID      int64  `json:"zone_id"`
		Code        string `json:"code"`
		Barcode     string `json:"barcode"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Location    string `json:"location"`
		Capacity    int64  `json:"capacity"`
		ParentName  string `json:"parent_name"`
		ParentCode  string `json:"parent_code"`
	}

	query := `
		SELECT
			z.zone_id,
			z.code,
			COALESCE(z.barcode, '') as barcode,
			z.name,
			z.type,
			COALESCE(z.description, '') as description,
			COALESCE(z.location, '') as location,
			COALESCE(z.capacity, 0) as capacity,
			COALESCE(pz.name, '') as parent_name,
			COALESCE(pz.code, '') as parent_code
		FROM storage_zones z
		LEFT JOIN storage_zones pz ON z.parent_zone_id = pz.zone_id
		WHERE z.zone_id = $1
	`

	if err := db.Raw(query, zoneID).Scan(&zoneData).Error; err != nil {
		return nil, fmt.Errorf("zone not found: %w", err)
	}

	barcodeVal := zoneData.Barcode
	if barcodeVal == "" {
		barcodeVal = zoneData.Code
	}

	// Build field map for easy lookup
	fields := map[string]string{
		"zone_id":     fmt.Sprintf("%d", zoneData.ZoneID),
		"code":        zoneData.Code,
		"zone_code":   zoneData.Code,
		"name":        zoneData.Name,
		"zone_name":   zoneData.Name,
		"type":        zoneData.Type,
		"zone_type":   zoneData.Type,
		"description": zoneData.Description,
		"location":    zoneData.Location,
		"barcode":     barcodeVal,
		"parent_name": zoneData.ParentName,
		"parent_code": zoneData.ParentCode,
	}
	if zoneData.Capacity > 0 {
		fields["capacity"] = fmt.Sprintf("%d", zoneData.Capacity)
	} else {
		fields["capacity"] = ""
	}

	// Process elements and generate barcodes/QR codes
	processedElements := make([]map[string]interface{}, 0, len(elements))
	for _, elem := range elements {
		processed := map[string]interface{}{
			"type":     elem.Type,
			"x":        elem.X,
			"y":        elem.Y,
			"width":    elem.Width,
			"height":   elem.Height,
			"rotation": elem.Rotation,
			"style":    elem.Style,
		}

		// Resolve content from field map (supports all fields)
		content := elem.Content
		if resolved, ok := fields[elem.Content]; ok {
			content = resolved
		}

		processed["content"] = content

		// Generate barcode/QR code if needed, or copy static image data
		if elem.Type == "qrcode" {
			sizePixels := int(elem.Width * 11.8)
			if sizePixels < 100 {
				sizePixels = 100
			}
			qrData, err := s.GenerateQRCode(content, sizePixels)
			if err != nil {
				return nil, err
			}
			processed["image_data"] = qrData
		} else if elem.Type == "barcode" {
			widthPixels := int(elem.Width * 11.8)
			heightPixels := int(elem.Height * 11.8)
			if widthPixels < 123 {
				widthPixels = 123
			}
			if heightPixels < 50 {
				heightPixels = 50
			}
			barcodeData, err := s.GenerateBarcode(content, widthPixels, heightPixels)
			if err != nil {
				return nil, err
			}
			processed["image_data"] = barcodeData
		} else if elem.Type == "image" && elem.ImageData != "" {
			processed["image_data"] = elem.ImageData
		}

		processedElements = append(processedElements, processed)
	}

	return map[string]interface{}{
		"template": template,
		"elements": processedElements,
		"zone":     zoneData,
	}, nil
}

// SaveLabelImage saves a base64-encoded label image to disk and updates the device record
func (s *LabelService) SaveLabelImage(deviceID string, base64Image string) (string, error) {
	// Validate deviceID to prevent path traversal and filename collisions —
	// only allow alphanumeric characters, dash, and underscore.
	safeDeviceID := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, deviceID)
	if safeDeviceID == "" || safeDeviceID != deviceID {
		return "", fmt.Errorf("device ID must contain only alphanumeric characters, dashes, or underscores")
	}

	// Check DB availability before writing to disk to avoid orphaned label
	// files when the DB update would fail.
	db := repository.GetDB()
	if db == nil {
		return "", fmt.Errorf("database connection is not available")
	}

	// Remove base64 prefix if present
	if len(base64Image) > 22 && base64Image[:22] == "data:image/png;base64," {
		base64Image = base64Image[22:]
	}

	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Create labels directory if it doesn't exist
	labelsDir := s.LabelsDir
	if labelsDir == "" {
		labelsDir = "./web/dist/labels"
	}
	if err := os.MkdirAll(labelsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create labels directory: %w", err)
	}

	// Save file
	filename := fmt.Sprintf("%s_label.png", safeDeviceID)

	// Resolve the labels directory (constant path) to handle symlinks
	resolvedLabelsDir, err := filepath.EvalSymlinks(labelsDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve labels directory: %w", err)
	}

	// Build target path from resolved constant directory + sanitized filename
	resolvedFilePath := filepath.Join(resolvedLabelsDir, filename)

	// Verify the resolved path stays within the labels directory
	relPath, err := filepath.Rel(resolvedLabelsDir, resolvedFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to validate file path: %w", err)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid file path: outside allowed directory")
	}

	// Refuse to write if the target path is an existing symlink file —
	// prevents an attacker from redirecting writes outside the labels directory
	if info, err := os.Lstat(resolvedFilePath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("invalid file path: target is a symlink")
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check target file: %w", err)
	}

	// Write to a temporary file created inside the resolved labels directory
	// and atomically rename it into place. This avoids the TOCTOU race between
	// checking the destination path and writing to it.
	tempFile, err := os.CreateTemp(resolvedLabelsDir, ".label.*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary label file: %w", err)
	}
	tempFilePath := tempFile.Name()
	cleanupTempFile := true
	defer func() {
		if cleanupTempFile {
			_ = os.Remove(tempFilePath)
		}
	}()

	if err := tempFile.Chmod(0644); err != nil {
		tempFile.Close()
		return "", fmt.Errorf("failed to set temporary label file permissions: %w", err)
	}

	if _, err := tempFile.Write(imageData); err != nil {
		tempFile.Close()
		return "", fmt.Errorf("failed to write temporary label file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temporary label file: %w", err)
	}

	// Before renaming, check if the destination already exists.
	// On Windows, os.Rename fails if the destination exists, so we need to
	// handle it first. Also reject symlinks and directories at the destination.
	if destInfo, err := os.Lstat(resolvedFilePath); err == nil {
		if destInfo.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to replace symlink label file: %s", resolvedFilePath)
		}
		if destInfo.IsDir() {
			return "", fmt.Errorf("refusing to replace directory with label file: %s", resolvedFilePath)
		}
		// Rename existing file to a unique backup so we can restore it if the
		// subsequent os.Rename fails (e.g. permissions, AV, disk full).
		backupPath := resolvedFilePath + fmt.Sprintf(".bak.%d", time.Now().UnixNano())
		if err := os.Rename(resolvedFilePath, backupPath); err != nil {
			return "", fmt.Errorf("failed to back up existing label file: %w", err)
		}
		if err := os.Rename(tempFilePath, resolvedFilePath); err != nil {
			// Restore the backup so the previous label is not lost.
			_ = os.Rename(backupPath, resolvedFilePath)
			return "", fmt.Errorf("failed to move label file into place: %w", err)
		}
		cleanupTempFile = false

		// Update device record with label path; restore backup on DB failure.
		labelPath := fmt.Sprintf("/labels/%s", filename)
		result := db.Exec("UPDATE devices SET label_path = $1 WHERE deviceID = $2", labelPath, deviceID)
		if result.Error != nil {
			// Restore the previous label file so disk+DB stay consistent.
			_ = os.Remove(resolvedFilePath)
			_ = os.Rename(backupPath, resolvedFilePath)
			return "", fmt.Errorf("failed to update device label path: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			// Device doesn't exist — restore backup and remove orphaned label.
			_ = os.Remove(resolvedFilePath)
			_ = os.Rename(backupPath, resolvedFilePath)
			return "", fmt.Errorf("device not found: %s", deviceID)
		}
		_ = os.Remove(backupPath)
		return labelPath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to inspect existing label file: %w", err)
	} else {
		if err := os.Rename(tempFilePath, resolvedFilePath); err != nil {
			return "", fmt.Errorf("failed to move label file into place: %w", err)
		}
	}
	cleanupTempFile = false
	// Update device record with label path
	labelPath := fmt.Sprintf("/labels/%s", filename)
	result := db.Exec("UPDATE devices SET label_path = $1 WHERE deviceID = $2", labelPath, deviceID)
	if result.Error != nil {
		// Remove orphaned label file so disk+DB stay consistent.
		_ = os.Remove(resolvedFilePath)
		return "", fmt.Errorf("failed to update device label path: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		// Device doesn't exist — remove orphaned label file.
		_ = os.Remove(resolvedFilePath)
		return "", fmt.Errorf("device not found: %s", deviceID)
	}

	return labelPath, nil
}

// SaveCaseLabelImage saves a base64-encoded label image to disk and updates the case record
func (s *LabelService) SaveCaseLabelImage(caseID int, base64Image string) (string, error) {
	// Remove base64 prefix if present
	if len(base64Image) > 22 && base64Image[:22] == "data:image/png;base64," {
		base64Image = base64Image[22:]
	}

	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Create labels/cases directory if it doesn't exist
	labelsDir := "./web/dist/labels/cases"
	if err := os.MkdirAll(labelsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create labels directory: %w", err)
	}

	// Save file
	filename := fmt.Sprintf("CASE-%d_label.png", caseID)
	filePath := filepath.Join(labelsDir, filename)

	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to write label file: %w", err)
	}

	// Update case record with label path
	labelPath := fmt.Sprintf("/labels/cases/%s", filename)
	db := repository.GetDB()
	result := db.Exec("UPDATE cases SET label_path = $1 WHERE caseID = $2", labelPath, caseID)
	if result.Error != nil {
		return "", fmt.Errorf("failed to update case label path: %w", result.Error)
	}

	return labelPath, nil
}

// SaveZoneLabelImage saves a base64-encoded label image to disk and updates the zone record
func (s *LabelService) SaveZoneLabelImage(zoneID int64, base64Image string) (string, error) {
	// Remove base64 prefix if present
	if len(base64Image) > 22 && base64Image[:22] == "data:image/png;base64," {
		base64Image = base64Image[22:]
	}

	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Create labels/zones directory if it doesn't exist
	labelsDir := "./web/dist/labels/zones"
	if err := os.MkdirAll(labelsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create labels directory: %w", err)
	}

	// Save file
	filename := fmt.Sprintf("ZONE-%d_label.png", zoneID)
	filePath := filepath.Join(labelsDir, filename)

	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to write label file: %w", err)
	}

	// Update zone record with label path
	labelPath := fmt.Sprintf("/labels/zones/%s", filename)
	db := repository.GetDB()
	result := db.Exec("UPDATE storage_zones SET label_path = $1 WHERE zone_id = $2", labelPath, zoneID)
	if result.Error != nil {
		return "", fmt.Errorf("failed to update zone label path: %w", result.Error)
	}

	return labelPath, nil
}

// AutoGenerateDeviceLabel automatically generates a label for a new device using the default template
// This uses a headless Chrome browser to render labels identically to the Label Designer UI
func (s *LabelService) AutoGenerateDeviceLabel(deviceID string) error {
	// Get default template
	db := repository.GetDB()
	var template models.LabelTemplate
	if err := db.Where("is_default = ?", true).First(&template).Error; err != nil {
		// No default template, skip label generation silently
		log.Printf("[LABEL DEBUG] No default template found, skipping auto-generation for device: %s", deviceID)
		return nil
	}

	log.Printf("[LABEL DEBUG] === Generating label for device: %s ===", deviceID)
	log.Printf("[LABEL DEBUG] Template: %s (ID: %d)", template.Name, template.ID)
	log.Printf("[LABEL DEBUG] Label size: %.1fmm x %.1fmm", template.Width, template.Height)

	// Guard against empty or invalid templates coming from the database
	if strings.TrimSpace(template.TemplateJSON) == "" {
		log.Printf("[LABEL DEBUG] Template ID %d has empty TemplateJSON, skipping auto-generation for device: %s", template.ID, deviceID)
		return nil
	}

	if template.Width == 0 || template.Height == 0 {
		log.Printf("[LABEL DEBUG] Template ID %d has invalid dimensions (%.1f x %.1f), skipping auto-generation for device: %s", template.ID, template.Width, template.Height, deviceID)
		return nil
	}

	// Generate label data using the template (includes device data, barcodes, QR codes)
	labelData, err := s.GenerateLabelForDevice(deviceID, template.ID)
	if err != nil {
		return fmt.Errorf("failed to generate label data: %w", err)
	}

	// Convert label data to JSON for embedding in HTML
	labelDataJSON, err := json.Marshal(labelData)
	if err != nil {
		return fmt.Errorf("failed to marshal label data: %w", err)
	}

	// Load HTML template
	htmlTemplatePath := "./internal/services/label_template.html"
	htmlTemplate, err := ioutil.ReadFile(htmlTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to load HTML template: %w", err)
	}

	// Replace placeholder with actual label data
	htmlContent := strings.Replace(string(htmlTemplate), "{{LABEL_DATA_JSON}}", string(labelDataJSON), 1)

	// Render label using headless Chrome
	base64PNG, err := s.renderLabelWithHeadlessBrowser(htmlContent)
	if err != nil {
		return fmt.Errorf("failed to render label with headless browser: %w", err)
	}

	// Save the label
	base64String := fmt.Sprintf("data:image/png;base64,%s", base64PNG)
	_, err = s.SaveLabelImage(deviceID, base64String)
	if err != nil {
		return fmt.Errorf("failed to save label: %w", err)
	}

	log.Printf("[LABEL DEBUG] Label generated successfully for device: %s", deviceID)
	return nil
}

// renderLabelWithHeadlessBrowser uses chromedp to render the label HTML and capture as PNG
func (s *LabelService) renderLabelWithHeadlessBrowser(htmlContent string) (string, error) {
	// Create context with timeout (increased to 60s for Docker environments)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create chromedp context with options optimized for Docker
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.Flag("disable-dev-shm-usage", true), // Critical for Docker - prevents shared memory issues
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	// Disable verbose logging to reduce noise
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	// Variable to store the screenshot
	var buf []byte

	// Run chromedp tasks
	err := chromedp.Run(taskCtx,
		chromedp.Navigate("data:text/html,"+htmlContent),
		chromedp.WaitVisible("canvas", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Wait for rendering to complete
		chromedp.Evaluate(`document.getElementById('canvas').toDataURL('image/png')`, &buf),
	)

	if err != nil {
		return "", fmt.Errorf("chromedp execution failed: %w", err)
	}

	// The result from toDataURL is already a data URL string, we need to extract the base64 part
	dataURL := string(buf)
	if strings.HasPrefix(dataURL, "data:image/png;base64,") {
		return dataURL[22:], nil
	}

	return "", fmt.Errorf("unexpected data URL format: %s", dataURL[:50])
}
