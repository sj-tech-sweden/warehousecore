package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"warehousecore/internal/models"
	"warehousecore/internal/services"
)

var labelService = services.NewLabelService()

// GenerateQRCode generates a QR code
// POST /api/v1/labels/qrcode
// Body: {"content": "text", "size": 256}
func GenerateQRCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		Size    int    `json:"size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	qrData, err := labelService.GenerateQRCode(req.Content, req.Size)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"image_data": qrData,
	})
}

// GenerateBarcode generates a barcode
// POST /api/v1/labels/barcode
// Body: {"content": "text", "width": 300, "height": 100}
func GenerateBarcode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		Width   int    `json:"width"`
		Height  int    `json:"height"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	barcodeData, err := labelService.GenerateBarcode(req.Content, req.Width, req.Height)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"image_data": barcodeData,
	})
}

// GetLabelTemplates retrieves all label templates
// GET /api/v1/labels/templates
func GetLabelTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := labelService.GetAllTemplates()
	if err != nil {
		log.Printf("Error fetching label templates: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure templates is never nil for JSON encoding (client expects array)
	if templates == nil {
		templates = []models.LabelTemplate{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(templates); err != nil {
		log.Printf("Error encoding label templates: %v", err)
	}
}

// GetLabelTemplate retrieves a specific template
// GET /api/v1/labels/templates/{id}
func GetLabelTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	template, err := labelService.GetTemplateByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// CreateLabelTemplate creates a new label template
// POST /api/v1/labels/templates
func CreateLabelTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.LabelTemplate

	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if template.Name == "" || template.Width == 0 || template.Height == 0 || template.TemplateJSON == "" {
		http.Error(w, "Name, width, height, and template_json are required", http.StatusBadRequest)
		return
	}

	if err := labelService.CreateTemplate(&template); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

// UpdateLabelTemplate updates a label template
// PUT /api/v1/labels/templates/{id}
func UpdateLabelTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := labelService.UpdateTemplate(id, updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteLabelTemplate deletes a label template
// DELETE /api/v1/labels/templates/{id}
func DeleteLabelTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	if err := labelService.DeleteTemplate(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateDeviceLabel generates a complete label for a device
// POST /api/v1/labels/device/{device_id}
// Body: {"template_id": 1}
func GenerateDeviceLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["device_id"]

	var req struct {
		TemplateID int `json:"template_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TemplateID == 0 {
		http.Error(w, "Template ID is required", http.StatusBadRequest)
		return
	}

	labelData, err := labelService.GenerateLabelForDevice(deviceID, req.TemplateID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(labelData)
}

// GenerateCaseLabel generates a complete label for a case
// POST /api/v1/labels/case/{case_id}
// Body: {"template_id": 1}
func GenerateCaseLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	caseIDStr := vars["case_id"]

	caseID, err := strconv.Atoi(caseIDStr)
	if err != nil {
		http.Error(w, "Invalid case ID", http.StatusBadRequest)
		return
	}

	var req struct {
		TemplateID int `json:"template_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TemplateID == 0 {
		http.Error(w, "Template ID is required", http.StatusBadRequest)
		return
	}

	labelData, err := labelService.GenerateLabelForCase(caseID, req.TemplateID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(labelData)
}

// SaveDeviceLabel saves a generated label image for a device
// POST /api/v1/labels/save
// Body: {"device_id": "DEV001", "image_data": "data:image/png;base64,..."}
func SaveDeviceLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID  string `json:"device_id"`
		ImageData string `json:"image_data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" || req.ImageData == "" {
		http.Error(w, "Device ID and image data are required", http.StatusBadRequest)
		return
	}

	labelPath, err := labelService.SaveLabelImage(req.DeviceID, req.ImageData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"label_path": labelPath,
		"message":    "Label saved successfully",
	})
}

// SaveCaseLabel saves a generated label image for a case
// POST /api/v1/labels/save-case
// Body: {"case_id": 1001, "image_data": "data:image/png;base64,..."}
func SaveCaseLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CaseID    int    `json:"case_id"`
		ImageData string `json:"image_data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CaseID == 0 || req.ImageData == "" {
		http.Error(w, "Case ID and image data are required", http.StatusBadRequest)
		return
	}

	labelPath, err := labelService.SaveCaseLabelImage(req.CaseID, req.ImageData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"label_path": labelPath,
		"message":    "Case label saved successfully",
	})
}

// GenerateZoneLabel generates a complete label for a zone
// POST /api/v1/labels/zone/{zone_id}
// Body: {"template_id": 1}
func GenerateZoneLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zoneIDStr := vars["zone_id"]

	zoneID, err := strconv.ParseInt(zoneIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid zone ID", http.StatusBadRequest)
		return
	}

	var req struct {
		TemplateID int `json:"template_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TemplateID == 0 {
		http.Error(w, "Template ID is required", http.StatusBadRequest)
		return
	}

	labelData, err := labelService.GenerateLabelForZone(zoneID, req.TemplateID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(labelData)
}

// SaveZoneLabel saves a generated label image for a zone
// POST /api/v1/labels/save-zone
// Body: {"zone_id": 1, "image_data": "data:image/png;base64,..."}
func SaveZoneLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ZoneID    int64  `json:"zone_id"`
		ImageData string `json:"image_data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ZoneID == 0 || req.ImageData == "" {
		http.Error(w, "Zone ID and image data are required", http.StatusBadRequest)
		return
	}

	labelPath, err := labelService.SaveZoneLabelImage(req.ZoneID, req.ImageData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"label_path": labelPath,
		"message":    "Zone label saved successfully",
	})
}
