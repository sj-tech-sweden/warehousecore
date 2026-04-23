package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
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
	validFieldNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,98}[a-z0-9]$|^[a-z]$`)
	validFieldTypes  = map[string]bool{"text": true, "number": true, "integer": true, "select": true, "boolean": true}
)

// validateFieldOptions checks that options is valid JSON string array for 'select' type,
// clears it for other types, and errors on invalid JSON for 'select'.
func validateFieldOptions(fieldType string, options *string) (*string, error) {
	if fieldType != "select" {
		// non-select fields must not store options
		return nil, nil
	}
	if options == nil || *options == "" {
		empty := "[]"
		return &empty, nil
	}
	var parsed []interface{}
	if err := json.Unmarshal([]byte(*options), &parsed); err != nil {
		return nil, fmt.Errorf("options must be a valid JSON array of strings")
	}
	return options, nil
}

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
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating product field definitions: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch field definitions"})
		return
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

	validatedOptions, err := validateFieldOptions(input.FieldType, input.Options)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	db := repository.GetSQLDB()

	var d ProductFieldDefinition
	err = db.QueryRow(`
		INSERT INTO product_field_definitions (name, label, field_type, options, unit, sort_order, is_required)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, label, field_type, options, unit, sort_order, is_required
	`, input.Name, input.Label, input.FieldType, validatedOptions, input.Unit, input.SortOrder, input.IsRequired).
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

	validatedOptions, err := validateFieldOptions(input.FieldType, input.Options)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	db := repository.GetSQLDB()

	var d ProductFieldDefinition
	err = db.QueryRow(`
		UPDATE product_field_definitions
		SET name=$1, label=$2, field_type=$3, options=$4, unit=$5, sort_order=$6, is_required=$7
		WHERE id=$8
		RETURNING id, name, label, field_type, options, unit, sort_order, is_required
	`, input.Name, input.Label, input.FieldType, validatedOptions, input.Unit, input.SortOrder, input.IsRequired, id).
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
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM products WHERE productID=$1)`, productID).Scan(&exists); err != nil {
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
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating product field values for product %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch field values"})
		return
	}

	respondJSON(w, http.StatusOK, values)
}

// SetProductFieldValues upserts field values for a specific product atomically.
// All field names are resolved in a single query; updates/deletes run inside a transaction.
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

	if len(body.Values) == 0 {
		respondJSON(w, http.StatusOK, map[string]string{"message": "Field values updated"})
		return
	}

	db := repository.GetSQLDB()

	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM products WHERE productID=$1)`, productID).Scan(&exists); err != nil {
		log.Printf("Error checking product existence %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify product"})
		return
	}
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	// Collect all field names to resolve in one query
	names := make([]string, 0, len(body.Values))
	for k := range body.Values {
		names = append(names, k)
	}

	nameToID := make(map[string]int, len(names))
	defRows, err := db.Query(`SELECT id, name FROM product_field_definitions WHERE name = ANY($1)`, pq.Array(names))
	if err != nil {
		log.Printf("Error resolving field definition names: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to look up field definitions"})
		return
	}
	defer defRows.Close()
	for defRows.Next() {
		var defID int
		var defName string
		if err := defRows.Scan(&defID, &defName); err != nil {
			log.Printf("Error scanning field definition name: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to look up field definitions"})
			return
		}
		nameToID[defName] = defID
	}
	if err := defRows.Err(); err != nil {
		log.Printf("Error iterating field definition names: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to look up field definitions"})
		return
	}

	// Validate all names exist before touching the DB
	for _, name := range names {
		if _, ok := nameToID[name]; !ok {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Unknown field name: " + name})
			return
		}
	}

	// Apply all updates/deletes inside a single transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction for SetProductFieldValues product %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to set field values"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	for fieldName, value := range body.Values {
		defID := nameToID[fieldName]
		if value == "" {
			if _, err := tx.Exec(`DELETE FROM product_field_values WHERE product_id=$1 AND field_definition_id=$2`, productID, defID); err != nil {
				log.Printf("Error deleting field value for product %d, definition %d: %v", productID, defID, err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete field value"})
				return
			}
		} else {
			if _, err := tx.Exec(`
				INSERT INTO product_field_values (product_id, field_definition_id, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (product_id, field_definition_id) DO UPDATE SET value = EXCLUDED.value
			`, productID, defID, value); err != nil {
				log.Printf("Error upserting field value for product %d, definition %d: %v", productID, defID, err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to set field value"})
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing SetProductFieldValues for product %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to set field values"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Field values updated"})
}
