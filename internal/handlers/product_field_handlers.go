package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	"warehousecore/internal/repository"
)

// ProductFieldDefinition represents a dynamic field definition for products
type ProductFieldDefinition struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Label      string  `json:"label"`
	FieldType  string  `json:"field_type"`
	Options    *string `json:"options"`
	Unit       *string `json:"unit"`
	SortOrder  int     `json:"sort_order"`
	IsRequired bool    `json:"is_required"`
}

// ProductFieldValue represents a field definition combined with its value for a specific product
type ProductFieldValue struct {
	FieldDefinitionID int     `json:"field_definition_id"`
	Name              string  `json:"name"`
	Label             string  `json:"label"`
	FieldType         string  `json:"field_type"`
	Options           *string `json:"options"`
	Unit              *string `json:"unit"`
	SortOrder         int     `json:"sort_order"`
	Value             string  `json:"value"`
}

var (
	validFieldNameRe  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,98}[a-z0-9]$|^[a-z]$`)
	validFieldTypes   = map[string]bool{"text": true, "number": true, "integer": true, "select": true, "boolean": true}
)

// GetProductFieldDefinitions retrieves all product field definitions ordered by sort_order
func GetProductFieldDefinitions(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()

	rows, err := db.Query(`
		SELECT id, name, label, field_type, options, unit, sort_order, is_required
		FROM product_field_definitions
		ORDER BY sort_order, id
	`)
	if err != nil {
		log.Printf("Error querying product field definitions: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch field definitions"})
		return
	}
	defer rows.Close()

	definitions := []ProductFieldDefinition{}
	for rows.Next() {
		var d ProductFieldDefinition
		if err := rows.Scan(&d.ID, &d.Name, &d.Label, &d.FieldType, &d.Options, &d.Unit, &d.SortOrder, &d.IsRequired); err != nil {
			log.Printf("Error scanning product field definition: %v", err)
			continue
		}
		definitions = append(definitions, d)
	}

	respondJSON(w, http.StatusOK, definitions)
}

// CreateProductFieldDefinition creates a new product field definition
func CreateProductFieldDefinition(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name       string  `json:"name"`
		Label      string  `json:"label"`
		FieldType  string  `json:"field_type"`
		Options    *string `json:"options"`
		Unit       *string `json:"unit"`
		SortOrder  int     `json:"sort_order"`
		IsRequired bool    `json:"is_required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if !validFieldNameRe.MatchString(input.Name) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Field name must start with a lowercase letter and contain only lowercase letters, digits, and underscores"})
		return
	}
	if input.Label == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Label is required"})
		return
	}
	if !validFieldTypes[input.FieldType] {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "field_type must be one of: text, number, integer, select, boolean"})
		return
	}

	db := repository.GetSQLDB()

	var d ProductFieldDefinition
	err := db.QueryRow(`
		INSERT INTO product_field_definitions (name, label, field_type, options, unit, sort_order, is_required)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, label, field_type, options, unit, sort_order, is_required
	`, input.Name, input.Label, input.FieldType, input.Options, input.Unit, input.SortOrder, input.IsRequired).
		Scan(&d.ID, &d.Name, &d.Label, &d.FieldType, &d.Options, &d.Unit, &d.SortOrder, &d.IsRequired)
	if err != nil {
		log.Printf("Error creating product field definition: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create field definition"})
		return
	}

	respondJSON(w, http.StatusCreated, d)
}

