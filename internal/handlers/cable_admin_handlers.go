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

	if search != "" {
		query += " AND (c.name LIKE ? OR cc1.name LIKE ? OR cc2.name LIKE ? OR ct.name LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if connector1Str != "" {
		if id, err := strconv.Atoi(connector1Str); err == nil {
			query += " AND c.connector1 = ?"
			args = append(args, id)
		}
	}

	if connector2Str != "" {
		if id, err := strconv.Atoi(connector2Str); err == nil {
			query += " AND c.connector2 = ?"
			args = append(args, id)
		}
	}

	if typeStr != "" {
		if id, err := strconv.Atoi(typeStr); err == nil {
			query += " AND c.typ = ?"
			args = append(args, id)
		}
	}

	if lengthMinStr != "" {
		if val, err := strconv.ParseFloat(lengthMinStr, 64); err == nil {
			query += " AND c.length >= ?"
			args = append(args, val)
		}
	}

	if lengthMaxStr != "" {
		if val, err := strconv.ParseFloat(lengthMaxStr, 64); err == nil {
			query += " AND c.length <= ?"
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
		WHERE c.cableID = ?
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

	query := `INSERT INTO cables (connector1, connector2, typ, length, mm2, name) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := db.Exec(query, input.Connector1, input.Connector2, input.Typ, input.Length, input.MM2, input.Name)
	if err != nil {
		log.Printf("Error creating cable: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create cable: %v", err), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

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

	if input.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, input.Name)
	}
	if input.Connector1 != nil {
		updates = append(updates, "connector1 = ?")
		args = append(args, *input.Connector1)
	}
	if input.Connector2 != nil {
		updates = append(updates, "connector2 = ?")
		args = append(args, *input.Connector2)
	}
	if input.Typ != nil {
		updates = append(updates, "typ = ?")
		args = append(args, *input.Typ)
	}
	if input.Length != nil {
		updates = append(updates, "length = ?")
		args = append(args, *input.Length)
	}
	if input.MM2 != nil {
		updates = append(updates, "mm2 = ?")
		args = append(args, *input.MM2)
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf("UPDATE cables SET %s WHERE cableID = ?", strings.Join(updates, ", "))
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

	query := "DELETE FROM cables WHERE cableID = ?"
	result, err := db.Exec(query, id)
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
