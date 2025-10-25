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
	"warehousecore/internal/services"
)

// Product represents a product (item type)
type Product struct {
	ProductID             int      `json:"product_id"`
	Name                  string   `json:"name"`
	CategoryID            *int     `json:"category_id"`
	SubcategoryID         *string  `json:"subcategory_id"`
	SubbiercategoryID     *string  `json:"subbiercategory_id"`
	ManufacturerID        *int     `json:"manufacturer_id"`
	BrandID               *int     `json:"brand_id"`
	Description           *string  `json:"description"`
	MaintenanceInterval   *int     `json:"maintenance_interval"`
	ItemCostPerDay        *float64 `json:"item_cost_per_day"`
	Weight                *float64 `json:"weight"`
	Height                *float64 `json:"height"`
	Width                 *float64 `json:"width"`
	Depth                 *float64 `json:"depth"`
	PowerConsumption      *float64 `json:"power_consumption"`
	PosInCategory         *int     `json:"pos_in_category"`

	// Joined fields for display
	CategoryName          *string  `json:"category_name,omitempty"`
	SubcategoryName       *string  `json:"subcategory_name,omitempty"`
	SubbiercategoryName   *string  `json:"subbiercategory_name,omitempty"`
}

// DeviceCreateRequest represents a request to create devices
type DeviceCreateRequest struct {
	ProductID       int      `json:"product_id"`
	Quantity        int      `json:"quantity"`
	StartingNumber  *int     `json:"starting_number"` // Optional, if not provided, auto-generate
	Prefix          *string  `json:"prefix"`          // Optional device ID prefix
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

	// Get product info for generating device IDs
	var productName string
	err := db.QueryRow("SELECT name FROM products WHERE productID = ?", req.ProductID).Scan(&productName)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch product"})
		return
	}

	// Generate device IDs
	prefix := "DEV"
	if req.Prefix != nil && *req.Prefix != "" {
		prefix = strings.ToUpper(*req.Prefix)
	}

	startNum := 1
	if req.StartingNumber != nil {
		startNum = *req.StartingNumber
	} else {
		// Find highest existing number for this prefix
		var maxNum sql.NullInt64
		db.QueryRow(`
			SELECT MAX(CAST(SUBSTRING(deviceID, ?) AS UNSIGNED))
			FROM devices
			WHERE deviceID LIKE ?
		`, len(prefix)+1, prefix+"%").Scan(&maxNum)

		if maxNum.Valid {
			startNum = int(maxNum.Int64) + 1
		}
	}

	// Create devices
	createdDeviceIDs := make([]string, 0, req.Quantity)
	labelService := services.NewLabelService()

	for i := 0; i < req.Quantity; i++ {
		deviceID := fmt.Sprintf("%s%04d", prefix, startNum+i)

		_, err := db.Exec(`
			INSERT INTO devices (deviceID, productID, status)
			VALUES (?, ?, 'free')
		`, deviceID, req.ProductID)

		if err != nil {
			log.Printf("Failed to create device %s: %v", deviceID, err)
			// Continue creating other devices
			continue
		}

		createdDeviceIDs = append(createdDeviceIDs, deviceID)

		// Automatically generate a basic label for the new device
		if err := labelService.AutoGenerateDeviceLabel(deviceID); err != nil {
			log.Printf("Failed to auto-generate label for device %s: %v", deviceID, err)
			// Don't fail device creation if label generation fails
		}
	}

	response := DeviceCreateResponse{
		CreatedCount: len(createdDeviceIDs),
		DeviceIDs:    createdDeviceIDs,
	}

	respondJSON(w, http.StatusCreated, response)
}