// UpdateProductFieldDefinition updates an existing product field definition by ID
func UpdateProductFieldDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid field definition ID"})
		return
	}

	var input struct {
		Name       string  `json:"name"`
		Label      string  `json:"label"`
		FieldType  string  `json:"field_type"`
		Options    *string `json:"options"`
		Unit       *string `json:"unit"`
		SortOrder  int     `json:"sort_order"`
		IsRequired bool    `json:"is_required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if !validFieldNameRe.MatchString(input.Name) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Field name must start with a lowercase letter and contain only lowercase letters, digits, and underscores"})
		return
	}
	if input.Label == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Label is required"})
		return
	}
	if !validFieldTypes[input.FieldType] {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "field_type must be one of: text, number, integer, select, boolean"})
		return
	}

	db := repository.GetSQLDB()

	var d ProductFieldDefinition
	err = db.QueryRow(`
		UPDATE product_field_definitions
		SET name=$1, label=$2, field_type=$3, options=$4, unit=$5, sort_order=$6, is_required=$7
		WHERE id=$8
		RETURNING id, name, label, field_type, options, unit, sort_order, is_required
	`, input.Name, input.Label, input.FieldType, input.Options, input.Unit, input.SortOrder, input.IsRequired, id).
		Scan(&d.ID, &d.Name, &d.Label, &d.FieldType, &d.Options, &d.Unit, &d.SortOrder, &d.IsRequired)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Field definition not found"})
		return
	}
	if err != nil {
		log.Printf("Error updating product field definition %d: %v", id, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update field definition"})
		return
	}

	respondJSON(w, http.StatusOK, d)
}

// DeleteProductFieldDefinition deletes a product field definition by ID
func DeleteProductFieldDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid field definition ID"})
		return
	}

	db := repository.GetSQLDB()

	result, err := db.Exec(`DELETE FROM product_field_definitions WHERE id=$1`, id)
	if err != nil {
		log.Printf("Error deleting product field definition %d: %v", id, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete field definition"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for field definition delete %d: %v", id, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify deletion"})
		return
	}
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Field definition not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Field definition deleted"})
}

// GetProductFieldValues retrieves all field definitions with their values for a specific product
func GetProductFieldValues(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	db := repository.GetSQLDB()

	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM products WHERE id=$1)`, productID).Scan(&exists); err != nil {
		log.Printf("Error checking product existence %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify product"})
		return
	}
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	rows, err := db.Query(`
		SELECT
			d.id,
			d.name,
			d.label,
			d.field_type,
			d.options,
			d.unit,
			d.sort_order,
			COALESCE(v.value, '') AS value
		FROM product_field_definitions d
		LEFT JOIN product_field_values v ON v.field_definition_id = d.id AND v.product_id = $1
		ORDER BY d.sort_order, d.id
	`, productID)
	if err != nil {
		log.Printf("Error querying product field values for product %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch field values"})
		return
	}
	defer rows.Close()

	values := []ProductFieldValue{}
	for rows.Next() {
		var v ProductFieldValue
		if err := rows.Scan(&v.FieldDefinitionID, &v.Name, &v.Label, &v.FieldType, &v.Options, &v.Unit, &v.SortOrder, &v.Value); err != nil {
			log.Printf("Error scanning product field value: %v", err)
			continue
		}
		values = append(values, v)
	}

	respondJSON(w, http.StatusOK, values)
}

// SetProductFieldValues upserts field values for a specific product
func SetProductFieldValues(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	var body struct {
		Values map[string]string `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	db := repository.GetSQLDB()

	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM products WHERE id=$1)`, productID).Scan(&exists); err != nil {
		log.Printf("Error checking product existence %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify product"})
		return
	}
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	for fieldName, value := range body.Values {
		var fieldDefinitionID int
		err := db.QueryRow(`SELECT id FROM product_field_definitions WHERE name=$1`, fieldName).Scan(&fieldDefinitionID)
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Unknown field name: " + fieldName})
			return
		}
		if err != nil {
			log.Printf("Error looking up field definition for name %q: %v", fieldName, err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to look up field definition"})
			return
		}

		if value == "" {
			if _, err := db.Exec(`DELETE FROM product_field_values WHERE product_id=$1 AND field_definition_id=$2`, productID, fieldDefinitionID); err != nil {
				log.Printf("Error deleting field value for product %d, definition %d: %v", productID, fieldDefinitionID, err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete field value"})
				return
			}
		} else {
			if _, err := db.Exec(`
				INSERT INTO product_field_values (product_id, field_definition_id, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (product_id, field_definition_id) DO UPDATE SET value = EXCLUDED.value
			`, productID, fieldDefinitionID, value); err != nil {
				log.Printf("Error upserting field value for product %d, definition %d: %v", productID, fieldDefinitionID, err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to set field value"})
				return
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Field values updated"})
}
