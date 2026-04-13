package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

// Cable represents a cable inventory item
type Cable struct {
	CableID    int      `json:"cable_id"`
	Name       *string  `json:"name"`
	Connector1 int      `json:"connector1"`
	Connector2 int      `json:"connector2"`
	Typ        int      `json:"typ"`
	Length     float64  `json:"length"`
	MM2        *float64 `json:"mm2"`

	// Joined fields for display
	Connector1Name   *string `json:"connector1_name,omitempty"`
	Connector2Name   *string `json:"connector2_name,omitempty"`
	CableTypeName    *string `json:"cable_type_name,omitempty"`
	Connector1Gender *string `json:"connector1_gender,omitempty"`
	Connector2Gender *string `json:"connector2_gender,omitempty"`
}

// CableConnector represents a cable connector type
type CableConnector struct {
	ConnectorID  int     `json:"connector_id"`
	Name         string  `json:"name"`
	Abbreviation *string `json:"abbreviation"`
	Gender       *string `json:"gender"`
}

// CableType represents a cable type
type CableType struct {
	CableTypeID int    `json:"cable_type_id"`
	Name        string `json:"name"`
	Count       int    `json:"count"`
}

// GetAllCables retrieves all cables with optional filtering
func GetAllCables(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	search := r.URL.Query().Get("search")
	connector1Str := r.URL.Query().Get("connector1")
	connector2Str := r.URL.Query().Get("connector2")
	typeStr := r.URL.Query().Get("type")
	lengthMinStr := r.URL.Query().Get("length_min")
	lengthMaxStr := r.URL.Query().Get("length_max")

	query := `
		SELECT
			c.cableID,
			c.name,
			c.connector1,
			c.connector2,
			c.typ,
			c.length,
			c.mm2,
			cc1.name as connector1_name,
			cc1.gender as connector1_gender,
			cc2.name as connector2_name,
			cc2.gender as connector2_gender,
			ct.name as cable_type_name
		FROM cables c
		LEFT JOIN cable_connectors cc1 ON c.connector1 = cc1.cable_connectorsID
		LEFT JOIN cable_connectors cc2 ON c.connector2 = cc2.cable_connectorsID
		LEFT JOIN cable_types ct ON c.typ = ct.cable_typesID
		WHERE 1=1
	`

	var args []interface{}
	paramCount := 0

	if search != "" {
		paramCount++
		query += fmt.Sprintf(" AND (c.name ILIKE $%[1]d OR cc1.name ILIKE $%[1]d OR cc2.name ILIKE $%[1]d OR ct.name ILIKE $%[1]d)", paramCount)
		args = append(args, "%"+search+"%")
	}

	if connector1Str != "" {
		if id, err := strconv.Atoi(connector1Str); err == nil {
			paramCount++
			query += fmt.Sprintf(" AND c.connector1 = $%d", paramCount)
			args = append(args, id)
		}
	}

	if connector2Str != "" {
		if id, err := strconv.Atoi(connector2Str); err == nil {
			paramCount++
			query += fmt.Sprintf(" AND c.connector2 = $%d", paramCount)
			args = append(args, id)
		}
	}

	if typeStr != "" {
		if id, err := strconv.Atoi(typeStr); err == nil {
			paramCount++
			query += fmt.Sprintf(" AND c.typ = $%d", paramCount)
			args = append(args, id)
		}
	}

	if lengthMinStr != "" {
		if val, err := strconv.ParseFloat(lengthMinStr, 64); err == nil {
			paramCount++
			query += fmt.Sprintf(" AND c.length >= $%d", paramCount)
			args = append(args, val)
		}
	}

	if lengthMaxStr != "" {
		if val, err := strconv.ParseFloat(lengthMaxStr, 64); err == nil {
			paramCount++
			query += fmt.Sprintf(" AND c.length <= $%d", paramCount)
			args = append(args, val)
		}
	}

	query += " ORDER BY c.name, c.cableID"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying cables: %v", err)
		http.Error(w, "Failed to fetch cables", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cables []Cable
	for rows.Next() {
		var cable Cable
		err := rows.Scan(
			&cable.CableID,
			&cable.Name,
			&cable.Connector1,
			&cable.Connector2,
			&cable.Typ,
			&cable.Length,
			&cable.MM2,
			&cable.Connector1Name,
			&cable.Connector1Gender,
			&cable.Connector2Name,
			&cable.Connector2Gender,
			&cable.CableTypeName,
		)
		if err != nil {
			log.Printf("Error scanning cable: %v", err)
			continue
		}
		cables = append(cables, cable)
	}

	if cables == nil {
		cables = []Cable{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cables)
}

// GetCable retrieves a single cable by ID
func GetCable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid cable ID", http.StatusBadRequest)
		return
	}

	db := repository.GetSQLDB()

	query := `
		SELECT
			c.cableID,
			c.name,
			c.connector1,
			c.connector2,
			c.typ,
			c.length,
			c.mm2,
			cc1.name as connector1_name,
			cc1.gender as connector1_gender,
			cc2.name as connector2_name,
			cc2.gender as connector2_gender,
			ct.name as cable_type_name
		FROM cables c
		LEFT JOIN cable_connectors cc1 ON c.connector1 = cc1.cable_connectorsID
		LEFT JOIN cable_connectors cc2 ON c.connector2 = cc2.cable_connectorsID
		LEFT JOIN cable_types ct ON c.typ = ct.cable_typesID
		WHERE c.cableID = $1
	`

	var cable Cable
	err = db.QueryRow(query, id).Scan(
		&cable.CableID,
		&cable.Name,
		&cable.Connector1,
		&cable.Connector2,
		&cable.Typ,
		&cable.Length,
		&cable.MM2,
		&cable.Connector1Name,
		&cable.Connector1Gender,
		&cable.Connector2Name,
		&cable.Connector2Gender,
		&cable.CableTypeName,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Cable not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error fetching cable: %v", err)
		http.Error(w, "Failed to fetch cable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cable)
}

// CreateCable creates a new cable
func CreateCable(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name       *string  `json:"name"`
		Connector1 int      `json:"connector1"`
		Connector2 int      `json:"connector2"`
		Typ        int      `json:"typ"`
		Length     float64  `json:"length"`
		MM2        *float64 `json:"mm2"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if input.Length <= 0 {
		http.Error(w, "Length must be greater than 0", http.StatusBadRequest)
		return
	}
	if input.Connector1 <= 0 || input.Connector2 <= 0 || input.Typ <= 0 {
		http.Error(w, "Connector1, Connector2, and Type are required", http.StatusBadRequest)
		return
	}

	db := repository.GetSQLDB()

	var id int64
	query := `INSERT INTO cables (connector1, connector2, typ, length, mm2, name) VALUES ($1, $2, $3, $4, $5, $6) RETURNING cable_id`
	err := db.QueryRow(query, input.Connector1, input.Connector2, input.Typ, input.Length, input.MM2, input.Name).Scan(&id)
	if err != nil {
		log.Printf("Error creating cable: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create cable: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[CABLE CREATE] Created cable ID %d", id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cable_id": id,
		"message":  "Cable created successfully",
	})
}

// UpdateCable updates an existing cable
func UpdateCable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid cable ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Name       *string  `json:"name"`
		Connector1 *int     `json:"connector1"`
		Connector2 *int     `json:"connector2"`
		Typ        *int     `json:"typ"`
		Length     *float64 `json:"length"`
		MM2        *float64 `json:"mm2"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if input.Length != nil && *input.Length <= 0 {
		http.Error(w, "Length must be greater than 0", http.StatusBadRequest)
		return
	}

	db := repository.GetSQLDB()

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}

	paramCount := 0
	if input.Name != nil {
		paramCount++
		updates = append(updates, fmt.Sprintf("name = $%d", paramCount))
		args = append(args, input.Name)
	}
	if input.Connector1 != nil {
		paramCount++
		updates = append(updates, fmt.Sprintf("connector1 = $%d", paramCount))
		args = append(args, *input.Connector1)
	}
	if input.Connector2 != nil {
		paramCount++
		updates = append(updates, fmt.Sprintf("connector2 = $%d", paramCount))
		args = append(args, *input.Connector2)
	}
	if input.Typ != nil {
		paramCount++
		updates = append(updates, fmt.Sprintf("typ = $%d", paramCount))
		args = append(args, *input.Typ)
	}
	if input.Length != nil {
		paramCount++
		updates = append(updates, fmt.Sprintf("length = $%d", paramCount))
		args = append(args, *input.Length)
	}
	if input.MM2 != nil {
		paramCount++
		updates = append(updates, fmt.Sprintf("mm2 = $%d", paramCount))
		args = append(args, *input.MM2)
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	paramCount++
	query := fmt.Sprintf("UPDATE cables SET %s WHERE cableID = $%d", strings.Join(updates, ", "), paramCount)
	args = append(args, id)

	result, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating cable: %v", err)
		http.Error(w, "Failed to update cable", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Cable not found", http.StatusNotFound)
		return
	}

	log.Printf("[CABLE UPDATE] Updated cable ID %d", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Cable updated successfully"})
}

// DeleteCable deletes a cable
func DeleteCable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid cable ID", http.StatusBadRequest)
		return
	}

	db := repository.GetSQLDB()

	result, err := db.Exec("DELETE FROM cables WHERE cableID = $1", id)
	if err != nil {
		log.Printf("Error deleting cable: %v", err)
		http.Error(w, "Failed to delete cable", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Cable not found", http.StatusNotFound)
		return
	}

	log.Printf("[CABLE DELETE] Deleted cable ID %d", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Cable deleted successfully"})
}

// GetCableConnectors retrieves all cable connector types
func GetCableConnectors(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	query := `SELECT cable_connectorsID, name, abbreviation, gender FROM cable_connectors ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying cable connectors: %v", err)
		http.Error(w, "Failed to fetch cable connectors", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var connectors []CableConnector
	for rows.Next() {
		var connector CableConnector
		err := rows.Scan(&connector.ConnectorID, &connector.Name, &connector.Abbreviation, &connector.Gender)
		if err != nil {
			log.Printf("Error scanning connector: %v", err)
			continue
		}
		connectors = append(connectors, connector)
	}

	if connectors == nil {
		connectors = []CableConnector{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(connectors)
}

// GetCableTypes retrieves all cable types
func GetCableTypes(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			ct.cable_typesID,
			ct.name,
			COALESCE(COUNT(c.cableID), 0) AS cable_count
		FROM cable_types ct
		LEFT JOIN cables c ON c.typ = ct.cable_typesID
		GROUP BY ct.cable_typesID, ct.name
		ORDER BY ct.name
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying cable types: %v", err)
		http.Error(w, "Failed to fetch cable types", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var types []CableType
	for rows.Next() {
		var cableType CableType
		err := rows.Scan(&cableType.CableTypeID, &cableType.Name, &cableType.Count)
		if err != nil {
			log.Printf("Error scanning cable type: %v", err)
			continue
		}
		types = append(types, cableType)
	}

	if types == nil {
		types = []CableType{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

// GetCableDevices retrieves all devices associated with a cable
func GetCableDevices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cableID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid cable ID"})
		return
	}

	db := repository.GetSQLDB()

	query := `
		WITH latest_job AS (
			SELECT jd.deviceID, MAX(jd.jobID) AS jobID
			FROM jobdevices jd
			GROUP BY jd.deviceID
		)
		SELECT d.deviceID, d.productID, d.serialnumber, d.barcode, d.qr_code, d.rfid, d.status,
		       d.current_location, d.zone_id,
		       COALESCE(d.condition_rating, 0), COALESCE(d.usage_hours, 0), d.purchaseDate, d.retire_date, d.warranty_end_date,
		       d.lastmaintenance, d.nextmaintenance,
		       d.notes, d.label_path,
		       COALESCE(p.name, '') AS product_name,
		       COALESCE(cat.name, '') AS product_category,
		       COALESCE(z.name, '') AS zone_name,
		       COALESCE(z.code, '') AS zone_code,
		       dc.caseID,
		       COALESCE(cs.name, '') AS case_name,
		       d.cable_id,
		       COALESCE(cab.name, '') AS cable_name,
		       lj.jobID,
		       COALESCE(CAST(lj.jobID AS TEXT), '') AS job_number
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN categories cat ON p.categoryID = cat.categoryID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		LEFT JOIN devicescases dc ON d.deviceID = dc.deviceID
		LEFT JOIN cases cs ON dc.caseID = cs.caseID
		LEFT JOIN cables cab ON d.cable_id = cab.cableID
		LEFT JOIN latest_job lj ON lj.deviceID = d.deviceID
		WHERE d.cable_id = $1
		ORDER BY d.deviceID ASC
	`

	rows, err := db.Query(query, cableID)
	if err != nil {
		log.Printf("[CABLE DEVICES] Failed to query devices for cable %d: %v", cableID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch cable devices"})
		return
	}
	defer rows.Close()

	var responses []DeviceAdminResponse
	for rows.Next() {
		var device models.DeviceWithDetails
		err := rows.Scan(
			&device.DeviceID,
			&device.ProductID,
			&device.SerialNumber,
			&device.Barcode,
			&device.QRCode,
			&device.RFID,
			&device.Status,
			&device.CurrentLocation,
			&device.ZoneID,
			&device.ConditionRating,
			&device.UsageHours,
			&device.PurchaseDate,
			&device.RetireDate,
			&device.WarrantyEndDate,
			&device.LastMaintenance,
			&device.NextMaintenance,
			&device.Notes,
			&device.LabelPath,
			&device.ProductName,
			&device.ProductCategory,
			&device.ZoneName,
			&device.ZoneCode,
			&device.CaseID,
			&device.CaseName,
			&device.CableID,
			&device.CableName,
			&device.CurrentJobID,
			&device.JobNumber,
		)
		if err != nil {
			log.Printf("[CABLE DEVICES] Failed to scan device: %v", err)
			continue
		}

		responses = append(responses, toDeviceAdminResponse(&device))
	}

	respondJSON(w, http.StatusOK, responses)
}

// CreateDevicesForCable creates one or more devices linked to a cable.
// The caller must supply a prefix; device IDs are generated as PREFIX001, PREFIX002, etc.
func CreateDevicesForCable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cableID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid cable ID"})
		return
	}

	var req struct {
		Quantity int    `json:"quantity"`
		Prefix   string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Quantity <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Valid quantity is required"})
		return
	}
	if req.Quantity > 100 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Cannot create more than 100 devices at once"})
		return
	}

	prefix := strings.ToUpper(strings.TrimSpace(req.Prefix))
	if prefix == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Prefix is required for cable devices"})
		return
	}

	db := repository.GetSQLDB()

	// Verify cable exists
	var exists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM cables WHERE cableID = $1)", cableID).Scan(&exists); err != nil || !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Cable not found"})
		return
	}

	// Use DB-side approach to find the next available counter for the prefix
	var createdIDs []string
	for i := 0; i < req.Quantity; i++ {
		inserted := false
		for attempt := 0; attempt < 1000; attempt++ {
			var nextCounter int
			err := db.QueryRow(`
				SELECT gs
				FROM generate_series(1, 999) AS gs
				WHERE NOT EXISTS (
					SELECT 1
					FROM devices
					WHERE deviceID = $1 || LPAD(gs::text, 3, '0')
				)
				ORDER BY gs
				LIMIT 1
			`, prefix).Scan(&nextCounter)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Printf("[CABLE DEVICE CREATE] No available device IDs remain for prefix %s", prefix)
				} else {
					log.Printf("[CABLE DEVICE CREATE] Failed to find next device ID for prefix %s: %v", prefix, err)
				}
				break
			}

			candidateID := fmt.Sprintf("%s%03d", prefix, nextCounter)
			_, err = db.Exec(
				"INSERT INTO devices (deviceID, cable_id, status) VALUES ($1, $2, 'in_storage')",
				candidateID, cableID,
			)
			if err != nil {
				var pqErr *pq.Error
				if errors.As(err, &pqErr) && pqErr.Code == "23505" { // unique_violation
					continue
				}
				log.Printf("[CABLE DEVICE CREATE] Failed to insert device %s: %v", candidateID, err)
				break
			}
			createdIDs = append(createdIDs, candidateID)
			inserted = true
			break
		}
		if !inserted {
			log.Printf("[CABLE DEVICE CREATE] Could not insert device %d/%d for cable %d", i+1, req.Quantity, cableID)
		}
	}

	if len(createdIDs) == 0 {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create any devices"})
		return
	}

	log.Printf("[CABLE DEVICE CREATE] Created %d devices for cable %d: %v", len(createdIDs), cableID, createdIDs)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"created_count": len(createdIDs),
		"device_ids":    createdIDs,
	})
}
