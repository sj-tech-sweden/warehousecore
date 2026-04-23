package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

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

// validateFieldOptions checks that options is a valid JSON string array for 'select' type,
// requires at least one non-empty unique option, trims each element, clears options for
// non-select types, and returns the normalised JSON string on success.
func validateFieldOptions(fieldType string, options *string) (*string, error) {
	if fieldType != "select" {
		// non-select fields must not store options
		return nil, nil
	}
	if options == nil || strings.TrimSpace(*options) == "" {
		return nil, fmt.Errorf("options must contain at least one value for select fields")
	}
	var parsed []string
	if err := json.Unmarshal([]byte(*options), &parsed); err != nil {
		return nil, fmt.Errorf("options must be a valid JSON array of strings")
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("options must contain at least one value for select fields")
	}
	normalized := make([]string, 0, len(parsed))
	seen := make(map[string]struct{}, len(parsed))
	for _, opt := range parsed {
		trimmed := strings.TrimSpace(opt)
		if trimmed == "" {
			return nil, fmt.Errorf("options must not contain empty values")
		}
		if _, exists := seen[trimmed]; exists {
			return nil, fmt.Errorf("options must not contain duplicate values")
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	normalizedJSON, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize options")
	}
	s := string(normalizedJSON)
	return &s, nil
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

	input.Name = strings.TrimSpace(input.Name)
	input.Label = strings.TrimSpace(input.Label)
	if input.Unit != nil {
		trimmed := strings.TrimSpace(*input.Unit)
		if trimmed == "" {
			input.Unit = nil
		} else {
			input.Unit = &trimmed
		}
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
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "A field definition with this name already exists"})
			return
		}
		log.Printf("Error creating product field definition: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create field definition"})
		return
	}

	respondJSON(w, http.StatusCreated, d)
}

