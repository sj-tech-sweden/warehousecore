package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"warehousecore/internal/repository"
)

// CountType represents a measurement unit for accessories/consumables
type CountType struct {
	CountTypeID  int    `json:"count_type_id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	IsActive     bool   `json:"is_active"`
}

// ListCountTypes returns all measurement units.
func ListCountTypes(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	rows, err := db.Query("SELECT count_type_id, name, abbreviation, is_active FROM count_types ORDER BY name")
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch measurement units"})
		return
	}
	defer rows.Close()

	var countTypes []CountType
	for rows.Next() {
		var ct CountType
		if err := rows.Scan(&ct.CountTypeID, &ct.Name, &ct.Abbreviation, &ct.IsActive); err == nil {
			countTypes = append(countTypes, ct)
		}
	}

	respondJSON(w, http.StatusOK, countTypes)
}

type countTypeRequest struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

// CreateCountType adds a new measurement unit.
func CreateCountType(w http.ResponseWriter, r *http.Request) {
	var req countTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Abbreviation = strings.TrimSpace(req.Abbreviation)
	if req.Name == "" || req.Abbreviation == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name and abbreviation are required"})
		return
	}
	if len(req.Abbreviation) > 10 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Abbreviation must be at most 10 characters"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	db := repository.GetSQLDB()
	result, err := db.Exec(
		"INSERT INTO count_types (name, abbreviation, is_active) VALUES (?, ?, ?)",
		req.Name, req.Abbreviation, isActive,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create measurement unit"})
		return
	}

	id, _ := result.LastInsertId()
	respondJSON(w, http.StatusCreated, CountType{
		CountTypeID:  int(id),
		Name:         req.Name,
		Abbreviation: req.Abbreviation,
		IsActive:     isActive,
	})
}

// UpdateCountType updates an existing measurement unit.
func UpdateCountType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil || id <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid measurement unit ID"})
		return
	}

	var req countTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Abbreviation = strings.TrimSpace(req.Abbreviation)
	if req.Name == "" || req.Abbreviation == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name and abbreviation are required"})
		return
	}
	if len(req.Abbreviation) > 10 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Abbreviation must be at most 10 characters"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	db := repository.GetSQLDB()
	result, err := db.Exec(
		"UPDATE count_types SET name = ?, abbreviation = ?, is_active = ? WHERE count_type_id = ?",
		req.Name, req.Abbreviation, isActive, id,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update measurement unit"})
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Measurement unit not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Measurement unit updated"})
}

// DeleteCountType removes a measurement unit.
func DeleteCountType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil || id <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid measurement unit ID"})
		return
	}

	db := repository.GetSQLDB()
	result, err := db.Exec("DELETE FROM count_types WHERE count_type_id = ?", id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete measurement unit"})
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Measurement unit not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Measurement unit deleted"})
}
