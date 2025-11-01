package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/skip2/go-qrcode"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

type LabelService struct{}

func NewLabelService() *LabelService {
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

	// Validate template JSON if provided
	if templateJSON, ok := updates["template_json"].(string); ok {
		var elements []models.LabelElement
		if err := json.Unmarshal([]byte(templateJSON), &elements); err != nil {
			return fmt.Errorf("invalid template JSON: %w", err)
		}
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
	if err := json.Unmarshal([]byte(template.TemplateJSON), &elements); err != nil {
		return nil, fmt.Errorf("invalid template JSON: %w", err)
	}

	// Get device data
	db := repository.GetDB()
	var device struct {
		DeviceID   string `json:"device_id"`
		DeviceName string `json:"device_name"`
		Product    string `json:"product"`
		Category   string `json:"category"`
	}

	query := `
		SELECT
			d.deviceID as device_id,
			p.name as device_name,
			sb.name as product,
			c.name as category
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN subbiercategories sb ON p.subbiercategoryID = sb.subbiercategoryID
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		WHERE d.deviceID = ?
	`

	if err := db.Raw(query, deviceID).Scan(&device).Error; err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
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

		// Resolve content from field names
		content := elem.Content
		switch elem.Content {
		case "device_id":
			content = device.DeviceID
		case "device_name":
			content = device.DeviceName
		case "product":
			content = device.Product
		case "category":
			content = device.Category
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

	// Get case data
	db := repository.GetDB()
	var caseData struct {
		CaseID      int     `json:"case_id"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
		Barcode     *string `json:"barcode"`
		RFIDTag     *string `json:"rfid_tag"`
		Width       *float64 `json:"width"`
		Height      *float64 `json:"height"`
		Depth       *float64 `json:"depth"`
		Weight      *float64 `json:"weight"`
		Status      string  `json:"status"`
		ZoneName    *string `json:"zone_name"`
	}

	query := `
		SELECT
			c.caseID as case_id,
			c.name,
			c.description,
			c.barcode,
			c.rfid_tag,
			c.width,
			c.height,
			c.depth,
			c.weight,
			c.status
		FROM cases c
		WHERE c.caseID = ?
	`

	if err := db.Raw(query, caseID).Scan(&caseData).Error; err != nil {
		return nil, fmt.Errorf("case not found: %w", err)
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

		// Resolve content from field names
		content := elem.Content
		switch elem.Content {
		case "case_id", "device_id": // Support both case_id and device_id (for compatibility with device templates)
			content = fmt.Sprintf("CASE-%d", caseData.CaseID)
		case "name", "product_name": // Support both name and product_name
			content = caseData.Name
		case "description":
			if caseData.Description != nil {
				content = *caseData.Description
			}
		case "barcode":
			if caseData.Barcode != nil {
				content = *caseData.Barcode
			} else {
				content = fmt.Sprintf("CASE-%d", caseData.CaseID) // fallback
			}
		case "rfid_tag":
			if caseData.RFIDTag != nil {
				content = *caseData.RFIDTag
			}
		case "dimensions":
			if caseData.Width != nil && caseData.Height != nil && caseData.Depth != nil {
				content = fmt.Sprintf("%.1fx%.1fx%.1f cm", *caseData.Width, *caseData.Height, *caseData.Depth)
			}
		case "weight":
			if caseData.Weight != nil {
				content = fmt.Sprintf("%.1f kg", *caseData.Weight)
			}
		case "zone_name":
			if caseData.ZoneName != nil {
				content = *caseData.ZoneName
			}
		case "status":
			content = caseData.Status
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

// SaveLabelImage saves a base64-encoded label image to disk and updates the device record
func (s *LabelService) SaveLabelImage(deviceID string, base64Image string) (string, error) {
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
	labelsDir := "./web/dist/labels"
	if err := os.MkdirAll(labelsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create labels directory: %w", err)
	}

	// Save file
	filename := fmt.Sprintf("%s_label.png", deviceID)
	filePath := filepath.Join(labelsDir, filename)

	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to write label file: %w", err)
	}

	// Update device record with label path
	labelPath := fmt.Sprintf("/labels/%s", filename)
	db := repository.GetDB()
	result := db.Exec("UPDATE devices SET label_path = ? WHERE deviceID = ?", labelPath, deviceID)
	if result.Error != nil {
		return "", fmt.Errorf("failed to update device label path: %w", result.Error)
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
	result := db.Exec("UPDATE cases SET label_path = ? WHERE caseID = ?", labelPath, caseID)
	if result.Error != nil {
		return "", fmt.Errorf("failed to update case label path: %w", result.Error)
	}

	return labelPath, nil
}

// AutoGenerateDeviceLabel automatically generates a label for a new device using the default template
// This renders the complete template with all elements (barcodes, QR codes, text, images)
func (s *LabelService) AutoGenerateDeviceLabel(deviceID string) error {
	// Get default template
	db := repository.GetDB()
	var template models.LabelTemplate
	if err := db.Where("is_default = ?", true).First(&template).Error; err != nil {
		// No default template, skip label generation silently
		return nil
	}

	// Generate label data using the template
	labelData, err := s.GenerateLabelForDevice(deviceID, template.ID)
	if err != nil {
		return fmt.Errorf("failed to generate label data: %w", err)
	}

	// Get template dimensions (convert mm to pixels at 300 DPI)
	// pixels = mm * 300 / 25.4 ≈ mm * 11.8
	labelWidthPx := int(template.Width * 11.8)
	labelHeightPx := int(template.Height * 11.8)

	// Create label image with white background
	labelImage := image.NewRGBA(image.Rect(0, 0, labelWidthPx, labelHeightPx))
	draw.Draw(labelImage, labelImage.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Process and draw all elements
	if elements, ok := labelData["elements"].([]map[string]interface{}); ok {
		for _, elem := range elements {
			// Get element position and dimensions (convert mm to pixels)
			x := int(elem["x"].(float64) * 11.8)
			y := int(elem["y"].(float64) * 11.8)

			// Draw element based on type
			elemType, _ := elem["type"].(string)

			if elemType == "text" {
				// Render text element
				content, ok := elem["content"].(string)
				if !ok || content == "" {
					continue
				}

				// Get style information
				style, _ := elem["style"].(map[string]interface{})

				// Parse font size (default to 16pt if not specified for better readability)
				fontSize := 16.0
				if style != nil {
					if fs, ok := style["font_size"].(float64); ok && fs > 0 {
						fontSize = fs
					}
				}

				// Parse text color (default to black)
				var textColor color.Color = color.Black
				if style != nil {
					if colorStr, ok := style["color"].(string); ok && colorStr != "" {
						// Simple color parsing for common colors
						switch colorStr {
						case "#000000", "black":
							textColor = color.Black
						case "#FFFFFF", "white":
							textColor = color.White
						case "#FF0000", "red":
							textColor = color.RGBA{R: 255, G: 0, B: 0, A: 255}
						}
					}
				}

				// Use basicfont for text rendering
				// basicfont.Face7x13 has 7px width and 13px height
				face := basicfont.Face7x13

				// Calculate scale factor based on desired point size
				// At 300 DPI: 1pt = 300/72 pixels = 4.167px
				// So fontSize in pt * 4.167 = desired height in pixels
				// Face7x13 is 13px tall, so scale = (fontSize * 4.167) / 13
				scaleFactor := (fontSize * 300.0 / 72.0) / 13.0

				// Calculate Y position (baseline)
				// Convert font size to pixels and add to Y position
				baselineOffset := int(fontSize * 300.0 / 72.0)
				scaledY := y + baselineOffset

				// Create a drawer for text
				point := fixed.Point26_6{
					X: fixed.Int26_6(x * 64),
					Y: fixed.Int26_6(scaledY * 64),
				}

				d := &font.Drawer{
					Dst:  labelImage,
					Src:  image.NewUniform(textColor),
					Face: face,
					Dot:  point,
				}

				// For larger text, scale each character individually with proper spacing
				if scaleFactor > 1.0 {
					// Calculate proper character spacing
					// Face7x13 has 7px character width
					// We want minimal spacing between characters for readability
					charWidth := 7.0
					charSpacing := charWidth * scaleFactor * 1.15 // 1.15x for slight letter spacing

					xOffset := float64(x)
					for _, ch := range content {
						d.Dot.X = fixed.Int26_6(int(xOffset) * 64)
						d.DrawString(string(ch))
						xOffset += charSpacing
					}
				} else {
					d.DrawString(content)
				}

			} else if elemType == "barcode" || elemType == "qrcode" || elemType == "image" {
				// Get image data
				imageData, ok := elem["image_data"].(string)
				if !ok || imageData == "" {
					continue
				}

				// Decode base64 image
				if len(imageData) > 22 && imageData[:22] == "data:image/png;base64," {
					imageData = imageData[22:]
				}
				imgBytes, err := base64.StdEncoding.DecodeString(imageData)
				if err != nil {
					continue
				}

				// Decode PNG
				elemImg, err := png.Decode(bytes.NewReader(imgBytes))
				if err != nil {
					continue
				}

				// Get target dimensions from template (convert mm to pixels)
				targetWidth := int(elem["width"].(float64) * 11.8)
				targetHeight := int(elem["height"].(float64) * 11.8)

				// Scale image to target dimensions
				srcBounds := elemImg.Bounds()
				scaledImg := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

				// Simple scaling by drawing with proper bounds
				scaleX := float64(targetWidth) / float64(srcBounds.Dx())
				scaleY := float64(targetHeight) / float64(srcBounds.Dy())

				for dy := 0; dy < targetHeight; dy++ {
					for dx := 0; dx < targetWidth; dx++ {
						srcX := int(float64(dx) / scaleX)
						srcY := int(float64(dy) / scaleY)
						scaledImg.Set(dx, dy, elemImg.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
					}
				}

				// Draw scaled image at position
				destRect := image.Rect(x, y, x+targetWidth, y+targetHeight)
				draw.Draw(labelImage, destRect, scaledImg, image.Point{}, draw.Over)
			}
		}
	}

	// Convert to PNG bytes
	var buf bytes.Buffer
	if err := png.Encode(&buf, labelImage); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	// Convert to base64
	base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())
	base64String := fmt.Sprintf("data:image/png;base64,%s", base64Image)

	// Save the label
	_, err = s.SaveLabelImage(deviceID, base64String)
	return err
}
