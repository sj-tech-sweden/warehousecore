package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"warehousecore/internal/repository"
	"warehousecore/internal/services"
)

// Product represents a product (item type)
type Product struct {
	ProductID           int      `json:"product_id"`
	Name                string   `json:"name"`
	CategoryID          *int     `json:"category_id"`
	SubcategoryID       *string  `json:"subcategory_id"`
	SubbiercategoryID   *string  `json:"subbiercategory_id"`
	ManufacturerID      *int     `json:"manufacturer_id"`
	BrandID             *int     `json:"brand_id"`
	Description         *string  `json:"description"`
	MaintenanceInterval *int     `json:"maintenance_interval"`
	ItemCostPerDay      *float64 `json:"item_cost_per_day"`
	Weight              *float64 `json:"weight"`
	Height              *float64 `json:"height"`
	Width               *float64 `json:"width"`
	Depth               *float64 `json:"depth"`
	PowerConsumption    *float64 `json:"power_consumption"`
	PosInCategory       *int     `json:"pos_in_category"`

	// Joined fields for display
	CategoryName        *string `json:"category_name,omitempty"`
	SubcategoryName     *string `json:"subcategory_name,omitempty"`
	SubbiercategoryName *string `json:"subbiercategory_name,omitempty"`
}

// DeviceCreateRequest represents a request to create devices
type DeviceCreateRequest struct {
	ProductID      int     `json:"product_id"`
	Quantity       int     `json:"quantity"`
	StartingNumber *int    `json:"starting_number"` // Optional, if not provided, auto-generate
	Prefix         *string `json:"prefix"`          // Optional device ID prefix
}

// DeviceCreateResponse represents the response after creating devices
type DeviceCreateResponse struct {
	CreatedCount int      `json:"created_count"`
	DeviceIDs    []string `json:"device_ids"`
}

// GetProducts retrieves all products with optional filtering
func GetProducts(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	categoryID := r.URL.Query().Get("category_id")
	subcategoryID := r.URL.Query().Get("subcategory_id")

	db := repository.GetSQLDB()

	query := `
		SELECT
			p.productID,
			p.name,
			p.categoryID,
			p.subcategoryID,
			p.subbiercategoryID,
			p.manufacturerID,
			p.brandID,
			p.description,
			p.maintenanceInterval,
			p.itemcostperday,
			p.weight,
			p.height,
			p.width,
			p.depth,
			p.powerconsumption,
			p.pos_in_category,
			c.name as category_name,
			sc.name as subcategory_name,
			sbc.name as subbiercategory_name
		FROM products p
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		LEFT JOIN subcategories sc ON p.subcategoryID = sc.subcategoryID
		LEFT JOIN subbiercategories sbc ON p.subbiercategoryID = sbc.subbiercategoryID
		WHERE 1=1
	`

	var args []interface{}

	if search != "" {
		query += " AND (p.name LIKE ? OR p.description LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	if categoryID != "" {
		query += " AND p.categoryID = ?"
		args = append(args, categoryID)
	}

	if subcategoryID != "" {
		query += " AND p.subcategoryID = ?"
		args = append(args, subcategoryID)
	}

	query += " ORDER BY p.name"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query products: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch products"})
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(
			&p.ProductID,
			&p.Name,
			&p.CategoryID,
			&p.SubcategoryID,
			&p.SubbiercategoryID,
			&p.ManufacturerID,
			&p.BrandID,
			&p.Description,
			&p.MaintenanceInterval,
			&p.ItemCostPerDay,
			&p.Weight,
			&p.Height,
			&p.Width,
			&p.Depth,
			&p.PowerConsumption,
			&p.PosInCategory,
			&p.CategoryName,
			&p.SubcategoryName,
			&p.SubbiercategoryName,
		)
		if err != nil {
			log.Printf("Failed to scan product: %v", err)
			continue
		}
		products = append(products, p)
	}

	respondJSON(w, http.StatusOK, products)
}

