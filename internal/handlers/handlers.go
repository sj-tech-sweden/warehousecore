package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
	"warehousecore/internal/services"
)

// HealthCheck returns server health status
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()
	if err := db.Ping(); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"service": "WarehouseCore",
	})
}

// HandleScan processes barcode/QR scan requests
func HandleScan(w http.ResponseWriter, r *http.Request) {
	var req models.ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	scanService := services.NewScanService()
	response, err := scanService.ProcessScan(req, nil, r.RemoteAddr, r.UserAgent())
	if err != nil {
		log.Printf("Scan processing error: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// GetScanHistory returns scan event history
func GetScanHistory(w http.ResponseWriter, r *http.Request) {
	limit := parseInt(r.URL.Query().Get("limit"), 50)
	deviceID := r.URL.Query().Get("device_id")

	db := repository.GetSQLDB()
	query := `SELECT scan_id, scan_code, device_id, action, success, timestamp
	          FROM scan_events WHERE 1=1`
	args := []interface{}{}

	if deviceID != "" {
		query += " AND device_id = ?"
		args = append(args, deviceID)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	scans := []map[string]interface{}{}
	for rows.Next() {
		var s models.ScanEvent
		rows.Scan(&s.ScanID, &s.ScanCode, &s.DeviceID, &s.Action, &s.Success, &s.Timestamp)
		scans = append(scans, map[string]interface{}{
			"scan_id":   s.ScanID,
			"scan_code": s.ScanCode,
			"device_id": s.DeviceID,
			"action":    s.Action,
			"success":   s.Success,
			"timestamp": s.Timestamp,
		})
	}

	respondJSON(w, http.StatusOK, scans)
}

// GetDevices returns a list of devices with filters
func GetDevices(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	zoneID := r.URL.Query().Get("zone_id")
	limit := parseInt(r.URL.Query().Get("limit"), 100)

	db := repository.GetSQLDB()
	query := `
		SELECT d.deviceID, d.productID, d.serialnumber, d.status, d.barcode, d.qr_code,
		       d.zone_id, d.condition_rating, d.usage_hours,
		       COALESCE(p.name, '') as product_name,
		       COALESCE(z.name, '') as zone_name,
		       COALESCE(z.code, '') as zone_code,
		       COALESCE(c.name, '') as case_name,
		       COALESCE(CAST(jd.jobID AS CHAR), '') as job_number
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		LEFT JOIN devicescases dc ON d.deviceID = dc.deviceID
		LEFT JOIN cases c ON dc.caseID = c.caseID
		LEFT JOIN jobdevices jd ON d.deviceID = jd.deviceID AND jd.pack_status IN ('packed', 'issued')
		WHERE 1=1`

	args := []interface{}{}
	if status != "" {
		query += " AND d.status = ?"
		args = append(args, status)
	}
	if zoneID != "" {
		query += " AND d.zone_id = ?"
		args = append(args, zoneID)
	}

	query += " ORDER BY d.deviceID LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Response struct with clean JSON types
	type DeviceResponse struct {
		DeviceID        string   `json:"device_id"`
		ProductID       *int64   `json:"product_id,omitempty"`
		ProductName     string   `json:"product_name,omitempty"`
		SerialNumber    *string  `json:"serial_number,omitempty"`
		Barcode         *string  `json:"barcode,omitempty"`
		QRCode          *string  `json:"qr_code,omitempty"`
		Status          string   `json:"status"`
		ZoneID          *int64   `json:"zone_id,omitempty"`
		ZoneName        string   `json:"zone_name,omitempty"`
		ZoneCode        string   `json:"zone_code,omitempty"`
		CaseName        string   `json:"case_name,omitempty"`
		JobNumber       string   `json:"job_number,omitempty"`
		ConditionRating float64  `json:"condition_rating"`
		UsageHours      float64  `json:"usage_hours"`
	}

	devices := []DeviceResponse{}
	for rows.Next() {
		var d models.DeviceWithDetails
		var caseName, jobNumber string
		if err := rows.Scan(&d.DeviceID, &d.ProductID, &d.SerialNumber, &d.Status, &d.Barcode, &d.QRCode,
			&d.ZoneID, &d.ConditionRating, &d.UsageHours, &d.ProductName, &d.ZoneName, &d.ZoneCode,
			&caseName, &jobNumber); err != nil {
			log.Printf("Error scanning device row: %v", err)
			continue
		}

		// Convert to clean response format
		resp := DeviceResponse{
			DeviceID:        d.DeviceID,
			ProductName:     d.ProductName,
			Status:          d.Status,
			ZoneName:        d.ZoneName,
			ZoneCode:        d.ZoneCode,
			CaseName:        caseName,
			JobNumber:       jobNumber,
			ConditionRating: d.ConditionRating,
			UsageHours:      d.UsageHours,
		}

		if d.ProductID.Valid {
			resp.ProductID = &d.ProductID.Int64
		}
		if d.SerialNumber.Valid {
			resp.SerialNumber = &d.SerialNumber.String
		}
		if d.Barcode.Valid {
			resp.Barcode = &d.Barcode.String
		}
		if d.QRCode.Valid {
			resp.QRCode = &d.QRCode.String
		}
		if d.ZoneID.Valid {
			resp.ZoneID = &d.ZoneID.Int64
		}

		devices = append(devices, resp)
	}

	respondJSON(w, http.StatusOK, devices)
}

// GetDevice returns a single device by ID
func GetDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	db := repository.GetSQLDB()

	// Response struct with clean JSON types
	type DeviceResponse struct {
		DeviceID        string   `json:"device_id"`
		ProductID       *int64   `json:"product_id,omitempty"`
		ProductName     string   `json:"product_name,omitempty"`
		SerialNumber    *string  `json:"serial_number,omitempty"`
		Barcode         *string  `json:"barcode,omitempty"`
		QRCode          *string  `json:"qr_code,omitempty"`
		Status          string   `json:"status"`
		ZoneID          *int64   `json:"zone_id,omitempty"`
		ZoneName        string   `json:"zone_name,omitempty"`
		ZoneCode        string   `json:"zone_code,omitempty"`
		CaseName        string   `json:"case_name,omitempty"`
		JobNumber       string   `json:"job_number,omitempty"`
		ConditionRating float64  `json:"condition_rating"`
		UsageHours      float64  `json:"usage_hours"`
	}

	var device models.DeviceWithDetails
	var caseName, jobNumber string
	err := db.QueryRow(`
		SELECT d.deviceID, d.productID, d.serialnumber, d.status, d.barcode, d.qr_code,
		       d.zone_id, d.condition_rating, d.usage_hours,
		       COALESCE(p.name, '') as product_name,
		       COALESCE(z.name, '') as zone_name,
		       COALESCE(z.code, '') as zone_code,
		       COALESCE(c.name, '') as case_name,
		       COALESCE(CAST(jd.jobID AS CHAR), '') as job_number
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		LEFT JOIN devicescases dc ON d.deviceID = dc.deviceID
		LEFT JOIN cases c ON dc.caseID = c.caseID
		LEFT JOIN jobdevices jd ON d.deviceID = jd.deviceID AND jd.pack_status IN ('packed', 'issued')
		WHERE d.deviceID = ?
	`, deviceID).Scan(&device.DeviceID, &device.ProductID, &device.SerialNumber, &device.Status,
		&device.Barcode, &device.QRCode, &device.ZoneID, &device.ConditionRating, &device.UsageHours,
		&device.ProductName, &device.ZoneName, &device.ZoneCode, &caseName, &jobNumber)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Device not found"})
		return
	}
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Convert to clean response format
	resp := DeviceResponse{
		DeviceID:        device.DeviceID,
		ProductName:     device.ProductName,
		Status:          device.Status,
		ZoneName:        device.ZoneName,
		ZoneCode:        device.ZoneCode,
		CaseName:        caseName,
		JobNumber:       jobNumber,
		ConditionRating: device.ConditionRating,
		UsageHours:      device.UsageHours,
	}

	if device.ProductID.Valid {
		resp.ProductID = &device.ProductID.Int64
	}
	if device.SerialNumber.Valid {
		resp.SerialNumber = &device.SerialNumber.String
	}
	if device.Barcode.Valid {
		resp.Barcode = &device.Barcode.String
	}
	if device.QRCode.Valid {
		resp.QRCode = &device.QRCode.String
	}
	if device.ZoneID.Valid {
		resp.ZoneID = &device.ZoneID.Int64
	}

	respondJSON(w, http.StatusOK, resp)
}

// UpdateDeviceStatus updates device status
func UpdateDeviceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	db := repository.GetSQLDB()
	_, err := db.Exec(`UPDATE devices SET status = ? WHERE deviceID = ?`, req.Status, deviceID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Status updated"})
}

// GetDeviceMovements returns movement history for a device
func GetDeviceMovements(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	db := repository.GetSQLDB()
	rows, err := db.Query(`
		SELECT movement_id, device_id, action, from_zone_id, to_zone_id, to_job_id, timestamp
		FROM device_movements
		WHERE device_id = ?
		ORDER BY timestamp DESC
		LIMIT 50
	`, deviceID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	movements := []models.DeviceMovement{}
	for rows.Next() {
		var m models.DeviceMovement
		rows.Scan(&m.MovementID, &m.DeviceID, &m.Action, &m.FromZoneID, &m.ToZoneID, &m.ToJobID, &m.Timestamp)
		movements = append(movements, m)
	}

	respondJSON(w, http.StatusOK, movements)
}

// GetZones returns all zones
func GetZones(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()
	rows, err := db.Query(`
		SELECT zone_id, code, barcode, name, type, description, parent_zone_id, capacity, is_active
		FROM storage_zones
		WHERE is_active = TRUE
		ORDER BY name
	`)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Response struct with clean JSON types
	type ZoneResponse struct {
		ZoneID       int64   `json:"zone_id"`
		Code         string  `json:"code"`
		Barcode      *string `json:"barcode,omitempty"`
		Name         string  `json:"name"`
		Type         string  `json:"type"`
		Description  *string `json:"description,omitempty"`
		ParentZoneID *int64  `json:"parent_zone_id,omitempty"`
		Capacity     *int64  `json:"capacity,omitempty"`
		IsActive     bool    `json:"is_active"`
	}

	zones := []ZoneResponse{}
	for rows.Next() {
		var z models.Zone
		if err := rows.Scan(&z.ZoneID, &z.Code, &z.Barcode, &z.Name, &z.Type, &z.Description, &z.ParentZoneID, &z.Capacity, &z.IsActive); err != nil {
			log.Printf("Error scanning zone row: %v", err)
			continue
		}

		// Convert to clean response format
		resp := ZoneResponse{
			ZoneID:   z.ZoneID,
			Code:     z.Code,
			Name:     z.Name,
			Type:     z.Type,
			IsActive: z.IsActive,
		}

		if z.Barcode.Valid {
			resp.Barcode = &z.Barcode.String
		}
		if z.Description.Valid {
			resp.Description = &z.Description.String
		}
		if z.ParentZoneID.Valid {
			resp.ParentZoneID = &z.ParentZoneID.Int64
		}
		if z.Capacity.Valid {
			resp.Capacity = &z.Capacity.Int64
		}

		zones = append(zones, resp)
	}

	respondJSON(w, http.StatusOK, zones)
}

// CreateZone creates a new zone with automatic code generation
func CreateZone(w http.ResponseWriter, r *http.Request) {
	// Input struct for API requests
	var input struct {
		Code         *string `json:"code"` // Optional - will be auto-generated if not provided
		Name         *string `json:"name"` // Optional for shelves - will be auto-generated
		Type         string  `json:"type"`
		Description  *string `json:"description"`
		ParentZoneID *int64  `json:"parent_zone_id"`
		Capacity     *int64  `json:"capacity"`
		IsActive     bool    `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("Zone creation error - JSON decode: %v", err)
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request: " + err.Error()})
		return
	}

	zoneService := services.NewZoneService()
	db := repository.GetSQLDB()

	// Auto-generate name for shelves if not provided
	var zoneName string
	if input.Name != nil && *input.Name != "" {
		zoneName = *input.Name
	} else if input.Type == "shelf" {
		// Generate automatic name for shelves (Fächer)
		generatedName, err := zoneService.GenerateShelfName(input.ParentZoneID)
		if err != nil {
			log.Printf("Shelf name generation error: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate shelf name"})
			return
		}
		zoneName = generatedName
		log.Printf("Auto-generated shelf name: %s", zoneName)
	} else {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name is required for non-shelf zones"})
		return
	}

	// Auto-generate code if not provided
	var zoneCode string
	if input.Code != nil && *input.Code != "" {
		zoneCode = *input.Code
	} else {
		generatedCode, err := zoneService.GenerateZoneCode(zoneName, input.Type, input.ParentZoneID)
		if err != nil {
			log.Printf("Zone code generation error: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate zone code"})
			return
		}
		zoneCode = generatedCode
		log.Printf("Auto-generated zone code: %s for zone: %s", zoneCode, zoneName)
	}

	// Convert pointers to proper SQL values
	var description, parentZoneID, capacity interface{}
	if input.Description != nil && *input.Description != "" {
		description = *input.Description
	} else {
		description = nil
	}
	if input.ParentZoneID != nil {
		parentZoneID = *input.ParentZoneID
	} else {
		parentZoneID = nil
	}
	if input.Capacity != nil {
		capacity = *input.Capacity
	} else {
		capacity = nil
	}

	// Generate barcode for shelves
	var barcode interface{}
	if input.Type == "shelf" {
		// Generate barcode - will be updated with actual ID after insert
		barcode = nil // Will be set after we get the ID
	} else {
		barcode = nil
	}

	result, err := db.Exec(`
		INSERT INTO storage_zones (code, barcode, name, type, description, parent_zone_id, capacity, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, zoneCode, barcode, zoneName, input.Type, description, parentZoneID, capacity, input.IsActive)
	if err != nil {
		log.Printf("Zone creation error - SQL insert: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()

	// Generate and update barcode for shelves
	var generatedBarcode *string
	if input.Type == "shelf" {
		barcodeStr := fmt.Sprintf("FACH-%08d", id)
		_, err := db.Exec(`UPDATE storage_zones SET barcode = ? WHERE zone_id = ?`, barcodeStr, id)
		if err != nil {
			log.Printf("Failed to update barcode: %v", err)
		} else {
			generatedBarcode = &barcodeStr
		}
	}

	// Return the created zone with clean JSON
	type ZoneResponse struct {
		ZoneID       int64   `json:"zone_id"`
		Code         string  `json:"code"`
		Barcode      *string `json:"barcode,omitempty"`
		Name         string  `json:"name"`
		Type         string  `json:"type"`
		Description  *string `json:"description,omitempty"`
		ParentZoneID *int64  `json:"parent_zone_id,omitempty"`
		Capacity     *int64  `json:"capacity,omitempty"`
		IsActive     bool    `json:"is_active"`
	}

	zone := ZoneResponse{
		ZoneID:       id,
		Code:         zoneCode,
		Barcode:      generatedBarcode,
		Name:         zoneName,
		Type:         input.Type,
		Description:  input.Description,
		ParentZoneID: input.ParentZoneID,
		Capacity:     input.Capacity,
		IsActive:     input.IsActive,
	}

	log.Printf("Zone created successfully: %s (Code: %s, ID: %d)", zoneName, zoneCode, id)
	respondJSON(w, http.StatusCreated, zone)
}

// GetZone returns a single zone with details (subzones, devices, breadcrumb)
func GetZone(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	zoneID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid zone ID"})
		return
	}

	zoneService := services.NewZoneService()
	details, err := zoneService.GetZoneDetails(zoneID)
	if err != nil {
		log.Printf("Error getting zone details: %v", err)
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Zone not found"})
		return
	}

	respondJSON(w, http.StatusOK, details)
}

// GetZoneDevices returns all devices in a specific zone
func GetZoneDevices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zoneID := vars["id"]

	db := repository.GetSQLDB()
	rows, err := db.Query(`
		SELECT d.deviceID, d.productID, d.serialnumber, d.status, d.barcode, d.qr_code,
		       d.condition_rating, d.usage_hours,
		       COALESCE(p.name, '') as product_name,
		       COALESCE(m.name, '') as manufacturer,
		       COALESCE(b.name, '') as model,
		       COALESCE(z.code, '') as zone_code
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN manufacturer m ON p.manufacturerID = m.manufacturerID
		LEFT JOIN brands b ON p.brandID = b.brandID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		WHERE d.zone_id = ? AND d.status = 'in_storage'
		ORDER BY p.name, d.deviceID
	`, zoneID)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type DeviceInZone struct {
		DeviceID        string  `json:"device_id"`
		ProductID       *int    `json:"product_id"`
		ProductName     string  `json:"product_name"`
		SerialNumber    *string `json:"serial_number,omitempty"`
		Manufacturer    string  `json:"manufacturer,omitempty"`
		Model           string  `json:"model,omitempty"`
		Status          string  `json:"status"`
		Barcode         *string `json:"barcode,omitempty"`
		QRCode          *string `json:"qr_code,omitempty"`
		ZoneCode        string  `json:"zone_code,omitempty"`
		ConditionRating float64 `json:"condition_rating"`
		UsageHours      float64 `json:"usage_hours"`
	}

	devices := []DeviceInZone{}
	for rows.Next() {
		var d DeviceInZone
		var productID sql.NullInt64
		var barcode, qrCode, serialNumber sql.NullString

		if err := rows.Scan(&d.DeviceID, &productID, &serialNumber, &d.Status, &barcode, &qrCode,
			&d.ConditionRating, &d.UsageHours, &d.ProductName, &d.Manufacturer, &d.Model, &d.ZoneCode); err != nil {
			log.Printf("Error scanning device row: %v", err)
			continue
		}

		if productID.Valid {
			pid := int(productID.Int64)
			d.ProductID = &pid
		}
		if barcode.Valid {
			d.Barcode = &barcode.String
		}
		if qrCode.Valid {
			d.QRCode = &qrCode.String
		}
		if serialNumber.Valid {
			d.SerialNumber = &serialNumber.String
		}

		devices = append(devices, d)
	}

	respondJSON(w, http.StatusOK, devices)
}

// UpdateZone updates a zone
func UpdateZone(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var zone models.Zone
	if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	db := repository.GetSQLDB()
	_, err := db.Exec(`
		UPDATE storage_zones
		SET code = ?, name = ?, type = ?, description = ?, parent_zone_id = ?, capacity = ?
		WHERE zone_id = ?
	`, zone.Code, zone.Name, zone.Type, zone.Description, zone.ParentZoneID, zone.Capacity, id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Zone updated"})
}

// DeleteZone soft-deletes a zone
func DeleteZone(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	db := repository.GetSQLDB()
	_, err := db.Exec(`UPDATE storage_zones SET is_active = FALSE WHERE zone_id = ?`, id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Zone deleted"})
}

// GetZoneByBarcode finds a zone by barcode or code
func GetZoneByBarcode(w http.ResponseWriter, r *http.Request) {
	scanCode := r.URL.Query().Get("scan_code")
	if scanCode == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "scan_code parameter required"})
		return
	}

	db := repository.GetSQLDB()
	var zone models.Zone
	err := db.QueryRow(`
		SELECT zone_id, code, barcode, name, type, description, parent_zone_id, capacity, is_active
		FROM storage_zones
		WHERE (barcode = ? OR code = ?) AND is_active = TRUE
		LIMIT 1
	`, scanCode, scanCode).Scan(&zone.ZoneID, &zone.Code, &zone.Barcode, &zone.Name, &zone.Type,
		&zone.Description, &zone.ParentZoneID, &zone.Capacity, &zone.IsActive)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Zone not found"})
		return
	}
	if err != nil {
		log.Printf("Error finding zone by barcode: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Response struct with clean JSON types
	type ZoneResponse struct {
		ZoneID       int64   `json:"zone_id"`
		Code         string  `json:"code"`
		Barcode      *string `json:"barcode,omitempty"`
		Name         string  `json:"name"`
		Type         string  `json:"type"`
		Description  *string `json:"description,omitempty"`
		ParentZoneID *int64  `json:"parent_zone_id,omitempty"`
		Capacity     *int64  `json:"capacity,omitempty"`
		IsActive     bool    `json:"is_active"`
	}

	resp := ZoneResponse{
		ZoneID:   zone.ZoneID,
		Code:     zone.Code,
		Name:     zone.Name,
		Type:     zone.Type,
		IsActive: zone.IsActive,
	}

	if zone.Barcode.Valid {
		resp.Barcode = &zone.Barcode.String
	}
	if zone.Description.Valid {
		resp.Description = &zone.Description.String
	}
	if zone.ParentZoneID.Valid {
		resp.ParentZoneID = &zone.ParentZoneID.Int64
	}
	if zone.Capacity.Valid {
		resp.Capacity = &zone.Capacity.Int64
	}

	respondJSON(w, http.StatusOK, resp)
}

// GetJobs returns jobs filtered by status
func GetJobs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	db := repository.GetSQLDB()
	query := `
		SELECT j.jobID, j.description, j.startDate, j.endDate, s.status,
		       COALESCE(c.firstName, '') as customer_first_name,
		       COALESCE(c.lastName, '') as customer_last_name,
		       COUNT(DISTINCT jd.deviceID) as device_count
		FROM jobs j
		LEFT JOIN status s ON j.statusID = s.statusID
		LEFT JOIN customers c ON j.customerID = c.customerID
		LEFT JOIN jobdevices jd ON j.jobID = jd.jobID
		WHERE 1=1`

	args := []interface{}{}
	if status != "" {
		query += " AND s.status = ?"
		args = append(args, status)
	}

	query += " GROUP BY j.jobID ORDER BY j.startDate ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error getting jobs: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type JobResponse struct {
		JobID             int     `json:"job_id"`
		Description       *string `json:"description,omitempty"`
		StartDate         *string `json:"start_date,omitempty"`
		EndDate           *string `json:"end_date,omitempty"`
		Status            string  `json:"status"`
		CustomerFirstName string  `json:"customer_first_name,omitempty"`
		CustomerLastName  string  `json:"customer_last_name,omitempty"`
		DeviceCount       int     `json:"device_count"`
	}

	jobs := []JobResponse{}
	for rows.Next() {
		var j JobResponse
		var description, startDate, endDate sql.NullString

		if err := rows.Scan(&j.JobID, &description, &startDate, &endDate, &j.Status,
			&j.CustomerFirstName, &j.CustomerLastName, &j.DeviceCount); err != nil {
			log.Printf("Error scanning job row: %v", err)
			continue
		}

		if description.Valid {
			j.Description = &description.String
		}
		if startDate.Valid {
			j.StartDate = &startDate.String
		}
		if endDate.Valid {
			j.EndDate = &endDate.String
		}

		jobs = append(jobs, j)
	}

	respondJSON(w, http.StatusOK, jobs)
}

func GetJobSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	db := repository.GetSQLDB()

	// Get job details
	var (
		description, startDate, endDate sql.NullString
		status                          string
		customerFirstName, customerLastName string
	)

	err := db.QueryRow(`
		SELECT j.description, j.startDate, j.endDate, s.status,
		       COALESCE(c.firstName, '') as customer_first_name,
		       COALESCE(c.lastName, '') as customer_last_name
		FROM jobs j
		LEFT JOIN status s ON j.statusID = s.statusID
		LEFT JOIN customers c ON j.customerID = c.customerID
		WHERE j.jobID = ?
	`, jobID).Scan(&description, &startDate, &endDate, &status, &customerFirstName, &customerLastName)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Job not found"})
		return
	}
	if err != nil {
		log.Printf("Error getting job: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Get devices for this job with their current status
	rows, err := db.Query(`
		SELECT jd.deviceID, d.status, d.barcode, d.qr_code,
		       COALESCE(p.name, '') as product_name,
		       COALESCE(z.name, '') as zone_name,
		       jd.pack_status
		FROM jobdevices jd
		LEFT JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		WHERE jd.jobID = ?
		ORDER BY p.name, jd.deviceID
	`, jobID)

	if err != nil {
		log.Printf("Error getting job devices: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type JobDevice struct {
		DeviceID    string  `json:"device_id"`
		Status      string  `json:"status"`
		ProductName string  `json:"product_name"`
		ZoneName    string  `json:"zone_name,omitempty"`
		Barcode     *string `json:"barcode,omitempty"`
		QRCode      *string `json:"qr_code,omitempty"`
		PackStatus  string  `json:"pack_status"`
		Scanned     bool    `json:"scanned"` // true if pack_status is 'issued' or device status is 'on_job'
	}

	devices := []JobDevice{}
	for rows.Next() {
		var jd JobDevice
		var barcode, qrCode sql.NullString

		if err := rows.Scan(&jd.DeviceID, &jd.Status, &barcode, &qrCode,
			&jd.ProductName, &jd.ZoneName, &jd.PackStatus); err != nil {
			log.Printf("Error scanning device row: %v", err)
			continue
		}

		if barcode.Valid {
			jd.Barcode = &barcode.String
		}
		if qrCode.Valid {
			jd.QRCode = &qrCode.String
		}

		// Mark as scanned if device is on_job or pack_status is issued
		jd.Scanned = jd.Status == "on_job" || jd.PackStatus == "issued"

		devices = append(devices, jd)
	}

	response := map[string]interface{}{
		"job_id":              jobID,
		"status":              status,
		"customer_first_name": customerFirstName,
		"customer_last_name":  customerLastName,
		"devices":             devices,
	}

	if description.Valid {
		response["description"] = description.String
	}
	if startDate.Valid {
		response["start_date"] = startDate.String
	}
	if endDate.Valid {
		response["end_date"] = endDate.String
	}

	respondJSON(w, http.StatusOK, response)
}

func CompleteJob(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "Job completed"})
}

func GetCases(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []map[string]string{})
}

