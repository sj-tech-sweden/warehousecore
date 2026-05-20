package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"warehousecore/internal/repository"
)

// Brand represents a product brand with optional manufacturer association.
type Brand struct {
	BrandID          int     `json:"brand_id"`
	Name             string  `json:"name"`
	ManufacturerID   *int    `json:"manufacturer_id,omitempty"`
	ManufacturerName *string `json:"manufacturer_name,omitempty"`
}

// Manufacturer represents a product manufacturer.
type Manufacturer struct {
	ManufacturerID int     `json:"manufacturer_id"`
	Name           string  `json:"name"`
	Website        *string `json:"website,omitempty"`
}

// GetBrands returns all brands with manufacturer metadata.
func GetBrands(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	// Inspect available columns for brands table
	cols := map[string]bool{}
	colRows, err := db.Query("SELECT column_name FROM information_schema.columns WHERE table_name = $1", "brands")
	if err == nil {
		defer colRows.Close()
		for colRows.Next() {
			var cn string
			if err := colRows.Scan(&cn); err == nil {
				cols[cn] = true
			}
		}
	}

	// Determine column names
	idCol := "brandid"
	if cols["brand_id"] {
		idCol = "brand_id"
	} else if cols["id"] {
		idCol = "id"
	}
	nameCol := "name"
	manufCol := ""
	if cols["manufacturerid"] {
		manufCol = "manufacturerid"
	} else if cols["manufacturer_id"] {
		manufCol = "manufacturer_id"
	}

	// Build query dynamically
	query := fmt.Sprintf("SELECT %s, %s", idCol, nameCol)
	if manufCol != "" {
		query = fmt.Sprintf("%s, %s", query, manufCol)
	}
	query = fmt.Sprintf("%s FROM brands ORDER BY %s", query, nameCol)

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[BRAND] failed to query brands: %v; query=%s", err, query)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch brands"})
		return
	}
	defer rows.Close()

	var brands []Brand
	for rows.Next() {
		if manufCol != "" {
			var (
				brandID        sql.NullInt64
				name           sql.NullString
				manufacturerID sql.NullInt64
			)
			if err := rows.Scan(&brandID, &name, &manufacturerID); err != nil {
				log.Printf("[BRAND] scan error: %v", err)
				continue
			}
			b := Brand{}
			if brandID.Valid {
				b.BrandID = int(brandID.Int64)
			}
			if name.Valid {
				b.Name = name.String
			}
			if manufacturerID.Valid {
				id := int(manufacturerID.Int64)
				b.ManufacturerID = &id
				if mName := lookupManufacturerName(db, manufacturerID.Int64); mName.Valid {
					v := mName.String
					b.ManufacturerName = &v
				}
			}
			brands = append(brands, b)
		} else {
			var (
				brandID int
				name    string
			)
			if err := rows.Scan(&brandID, &name); err != nil {
				log.Printf("[BRAND] scan error: %v", err)
				continue
			}
			brands = append(brands, Brand{BrandID: brandID, Name: name})
		}
	}

	respondJSON(w, http.StatusOK, brands)
}

// CreateBrand creates a new brand.
func CreateBrand(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name           string `json:"name"`
		ManufacturerID *int   `json:"manufacturer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if payload.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name is required"})
		return
	}

	db := repository.GetSQLDB()
	var id int64
	err := db.QueryRow(
		"INSERT INTO brands (name, manufacturerID) VALUES ($1, $2) RETURNING brandID",
		payload.Name,
		payload.ManufacturerID,
	).Scan(&id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create brand"})
		return
	}
	brand := Brand{
		BrandID: int(id),
		Name:    payload.Name,
	}

	if payload.ManufacturerID != nil {
		brand.ManufacturerID = payload.ManufacturerID

		var manufacturerName sql.NullString
		if err := db.QueryRow(
			"SELECT name FROM manufacturer WHERE manufacturerID = $1",
			*payload.ManufacturerID,
		).Scan(&manufacturerName); err == nil && manufacturerName.Valid {
			name := manufacturerName.String
			brand.ManufacturerName = &name
		}
	}

	respondJSON(w, http.StatusCreated, brand)
}

// UpdateBrand updates an existing brand.
func UpdateBrand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid brand ID"})
		return
	}

	var payload struct {
		Name           string `json:"name"`
		ManufacturerID *int   `json:"manufacturer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if payload.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name is required"})
		return
	}

	db := repository.GetSQLDB()
	result, err := db.Exec(
		"UPDATE brands SET name = $1, manufacturerID = $2 WHERE brandID = $3",
		payload.Name,
		payload.ManufacturerID,
		id,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update brand"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Brand not found"})
		return
	}

	response := Brand{
		BrandID: id,
		Name:    payload.Name,
	}
	if payload.ManufacturerID != nil {
		response.ManufacturerID = payload.ManufacturerID

		var manufacturerName sql.NullString
		if err := db.QueryRow(
			"SELECT name FROM manufacturer WHERE manufacturerID = $1",
			*payload.ManufacturerID,
		).Scan(&manufacturerName); err == nil && manufacturerName.Valid {
			name := manufacturerName.String
			response.ManufacturerName = &name
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// DeleteBrand removes a brand.
func DeleteBrand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid brand ID"})
		return
	}

	db := repository.GetSQLDB()
	result, err := db.Exec("DELETE FROM brands WHERE brandID = $1", id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete brand"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Brand not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Brand deleted successfully"})
}