// GetProduct retrieves a single product by ID
func GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	db := repository.GetSQLDB()
	query := `
		SELECT
			p.productID,
			p.name,
			p.categoryID,
			p.subcategoryID,
			p.subbiercategoryID,
			p.manufacturerID,
			p.brandID,
			p.description,
			p.maintenanceInterval,
			p.itemcostperday,
			p.weight,
			p.height,
			p.width,
			p.depth,
			p.powerconsumption,
			p.pos_in_category,
			c.name as category_name,
			sc.name as subcategory_name,
			sbc.name as subbiercategory_name
		FROM products p
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		LEFT JOIN subcategories sc ON p.subcategoryID = sc.subcategoryID
		LEFT JOIN subbiercategories sbc ON p.subbiercategoryID = sbc.subbiercategoryID
		WHERE p.productID = ?
	`

	var p Product
	err = db.QueryRow(query, id).Scan(
		&p.ProductID,
		&p.Name,
		&p.CategoryID,
		&p.SubcategoryID,
		&p.SubbiercategoryID,
		&p.ManufacturerID,
		&p.BrandID,
		&p.Description,
		&p.MaintenanceInterval,
		&p.ItemCostPerDay,
		&p.Weight,
		&p.Height,
		&p.Width,
		&p.Depth,
		&p.PowerConsumption,
		&p.PosInCategory,
		&p.CategoryName,
		&p.SubcategoryName,
		&p.SubbiercategoryName,
	)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch product"})
		return
	}

	respondJSON(w, http.StatusOK, p)
}