func GetCase(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{})
}

func GetCaseContents(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []map[string]string{})
}

// GetDefects returns defect reports with filters
func GetDefects(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	severity := r.URL.Query().Get("severity")
	deviceID := r.URL.Query().Get("device_id")

	db := repository.GetSQLDB()
	query := `
		SELECT dr.defect_id, dr.device_id, dr.severity, dr.status, dr.title, dr.description,
		       dr.reported_at, dr.repair_cost, dr.repaired_at, dr.closed_at,
		       COALESCE(p.name, '') as product_name
		FROM defect_reports dr
		LEFT JOIN devices d ON dr.device_id = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		WHERE 1=1`

	args := []interface{}{}
	if status != "" {
		query += " AND dr.status = ?"
		args = append(args, status)
	}
	if severity != "" {
		query += " AND dr.severity = ?"
		args = append(args, severity)
	}
	if deviceID != "" {
		query += " AND dr.device_id = ?"
		args = append(args, deviceID)
	}

	query += " ORDER BY dr.reported_at DESC LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error getting defects: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type DefectResponse struct {
		DefectID    int64   `json:"defect_id"`
		DeviceID    string  `json:"device_id"`
		Severity    string  `json:"severity"`
		Status      string  `json:"status"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		ReportedAt  string  `json:"reported_at"`
		RepairCost  *float64 `json:"repair_cost,omitempty"`
		RepairedAt  *string `json:"repaired_at,omitempty"`
		ClosedAt    *string `json:"closed_at,omitempty"`
		ProductName string  `json:"product_name,omitempty"`
	}

	defects := []DefectResponse{}
	for rows.Next() {
		var d DefectResponse
		var reportedAt, repairedAt, closedAt sql.NullTime
		var repairCost sql.NullFloat64

		if err := rows.Scan(&d.DefectID, &d.DeviceID, &d.Severity, &d.Status, &d.Title, &d.Description,
			&reportedAt, &repairCost, &repairedAt, &closedAt, &d.ProductName); err != nil {
			log.Printf("Error scanning defect row: %v", err)
			continue
		}

		if reportedAt.Valid {
			d.ReportedAt = reportedAt.Time.Format("2006-01-02T15:04:05Z")
		}
		if repairCost.Valid {
			cost := repairCost.Float64
			d.RepairCost = &cost
		}
		if repairedAt.Valid {
			repaired := repairedAt.Time.Format("2006-01-02T15:04:05Z")
			d.RepairedAt = &repaired
		}
		if closedAt.Valid {
			closed := closedAt.Time.Format("2006-01-02T15:04:05Z")
			d.ClosedAt = &closed
		}

		defects = append(defects, d)
	}

	respondJSON(w, http.StatusOK, defects)
}

func CreateDefect(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DeviceID    string  `json:"device_id"`
		Severity    string  `json:"severity"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		AssignedTo  *int64  `json:"assigned_to,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if input.DeviceID == "" || input.Title == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "device_id and title are required"})
		return
	}

	db := repository.GetSQLDB()

	// Set device status to defective
	_, err := db.Exec(`UPDATE devices SET status = 'defective' WHERE deviceID = ?`, input.DeviceID)
	if err != nil {
		log.Printf("Error updating device status: %v", err)
	}

	// Create defect report
	result, err := db.Exec(`
		INSERT INTO defect_reports (device_id, severity, title, description, assigned_to, status)
		VALUES (?, ?, ?, ?, ?, 'open')
	`, input.DeviceID, input.Severity, input.Title, input.Description, input.AssignedTo)

	if err != nil {
		log.Printf("Error creating defect: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	defectID, _ := result.LastInsertId()

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"defect_id": defectID,
		"message":   "Defect report created successfully",
	})
}

func UpdateDefect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	defectID := vars["id"]

	var input struct {
		Status      *string  `json:"status,omitempty"`
		AssignedTo  *int64   `json:"assigned_to,omitempty"`
		RepairCost  *float64 `json:"repair_cost,omitempty"`
		RepairNotes *string  `json:"repair_notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	db := repository.GetSQLDB()

	// Build dynamic UPDATE query
	updates := []string{}
	args := []interface{}{}

	if input.Status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *input.Status)

		// Update timestamps based on status
		if *input.Status == "repaired" {
			updates = append(updates, "repaired_at = NOW()")
		} else if *input.Status == "closed" {
			updates = append(updates, "closed_at = NOW()")
		}
	}
	if input.AssignedTo != nil {
		updates = append(updates, "assigned_to = ?")
		args = append(args, *input.AssignedTo)
	}
	if input.RepairCost != nil {
		updates = append(updates, "repair_cost = ?")
		args = append(args, *input.RepairCost)
	}
	if input.RepairNotes != nil {
		updates = append(updates, "repair_notes = ?")
		args = append(args, *input.RepairNotes)
	}

	if len(updates) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "No fields to update"})
		return
	}

	query := "UPDATE defect_reports SET " + strings.Join(updates, ", ") + " WHERE defect_id = ?"
	args = append(args, defectID)

	_, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating defect: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// If status is repaired or closed, update device status
	if input.Status != nil && (*input.Status == "repaired" || *input.Status == "closed") {
		var deviceID string
		db.QueryRow(`SELECT device_id FROM defect_reports WHERE defect_id = ?`, defectID).Scan(&deviceID)
		if deviceID != "" {
			db.Exec(`UPDATE devices SET status = 'in_storage' WHERE deviceID = ?`, deviceID)
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Defect updated successfully"})
}

func GetInspections(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status") // upcoming, overdue, all
	deviceID := r.URL.Query().Get("device_id")

	db := repository.GetSQLDB()
	query := `
		SELECT i.schedule_id, i.device_id, i.product_id, i.inspection_type,
		       i.interval_days, i.last_inspection, i.next_inspection, i.is_active,
		       COALESCE(p.name, '') as product_name,
		       COALESCE(d.deviceID, '') as device_name
		FROM inspection_schedules i
		LEFT JOIN products p ON i.product_id = p.productID
		LEFT JOIN devices d ON i.device_id = d.deviceID
		WHERE 1=1`

	args := []interface{}{}

	if status == "upcoming" {
		query += " AND i.next_inspection >= NOW() AND i.next_inspection <= DATE_ADD(NOW(), INTERVAL 30 DAY) AND i.is_active = 1"
	} else if status == "overdue" {
		query += " AND i.next_inspection < NOW() AND i.is_active = 1"
	} else if status == "active" {
		query += " AND i.is_active = 1"
	}

	if deviceID != "" {
		query += " AND i.device_id = ?"
		args = append(args, deviceID)
	}

	query += " ORDER BY i.next_inspection ASC LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error getting inspections: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type InspectionResponse struct {
		ScheduleID      int64   `json:"schedule_id"`
		DeviceID        *string `json:"device_id,omitempty"`
		ProductID       *int64  `json:"product_id,omitempty"`
		InspectionType  string  `json:"inspection_type"`
		IntervalDays    int     `json:"interval_days"`
		LastInspection  *string `json:"last_inspection,omitempty"`
		NextInspection  *string `json:"next_inspection,omitempty"`
		IsActive        bool    `json:"is_active"`
		ProductName     string  `json:"product_name,omitempty"`
		DeviceName      string  `json:"device_name,omitempty"`
	}

	inspections := []InspectionResponse{}
	for rows.Next() {
		var i InspectionResponse
		var deviceID sql.NullString
		var lastInspection, nextInspection sql.NullTime
		var prodID sql.NullInt64

		if err := rows.Scan(&i.ScheduleID, &deviceID, &prodID, &i.InspectionType,
			&i.IntervalDays, &lastInspection, &nextInspection, &i.IsActive,
			&i.ProductName, &i.DeviceName); err != nil {
			log.Printf("Error scanning inspection row: %v", err)
			continue
		}

		if deviceID.Valid {
			i.DeviceID = &deviceID.String
		}
		if prodID.Valid {
			pid := prodID.Int64
			i.ProductID = &pid
		}
		if lastInspection.Valid {
			last := lastInspection.Time.Format("2006-01-02T15:04:05Z")
			i.LastInspection = &last
		}
		if nextInspection.Valid {
			next := nextInspection.Time.Format("2006-01-02T15:04:05Z")
			i.NextInspection = &next
		}

		inspections = append(inspections, i)
	}

	respondJSON(w, http.StatusOK, inspections)
}

func GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	var inStorage, onJob, defective int
	db.QueryRow(`SELECT COUNT(*) FROM devices WHERE status = 'in_storage'`).Scan(&inStorage)
	db.QueryRow(`SELECT COUNT(*) FROM devices WHERE status = 'on_job' OR status = 'rented'`).Scan(&onJob)
	db.QueryRow(`SELECT COUNT(*) FROM devices WHERE status = 'defective'`).Scan(&defective)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"in_storage": inStorage,
		"on_job":     onJob,
		"defective":  defective,
		"total":      inStorage + onJob + defective,
	})
}

func GetMovements(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []map[string]string{})
}

// GetDeviceTree returns devices organized in a hierarchical tree structure by categories
func GetDeviceTree(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	// Query for device tree with categories
	query := `
		SELECT
			c.categoryID,
			c.name as category_name,
			sc.subcategoryID,
			sc.name as subcategory_name,
			sbc.subbiercategoryID,
			sbc.name as subbiercategory_name,
			d.deviceID,
			d.status,
			d.barcode,
			d.serialnumber,
			COALESCE(p.name, '') as product_name,
			d.zone_id,
			COALESCE(z.code, '') as zone_code
		FROM categories c
		LEFT JOIN subcategories sc ON c.categoryID = sc.categoryID
		LEFT JOIN subbiercategories sbc ON sc.subcategoryID = sbc.subcategoryID
		LEFT JOIN products p ON sbc.subbiercategoryID = p.subbiercategoryID
		LEFT JOIN devices d ON p.productID = d.productID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		ORDER BY c.name, sc.name, sbc.name, p.name, d.deviceID
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying device tree: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Build tree structure
	categories := make(map[int]*map[string]interface{})
	subcategories := make(map[string]*map[string]interface{})
	subbiercategories := make(map[string]*map[string]interface{})

	for rows.Next() {
		var categoryID sql.NullInt64
		var subcategoryID, subbiercategoryID sql.NullString
		var categoryName, subcategoryName, subbiercategoryName sql.NullString
		var deviceID, status, barcode, serialNumber, productName sql.NullString
		var zoneID sql.NullInt64
		var zoneCode sql.NullString

		err := rows.Scan(&categoryID, &categoryName, &subcategoryID, &subcategoryName,
			&subbiercategoryID, &subbiercategoryName, &deviceID, &status, &barcode,
			&serialNumber, &productName, &zoneID, &zoneCode)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Skip if no category
		if !categoryID.Valid {
			continue
		}

		// Get or create category
		catID := int(categoryID.Int64)
		if _, exists := categories[catID]; !exists {
			categories[catID] = &map[string]interface{}{
				"id":              catID,
				"name":            categoryName.String,
				"subcategories":   []interface{}{},
				"direct_devices":  []interface{}{},
				"device_count":    0,
			}
		}

		// Process subcategory if exists
		if subcategoryID.Valid {
			subCatID := subcategoryID.String
			if _, exists := subcategories[subCatID]; !exists {
				subcategories[subCatID] = &map[string]interface{}{
					"id":                 subCatID,
					"name":               subcategoryName.String,
					"subbiercategories":  []interface{}{},
					"direct_devices":     []interface{}{},
					"device_count":       0,
				}
				// Add subcategory to category
				cat := *categories[catID]
				cat["subcategories"] = append(cat["subcategories"].([]interface{}), subcategories[subCatID])
			}

			// Process subbiercategory if exists
			if subbiercategoryID.Valid {
				subBierCatID := subbiercategoryID.String
				if _, exists := subbiercategories[subBierCatID]; !exists {
					subbiercategories[subBierCatID] = &map[string]interface{}{
						"id":           subBierCatID,
						"name":         subbiercategoryName.String,
						"devices":      []interface{}{},
						"device_count": 0,
					}
					// Add subbiercategory to subcategory
					subCat := *subcategories[subCatID]
					subCat["subbiercategories"] = append(subCat["subbiercategories"].([]interface{}), subbiercategories[subBierCatID])
				}

				// Add device to subbiercategory if exists
				if deviceID.Valid {
					device := map[string]interface{}{
						"device_id":    deviceID.String,
						"product_name": productName.String,
						"status":       status.String,
					}
					if barcode.Valid {
						device["barcode"] = barcode.String
					}
					if serialNumber.Valid {
						device["serial_number"] = serialNumber.String
					}
					if zoneID.Valid {
						device["zone_id"] = zoneID.Int64
						device["zone_code"] = zoneCode.String
					}

					subBierCat := *subbiercategories[subBierCatID]
					subBierCat["devices"] = append(subBierCat["devices"].([]interface{}), device)
					subBierCat["device_count"] = subBierCat["device_count"].(int) + 1

					// Update counts
					subCat := *subcategories[subCatID]
					subCat["device_count"] = subCat["device_count"].(int) + 1
					cat := *categories[catID]
					cat["device_count"] = cat["device_count"].(int) + 1
				}
			}
		}
	}

	// Convert map to slice
	treeData := []interface{}{}
	for _, cat := range categories {
		treeData = append(treeData, *cat)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"treeData": treeData,
	})
}

// AssignDevicesToZone assigns multiple devices to a storage zone
func AssignDevicesToZone(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zoneID := vars["id"]

	var input struct {
		DeviceIDs []string `json:"device_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if len(input.DeviceIDs) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "No devices specified"})
		return
	}

	db := repository.GetSQLDB()

	// Verify zone exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM storage_zones WHERE zone_id = ?)", zoneID).Scan(&exists)
	if err != nil || !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Zone not found"})
		return
	}

	// Update devices
	successCount := 0
	failedDevices := []string{}

	for _, deviceID := range input.DeviceIDs {
		result, err := db.Exec(`
			UPDATE devices
			SET zone_id = ?,
			    status = CASE
			        WHEN status = 'on_job' OR status = 'rented' THEN status
			        ELSE 'in_storage'
			    END
			WHERE deviceID = ?
		`, zoneID, deviceID)

		if err != nil {
			log.Printf("Error assigning device %s to zone %s: %v", deviceID, zoneID, err)
			failedDevices = append(failedDevices, deviceID)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			successCount++

			// Log movement
			_, _ = db.Exec(`
				INSERT INTO device_movements (device_id, from_zone_id, to_zone_id, movement_type, moved_at)
				SELECT ?, zone_id, ?, 'assignment', NOW()
				FROM devices WHERE deviceID = ?
			`, deviceID, zoneID, deviceID)
		} else {
			failedDevices = append(failedDevices, deviceID)
		}
	}

	response := map[string]interface{}{
		"success":       successCount,
		"total":         len(input.DeviceIDs),
		"failed_count":  len(failedDevices),
	}

	if len(failedDevices) > 0 {
		response["failed_devices"] = failedDevices
	}

	respondJSON(w, http.StatusOK, response)
}