// UpdateProductFieldDefinition updates an existing product field definition by ID.
// The field name is immutable after creation and is ignored in the request body.
func UpdateProductFieldDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid field definition ID"})
		return
	}

	var input struct {
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

	input.Label = strings.TrimSpace(input.Label)
	if input.Unit != nil {
		trimmed := strings.TrimSpace(*input.Unit)
		if trimmed == "" {
			input.Unit = nil
		} else {
			input.Unit = &trimmed
		}
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
		SET label=$1, field_type=$2, options=$3, unit=$4, sort_order=$5, is_required=$6
		WHERE id=$7
		RETURNING id, name, label, field_type, options, unit, sort_order, is_required
	`, input.Label, input.FieldType, validatedOptions, input.Unit, input.SortOrder, input.IsRequired, id).
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

// fieldDefMeta holds the metadata needed to validate an incoming field value.
type fieldDefMeta struct {
	ID         int
	FieldType  string
	Options    []string
	IsRequired bool
}

// validateFieldValue checks that value is acceptable for the given field definition.
// An empty value is allowed only for non-required fields (it signals deletion).
func validateFieldValue(name, value string, def fieldDefMeta) error {
	if value == "" {
		if def.IsRequired {
			return fmt.Errorf("field '%s' is required and cannot be empty", name)
		}
		return nil
	}
	switch def.FieldType {
	case "number":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("field '%s' must be a valid finite number", name)
		}
	case "integer":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("field '%s' must be a valid integer", name)
		}
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("field '%s' must be 'true' or 'false'", name)
		}
	case "select":
		found := false
		for _, opt := range def.Options {
			if opt == value {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field '%s': value '%s' is not a valid option", name, value)
		}
	}
	return nil
}

// SetProductFieldValues upserts field values for a specific product atomically.
// An empty "values" map explicitly clears all field values for the product.
// All field names are resolved in a single query; values are validated against their
// field definitions before any writes; updates/deletes run inside a transaction.
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

	// nil Values means the "values" key was omitted entirely — nothing to do.
	if body.Values == nil {
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

	// An empty map is an explicit "clear all field values for this product".
	// Reject the clear if any required field definitions exist, to prevent
	// required values from being removed through this path.
	if len(body.Values) == 0 {
		var hasRequired bool
		if err := db.QueryRowContext(
			r.Context(),
			`SELECT EXISTS(SELECT 1 FROM product_field_definitions WHERE is_required = TRUE)`,
		).Scan(&hasRequired); err != nil {
			log.Printf("Error checking required field definitions for product %d: %v", productID, err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to validate field values"})
			return
		}
		if hasRequired {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Cannot clear field values while required field definitions exist"})
			return
		}
		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			log.Printf("Error starting transaction for SetProductFieldValues product %d: %v", productID, err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to set field values"})
			return
		}
		defer func() { _ = tx.Rollback() }()
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM product_field_values WHERE product_id=$1`, productID); err != nil {
			log.Printf("Error clearing field values for product %d: %v", productID, err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to clear field values"})
			return
		}
		if err := tx.Commit(); err != nil {
			log.Printf("Error committing clear of field values for product %d: %v", productID, err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to clear field values"})
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"message": "Field values updated"})
		return
	}

	// Collect all field names to resolve in one query (with metadata for validation).
	names := make([]string, 0, len(body.Values))
	for k := range body.Values {
		names = append(names, k)
	}

	nameToMeta := make(map[string]fieldDefMeta, len(names))
	defRows, err := db.QueryContext(r.Context(),
		`SELECT id, name, field_type, options, is_required FROM product_field_definitions WHERE name = ANY($1)`,
		pq.Array(names))
	if err != nil {
		log.Printf("Error resolving field definition names: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to look up field definitions"})
		return
	}
	defer defRows.Close()
	for defRows.Next() {
		var meta fieldDefMeta
		var defName string
		var rawOptions *string
		if err := defRows.Scan(&meta.ID, &defName, &meta.FieldType, &rawOptions, &meta.IsRequired); err != nil {
			log.Printf("Error scanning field definition: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to look up field definitions"})
			return
		}
		if rawOptions != nil && *rawOptions != "" {
			if err := json.Unmarshal([]byte(*rawOptions), &meta.Options); err != nil {
				log.Printf("Error parsing options for field '%s': %v", defName, err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Malformed options data for field: " + defName})
				return
			}
		}
		nameToMeta[defName] = meta
	}
	if err := defRows.Err(); err != nil {
		log.Printf("Error iterating field definition names: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to look up field definitions"})
		return
	}

	// Validate all names exist and values are acceptable before touching the DB.
	for _, name := range names {
		meta, ok := nameToMeta[name]
		if !ok {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Unknown field name: " + name})
			return
		}
		if err := validateFieldValue(name, body.Values[name], meta); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	// Enforce required fields that are NOT in the incoming request map.
	// Each such field must already have a stored value; if not, the update would leave
	// a required field empty.
	missingRows, err := db.QueryContext(r.Context(), `
		SELECT d.name
		FROM product_field_definitions d
		WHERE d.is_required = true
		  AND NOT (d.name = ANY($1))
		  AND NOT EXISTS(
		      SELECT 1 FROM product_field_values v
		      WHERE v.product_id = $2 AND v.field_definition_id = d.id
		  )
	`, pq.Array(names), productID)
	if err != nil {
		log.Printf("Error checking required fields for product %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to validate required fields"})
		return
	}
	defer missingRows.Close()
	var missingRequired []string
	for missingRows.Next() {
		var fieldName string
		if err := missingRows.Scan(&fieldName); err != nil {
			log.Printf("Error scanning missing required field: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to validate required fields"})
			return
		}
		missingRequired = append(missingRequired, fieldName)
	}
	if err := missingRows.Err(); err != nil {
		log.Printf("Error iterating missing required fields for product %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to validate required fields"})
		return
	}
	if len(missingRequired) > 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Missing required field(s): %s", strings.Join(missingRequired, ", ")),
		})
		return
	}

	// Apply all updates/deletes inside a single transaction bound to the request context.
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("Error starting transaction for SetProductFieldValues product %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to set field values"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	for fieldName, value := range body.Values {
		defID := nameToMeta[fieldName].ID
		if value == "" {
			if _, err := tx.ExecContext(r.Context(), `DELETE FROM product_field_values WHERE product_id=$1 AND field_definition_id=$2`, productID, defID); err != nil {
				log.Printf("Error deleting field value for product %d, definition %d: %v", productID, defID, err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete field value"})
				return
			}
		} else {
			if _, err := tx.ExecContext(r.Context(), `
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