// GetManufacturers returns all manufacturers.
func GetManufacturers(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	// Detect whether the manufacturer table has a website column
	hasWebsite := false
	colRows, err := db.Query("SELECT column_name FROM information_schema.columns WHERE table_name = $1", "manufacturer")
	if err == nil {
		defer colRows.Close()
		for colRows.Next() {
			var cn string
			if err := colRows.Scan(&cn); err == nil && cn == "website" {
				hasWebsite = true
			}
		}
	}

	query := "SELECT manufacturerID, name"
	if hasWebsite {
		query += ", website"
	}
	query += " FROM manufacturer ORDER BY name"

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[BRAND] failed to query manufacturers: %v; query=%s", err, query)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch manufacturers"})
		return
	}
	defer rows.Close()

	var manufacturers []Manufacturer
	for rows.Next() {
		if hasWebsite {
			var (
				manufacturerID int
				name           string
				website        sql.NullString
			)
			if err := rows.Scan(&manufacturerID, &name, &website); err != nil {
				log.Printf("[BRAND] scan error (manufacturers): %v", err)
				continue
			}
			manufacturer := Manufacturer{ManufacturerID: manufacturerID, Name: name}
			if website.Valid {
				v := website.String
				manufacturer.Website = &v
			}
			manufacturers = append(manufacturers, manufacturer)
		} else {
			var (
				manufacturerID int
				name           string
			)
			if err := rows.Scan(&manufacturerID, &name); err != nil {
				log.Printf("[BRAND] scan error (manufacturers): %v", err)
				continue
			}
			manufacturers = append(manufacturers, Manufacturer{ManufacturerID: manufacturerID, Name: name})
		}
	}

	respondJSON(w, http.StatusOK, manufacturers)
}

// lookupManufacturerName tries several id column variants to find the manufacturer's name.
func lookupManufacturerName(db *sql.DB, id int64) sql.NullString {
	var name sql.NullString
	queries := []string{
		"SELECT name FROM manufacturer WHERE manufacturerID = $1",
		"SELECT name FROM manufacturer WHERE manufacturer_id = $1",
		"SELECT name FROM manufacturer WHERE id = $1",
	}
	for _, q := range queries {
		if err := db.QueryRow(q, id).Scan(&name); err == nil {
			return name
		} else if err == sql.ErrNoRows {
			// not found for this key, continue to next variant
			continue
		} else {
			// other errors (e.g., column doesn't exist) -> try next variant
			continue
		}
	}
	return sql.NullString{}
}

// CreateManufacturer creates a new manufacturer.
func CreateManufacturer(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name    string  `json:"name"`
		Website *string `json:"website"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if payload.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name is required"})
		return
	}

	db := repository.GetSQLDB()
	var id int64
	err := db.QueryRow(
		"INSERT INTO manufacturer (name, website) VALUES ($1, $2) RETURNING manufacturerID",
		payload.Name,
		payload.Website,
	).Scan(&id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create manufacturer"})
		return
	}
	manufacturer := Manufacturer{
		ManufacturerID: int(id),
		Name:           payload.Name,
		Website:        payload.Website,
	}

	respondJSON(w, http.StatusCreated, manufacturer)
}

// UpdateManufacturer updates an existing manufacturer.
func UpdateManufacturer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid manufacturer ID"})
		return
	}

	var payload struct {
		Name    string  `json:"name"`
		Website *string `json:"website"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if payload.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name is required"})
		return
	}

	db := repository.GetSQLDB()
	result, err := db.Exec(
		"UPDATE manufacturer SET name = $1, website = $2 WHERE manufacturerID = $3",
		payload.Name,
		payload.Website,
		id,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update manufacturer"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Manufacturer not found"})
		return
	}

	respondJSON(w, http.StatusOK, Manufacturer{
		ManufacturerID: id,
		Name:           payload.Name,
		Website:        payload.Website,
	})
}

// DeleteManufacturer removes a manufacturer.
func DeleteManufacturer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid manufacturer ID"})
		return
	}

	db := repository.GetSQLDB()
	result, err := db.Exec("DELETE FROM manufacturer WHERE manufacturerID = $1", id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete manufacturer"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Manufacturer not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Manufacturer deleted successfully"})
}