// Helper functions
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetMaintenanceStats returns maintenance dashboard statistics  
func GetMaintenanceStats(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	var openDefects, inProgressDefects, repairedDefects int
	var overdueInspections, upcomingInspections int

	db.QueryRow(`SELECT COUNT(*) FROM defect_reports WHERE status = 'open'`).Scan(&openDefects)
	db.QueryRow(`SELECT COUNT(*) FROM defect_reports WHERE status = 'in_progress'`).Scan(&inProgressDefects)
	db.QueryRow(`SELECT COUNT(*) FROM defect_reports WHERE status = 'repaired'`).Scan(&repairedDefects)
	db.QueryRow(`SELECT COUNT(*) FROM inspection_schedules WHERE next_inspection < NOW() AND is_active = 1`).Scan(&overdueInspections)
	db.QueryRow(`SELECT COUNT(*) FROM inspection_schedules WHERE next_inspection >= NOW() AND next_inspection <= DATE_ADD(NOW(), INTERVAL 30 DAY) AND is_active = 1`).Scan(&upcomingInspections)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"open_defects":        openDefects,
		"in_progress_defects": inProgressDefects,
		"repaired_defects":    repairedDefects,
		"overdue_inspections": overdueInspections,
		"upcoming_inspections": upcomingInspections,
	})
}