// CreateProduct creates a new product
func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req Product
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Product name is required"})
		return
	}

	db := repository.GetSQLDB()
	result, err := db.Exec(`
		INSERT INTO products (
			name, categoryID, subcategoryID, subbiercategoryID, manufacturerID, brandID,
			description, maintenanceInterval, itemcostperday, weight, height, width, depth,
			powerconsumption, pos_in_category
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		req.Name, req.CategoryID, req.SubcategoryID, req.SubbiercategoryID,
		req.ManufacturerID, req.BrandID, req.Description, req.MaintenanceInterval,
		req.ItemCostPerDay, req.Weight, req.Height, req.Width, req.Depth,
		req.PowerConsumption, req.PosInCategory,
	)

	if err != nil {
		log.Printf("Failed to create product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create product"})
		return
	}

	id, _ := result.LastInsertId()
	req.ProductID = int(id)

	respondJSON(w, http.StatusCreated, req)
}

// UpdateProduct updates an existing product
func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	var req Product
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	db := repository.GetSQLDB()
	result, err := db.Exec(`
		UPDATE products SET
			name = ?, categoryID = ?, subcategoryID = ?, subbiercategoryID = ?,
			manufacturerID = ?, brandID = ?, description = ?, maintenanceInterval = ?,
			itemcostperday = ?, weight = ?, height = ?, width = ?, depth = ?,
			powerconsumption = ?, pos_in_category = ?
		WHERE productID = ?
	`,
		req.Name, req.CategoryID, req.SubcategoryID, req.SubbiercategoryID,
		req.ManufacturerID, req.BrandID, req.Description, req.MaintenanceInterval,
		req.ItemCostPerDay, req.Weight, req.Height, req.Width, req.Depth,
		req.PowerConsumption, req.PosInCategory, id,
	)

	if err != nil {
		log.Printf("Failed to update product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update product"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Product updated successfully"})
}

// DeleteProduct deletes a product
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	db := repository.GetSQLDB()

	// Check if product has devices
	var deviceCount int
	err = db.QueryRow("SELECT COUNT(*) FROM devices WHERE productID = ?", id).Scan(&deviceCount)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to check product devices"})
		return
	}

	if deviceCount > 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "Cannot delete product with devices",
			"message": fmt.Sprintf("Product has %d device(s). Please delete devices first.", deviceCount),
		})
		return
	}

	result, err := db.Exec("DELETE FROM products WHERE productID = ?", id)
	if err != nil {
		log.Printf("Failed to delete product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete product"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Product deleted successfully"})
}

// CreateDevicesForProduct creates multiple devices for a product
func CreateDevicesForProduct(w http.ResponseWriter, r *http.Request) {
	var req DeviceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.ProductID == 0 || req.Quantity <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "product_id and valid quantity are required"})
		return
	}

	if req.Quantity > 100 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Cannot create more than 100 devices at once"})
		return
	}

	db := repository.GetSQLDB()

	// Validate product exists and has required fields for device ID generation trigger
	var productName string
	var abbreviation sql.NullString
	var posInCategory sql.NullInt64
	err := db.QueryRow(`
		SELECT p.name, s.abbreviation, p.pos_in_category
		FROM products p
		LEFT JOIN subcategories s ON p.subcategoryID = s.subcategoryID
		WHERE p.productID = ?
	`, req.ProductID).Scan(&productName, &abbreviation, &posInCategory)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch product"})
		return
	}

	// Check if product has required fields for device ID generation trigger
	if !abbreviation.Valid || abbreviation.String == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "Product missing required subcategory",
			"message": "Product must have a subcategory with an abbreviation to create devices",
		})
		return
	}

	// Log warning if pos_in_category is NULL but allow device creation
	// The database trigger will set pos_in_category for new products automatically
	// Legacy products may have NULL values but device creation should still work
	if !posInCategory.Valid {
		log.Printf("[DEVICE CREATE WARNING] Product %d (%s) has NULL pos_in_category - this may indicate legacy data",
			req.ProductID, productName)
		log.Printf("[DEVICE CREATE] Creating %d devices for product %d (%s) - Subcategory: %s, Position: NULL (legacy)",
			req.Quantity, req.ProductID, productName, abbreviation.String)
	} else {
		log.Printf("[DEVICE CREATE] Creating %d devices for product %d (%s) - Subcategory: %s, Position: %d",
			req.Quantity, req.ProductID, productName, abbreviation.String, posInCategory.Int64)
	}

	// Optional prefix: if triggers respect a session variable we pass it through
	if req.Prefix != nil && *req.Prefix != "" {
		upperPrefix := strings.ToUpper(*req.Prefix)
		if _, err := db.Exec("SET @device_prefix = ?", upperPrefix); err != nil {
			log.Printf("Failed to set device prefix session variable: %v", err)
		} else {
			defer db.Exec("SET @device_prefix = NULL")
		}
	}

	existingIDs := make(map[string]struct{})
	rows, err := db.Query("SELECT deviceID FROM devices WHERE productID = ?", req.ProductID)
	if err == nil {
		defer rows.Close()
		var id string
		for rows.Next() {
			if err := rows.Scan(&id); err == nil {
				existingIDs[id] = struct{}{}
			}
		}
	}

	log.Printf("[DEVICE CREATE] Found %d existing devices for product %d", len(existingIDs), req.ProductID)

	// Create devices one by one and track failures
	failedCount := 0
	for i := 0; i < req.Quantity; i++ {
		if _, err := db.Exec("INSERT INTO devices (productID, status) VALUES (?, 'free')", req.ProductID); err != nil {
			log.Printf("[DEVICE CREATE ERROR] Failed to create device %d/%d for product %d: %v", i+1, req.Quantity, req.ProductID, err)
			failedCount++
		}
	}

	if failedCount > 0 {
		log.Printf("[DEVICE CREATE WARNING] %d/%d device insertions failed", failedCount, req.Quantity)
	}

	createdDeviceIDs := make([]string, 0, req.Quantity)
	rowsNew, err := db.Query("SELECT deviceID FROM devices WHERE productID = ?", req.ProductID)
	if err != nil {
		log.Printf("Failed to fetch generated device IDs: %v", err)
	} else {
		defer rowsNew.Close()
		var id string
		for rowsNew.Next() {
			if err := rowsNew.Scan(&id); err != nil {
				log.Printf("Failed to scan device id: %v", err)
				continue
			}
			if _, existed := existingIDs[id]; !existed {
				createdDeviceIDs = append(createdDeviceIDs, id)
			}
		}
	}

	sort.Strings(createdDeviceIDs)

	log.Printf("[DEVICE CREATE] Successfully created %d devices: %v", len(createdDeviceIDs), createdDeviceIDs)

	labelService := services.NewLabelService()
	for _, deviceID := range createdDeviceIDs {
		if err := labelService.AutoGenerateDeviceLabel(deviceID); err != nil {
			log.Printf("Failed to auto-generate label for device %s: %v", deviceID, err)
		}
	}

	// Return error if no devices were created despite requesting them
	if len(createdDeviceIDs) == 0 {
		var posMsg string
		if posInCategory.Valid {
			posMsg = fmt.Sprintf("pos_in_category: %d", posInCategory.Int64)
		} else {
			posMsg = "pos_in_category: NULL (legacy product)"
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "Failed to create devices",
			"message": fmt.Sprintf("Device creation failed. Check that product has subcategory (%s) and %s.", abbreviation.String, posMsg),
		})
		return
	}

	response := DeviceCreateResponse{
		CreatedCount: len(createdDeviceIDs),
		DeviceIDs:    createdDeviceIDs,
	}

	respondJSON(w, http.StatusCreated, response)
}
