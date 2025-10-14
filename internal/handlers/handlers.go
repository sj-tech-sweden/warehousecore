package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"storagecore/internal/models"
	"storagecore/internal/repository"
	"storagecore/internal/services"
)

// HealthCheck returns server health status
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	db := repository.GetDB()
	if err := db.Ping(); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"service": "StorageCore",
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

	db := repository.GetDB()
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

	db := repository.GetDB()
	query := `
		SELECT d.deviceID, d.productID, d.status, d.barcode, d.qr_code, d.zone_id,
		       COALESCE(p.name, '') as product_name,
		       COALESCE(z.name, '') as zone_name
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
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

	devices := []models.DeviceWithDetails{}
	for rows.Next() {
		var d models.DeviceWithDetails
		rows.Scan(&d.DeviceID, &d.ProductID, &d.Status, &d.Barcode, &d.QRCode, &d.ZoneID, &d.ProductName, &d.ZoneName)
		devices = append(devices, d)
	}

	respondJSON(w, http.StatusOK, devices)
}

// GetDevice returns a single device by ID
func GetDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	db := repository.GetDB()
	var device models.DeviceWithDetails
	err := db.QueryRow(`
		SELECT d.*, COALESCE(p.name, '') as product_name,
		       COALESCE(z.name, '') as zone_name
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		WHERE d.deviceID = ?
	`, deviceID).Scan(&device.DeviceID, &device.ProductID, &device.SerialNumber, &device.Status,
		&device.Barcode, &device.QRCode, &device.ZoneID, &device.ProductName, &device.ZoneName)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Device not found"})
		return
	}
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, device)
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

	db := repository.GetDB()
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

	db := repository.GetDB()
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
	db := repository.GetDB()
	rows, err := db.Query(`
		SELECT zone_id, code, name, type, description, parent_zone_id, capacity, is_active
		FROM storage_zones
		WHERE is_active = TRUE
		ORDER BY name
	`)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	zones := []models.Zone{}
	for rows.Next() {
		var z models.Zone
		rows.Scan(&z.ZoneID, &z.Code, &z.Name, &z.Type, &z.Description, &z.ParentZoneID, &z.Capacity, &z.IsActive)
		zones = append(zones, z)
	}

	respondJSON(w, http.StatusOK, zones)
}

// CreateZone creates a new zone
func CreateZone(w http.ResponseWriter, r *http.Request) {
	// Input struct for API requests
	var input struct {
		Code         string  `json:"code"`
		Name         string  `json:"name"`
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

	db := repository.GetDB()

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

	result, err := db.Exec(`
		INSERT INTO storage_zones (code, name, type, description, parent_zone_id, capacity, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, input.Code, input.Name, input.Type, description, parentZoneID, capacity, input.IsActive)
	if err != nil {
		log.Printf("Zone creation error - SQL insert: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()

	// Return the created zone
	zone := models.Zone{
		ZoneID:   id,
		Code:     input.Code,
		Name:     input.Name,
		Type:     input.Type,
		IsActive: input.IsActive,
	}

	log.Printf("Zone created successfully: %s (ID: %d)", input.Name, id)
	respondJSON(w, http.StatusCreated, zone)
}

// GetZone returns a single zone
func GetZone(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	db := repository.GetDB()
	var z models.Zone
	err := db.QueryRow(`
		SELECT zone_id, code, name, type, description, parent_zone_id, capacity, is_active
		FROM storage_zones
		WHERE zone_id = ?
	`, id).Scan(&z.ZoneID, &z.Code, &z.Name, &z.Type, &z.Description, &z.ParentZoneID, &z.Capacity, &z.IsActive)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Zone not found"})
		return
	}
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, z)
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

	db := repository.GetDB()
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

	db := repository.GetDB()
	_, err := db.Exec(`UPDATE storage_zones SET is_active = FALSE WHERE zone_id = ?`, id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Zone deleted"})
}

// Placeholder handlers (simplified versions - to be expanded)
func GetJobs(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []map[string]string{})
}

func GetJobSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]
	respondJSON(w, http.StatusOK, map[string]interface{}{"job_id": jobID, "status": "in_progress"})
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

func GetDefects(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []map[string]string{})
}

func CreateDefect(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusCreated, map[string]string{"message": "Defect created"})
}

func UpdateDefect(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "Defect updated"})
}

func GetInspections(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []map[string]string{})
}

func GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	db := repository.GetDB()

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
