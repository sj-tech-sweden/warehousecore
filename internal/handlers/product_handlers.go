package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
	"warehousecore/internal/services"
)

var productPictureService = services.NewProductPictureServiceFromEnv()

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
	IsAccessory         bool     `json:"is_accessory"`
	IsConsumable        bool     `json:"is_consumable"`
	CountTypeID         *int     `json:"count_type_id"`
	StockQuantity       *float64 `json:"stock_quantity"`
	MinStockLevel       *float64 `json:"min_stock_level"`
	GenericBarcode      *string  `json:"generic_barcode"`
	PricePerUnit        *float64 `json:"price_per_unit"`

	// Joined fields for display
	CategoryName        *string `json:"category_name,omitempty"`
	SubcategoryName     *string `json:"subcategory_name,omitempty"`
	SubbiercategoryName *string `json:"subbiercategory_name,omitempty"`
	BrandName           *string `json:"brand_name,omitempty"`
	ManufacturerName    *string `json:"manufacturer_name,omitempty"`
	CountTypeName       *string `json:"count_type_name,omitempty"`
	CountTypeAbbr       *string `json:"count_type_abbr,omitempty"`
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
			p.is_accessory,
			p.is_consumable,
			p.count_type_id,
			p.stock_quantity,
			p.min_stock_level,
			p.generic_barcode,
			p.price_per_unit,
			c.name as category_name,
			sc.name as subcategory_name,
			sbc.name as subbiercategory_name,
			b.name as brand_name,
			m.name as manufacturer_name,
			ct.name as count_type_name,
			ct.abbreviation as count_type_abbr
		FROM products p
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		LEFT JOIN subcategories sc ON p.subcategoryID = sc.subcategoryID
		LEFT JOIN subbiercategories sbc ON p.subbiercategoryID = sbc.subbiercategoryID
		LEFT JOIN brands b ON p.brandID = b.brandID
		LEFT JOIN manufacturer m ON p.manufacturerID = m.manufacturerID
		LEFT JOIN count_types ct ON p.count_type_id = ct.count_type_id
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
			&p.IsAccessory,
			&p.IsConsumable,
			&p.CountTypeID,
			&p.StockQuantity,
			&p.MinStockLevel,
			&p.GenericBarcode,
			&p.PricePerUnit,
			&p.CategoryName,
			&p.SubcategoryName,
			&p.SubbiercategoryName,
			&p.BrandName,
			&p.ManufacturerName,
			&p.CountTypeName,
			&p.CountTypeAbbr,
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
			p.is_accessory,
			p.is_consumable,
			p.count_type_id,
			p.stock_quantity,
			p.min_stock_level,
			p.generic_barcode,
			p.price_per_unit,
			c.name as category_name,
			sc.name as subcategory_name,
			sbc.name as subbiercategory_name,
			b.name as brand_name,
			m.name as manufacturer_name,
			ct.name as count_type_name,
			ct.abbreviation as count_type_abbr
		FROM products p
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		LEFT JOIN subcategories sc ON p.subcategoryID = sc.subcategoryID
		LEFT JOIN subbiercategories sbc ON p.subbiercategoryID = sbc.subbiercategoryID
		LEFT JOIN brands b ON p.brandID = b.brandID
		LEFT JOIN manufacturer m ON p.manufacturerID = m.manufacturerID
		LEFT JOIN count_types ct ON p.count_type_id = ct.count_type_id
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
		&p.IsAccessory,
		&p.IsConsumable,
		&p.CountTypeID,
		&p.StockQuantity,
		&p.MinStockLevel,
		&p.GenericBarcode,
		&p.PricePerUnit,
		&p.CategoryName,
		&p.SubcategoryName,
		&p.SubbiercategoryName,
		&p.BrandName,
		&p.ManufacturerName,
		&p.CountTypeName,
		&p.CountTypeAbbr,
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

// GetProductPictures lists all stored pictures for a product.
func GetProductPictures(w http.ResponseWriter, r *http.Request) {
	if !productPictureService.Enabled() {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Product pictures are not configured"})
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	productName, err := getProductName(id)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	} else if err != nil {
		log.Printf("[PICTURES] Failed to resolve product name: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load product"})
		return
	}

	items, err := productPictureService.ListPictures(productName)
	if err != nil {
		log.Printf("[PICTURES] List failed for product %d: %v", id, err)
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Failed to list pictures"})
		return
	}

	// Sort newest first
	sort.Slice(items, func(i, j int) bool {
		return items[i].ModifiedAt.After(items[j].ModifiedAt)
	})

	type pictureResponse struct {
		FileName    string    `json:"file_name"`
		Size        int64     `json:"size"`
		ContentType string    `json:"content_type"`
		ModifiedAt  time.Time `json:"modified_at"`
		DownloadURL string    `json:"download_url"`
	}

	resp := make([]pictureResponse, 0, len(items))
	for _, pic := range items {
		resp = append(resp, pictureResponse{
			FileName:    pic.FileName,
			Size:        pic.Size,
			ContentType: pic.ContentType,
			ModifiedAt:  pic.ModifiedAt,
			DownloadURL: fmt.Sprintf("/api/v1/admin/products/%d/pictures/%s", id, url.PathEscape(pic.FileName)),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pictures": resp,
	})
}

// UploadProductPictures stores one or more images for a product in Nextcloud.
func UploadProductPictures(w http.ResponseWriter, r *http.Request) {
	if !productPictureService.Enabled() {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Product pictures are not configured"})
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	productName, err := getProductName(id)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	} else if err != nil {
		log.Printf("[PICTURES] Failed to resolve product name: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load product"})
		return
	}

	if err := r.ParseMultipartForm(productPictureService.MaxFileSize() * 4); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid multipart form: " + err.Error()})
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		if singleFile, singleHeader, err := r.FormFile("file"); err == nil {
			singleFile.Close()
			files = append(files, singleHeader)
		}
	}
	if len(files) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "No files provided"})
		return
	}

	uploaded := make([]string, 0, len(files))

	for _, header := range files {
		src, err := header.Open()
		if err != nil {
			log.Printf("[PICTURES] Failed to open upload %s: %v", header.Filename, err)
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to read uploaded file"})
			return
		}

		stored, err := productPictureService.UploadPicture(productName, src, header)
		src.Close()
		if err != nil {
			log.Printf("[PICTURES] Upload failed for product %d: %v", id, err)
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		uploaded = append(uploaded, stored)
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":          "Pictures uploaded successfully",
		"uploaded_files":   uploaded,
		"uploaded_count":   len(uploaded),
		"product_name":     productName,
		"nextcloud_folder": productPictureService.FolderForProduct(productName),
	})
}

// DownloadProductPicture streams a product picture from Nextcloud.
func DownloadProductPicture(w http.ResponseWriter, r *http.Request) {
	if !productPictureService.Enabled() {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Product pictures are not configured"})
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	filename, err := url.PathUnescape(vars["filename"])
	if err != nil || filename == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid filename"})
		return
	}

	productName, err := getProductName(id)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	} else if err != nil {
		log.Printf("[PICTURES] Failed to resolve product name: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load product"})
		return
	}

	reader, contentType, err := productPictureService.DownloadPicture(productName, filename)
	if err != nil {
		log.Printf("[PICTURES] Download failed for product %d (%s): %v", id, filename, err)
		status := http.StatusNotFound
		if strings.Contains(err.Error(), "upload") || strings.Contains(err.Error(), "list") {
			status = http.StatusServiceUnavailable
		}
		respondJSON(w, status, map[string]string{"error": "File not found or storage unavailable"})
		return
	}
	defer reader.Close()

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", url.PathEscape(filename)))

	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("[PICTURES] Failed to stream %s: %v", filename, err)
	}
}

func getProductName(productID int) (string, error) {
	db := repository.GetSQLDB()
	var name string
	err := db.QueryRow("SELECT name FROM products WHERE productID = ?", productID).Scan(&name)
	return name, err
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
			powerconsumption, pos_in_category, is_accessory, is_consumable, count_type_id,
			stock_quantity, min_stock_level, generic_barcode, price_per_unit
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		req.Name, req.CategoryID, req.SubcategoryID, req.SubbiercategoryID,
		req.ManufacturerID, req.BrandID, req.Description, req.MaintenanceInterval,
		req.ItemCostPerDay, req.Weight, req.Height, req.Width, req.Depth,
		req.PowerConsumption, req.PosInCategory, req.IsAccessory, req.IsConsumable,
		req.CountTypeID, req.StockQuantity, req.MinStockLevel, req.GenericBarcode, req.PricePerUnit,
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

	db := repository.GetSQLDB()

	// Check if this product is linked to a package so we can keep metadata in sync
	var packageID sql.NullInt64
	if err := db.QueryRow("SELECT package_id FROM product_packages WHERE product_id = ?", id).Scan(&packageID); err != nil && err != sql.ErrNoRows {
		log.Printf("Failed to check if product is a package: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update product"})
		return
	}

	var req Product
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to start product update transaction: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update product"})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE products SET
			name = ?, categoryID = ?, subcategoryID = ?, subbiercategoryID = ?,
			manufacturerID = ?, brandID = ?, description = ?, maintenanceInterval = ?,
			itemcostperday = ?, weight = ?, height = ?, width = ?, depth = ?,
			powerconsumption = ?, pos_in_category = ?,
			is_accessory = ?, is_consumable = ?, count_type_id = ?,
			stock_quantity = ?, min_stock_level = ?, generic_barcode = ?, price_per_unit = ?
		WHERE productID = ?
	`,
		req.Name, req.CategoryID, req.SubcategoryID, req.SubbiercategoryID,
		req.ManufacturerID, req.BrandID, req.Description, req.MaintenanceInterval,
		req.ItemCostPerDay, req.Weight, req.Height, req.Width, req.Depth,
		req.PowerConsumption, req.PosInCategory,
		req.IsAccessory, req.IsConsumable, req.CountTypeID,
		req.StockQuantity, req.MinStockLevel, req.GenericBarcode, req.PricePerUnit,
		id,
	)

	if err != nil {
		log.Printf("Failed to update product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update product"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// MySQL returns 0 when values are unchanged; verify existence before treating as not found.
		var exists bool
		if err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE productID = ?)", id).Scan(&exists); err != nil {
			log.Printf("Failed to verify product existence after update: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update product"})
			return
		}
		if !exists {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
			return
		}
	}

	// For consumables/accessories: sync stock changes to product_locations
	if (req.IsConsumable || req.IsAccessory) && req.StockQuantity != nil {
		// Get current total from product_locations
		var currentTotal float64
		err := tx.QueryRow(`
			SELECT COALESCE(SUM(quantity), 0) FROM product_locations WHERE product_id = ?
		`, id).Scan(&currentTotal)

		if err != nil {
			log.Printf("Warning: Failed to get current stock total: %v", err)
		} else {
			newTotal := *req.StockQuantity
			difference := newTotal - currentTotal

			if difference != 0 {
				// Get the zone with most stock, or create default location
				var defaultZoneID sql.NullInt64
				err := tx.QueryRow(`
					SELECT zone_id FROM product_locations WHERE product_id = ? ORDER BY quantity DESC LIMIT 1
				`, id).Scan(&defaultZoneID)

				if err == sql.ErrNoRows {
					// No locations exist - create in zone with full quantity
					_, err = tx.Exec(`
						INSERT INTO product_locations (product_id, zone_id, quantity) VALUES (?, NULL, ?)
					`, id, newTotal)
					if err != nil {
						log.Printf("Error: Failed to create product location: %v", err)
					}
				} else if err == nil {
					// Update the primary zone
					_, err = tx.Exec(`
						UPDATE product_locations SET quantity = quantity + ? WHERE product_id = ? AND zone_id <=> ?
					`, difference, id, defaultZoneID)
					if err != nil {
						log.Printf("Error: Failed to update product_locations: %v", err)
					} else {
						log.Printf("Updated product_locations for product %d: %+.2f kg (zone_id=%v)", id, difference, defaultZoneID)
					}
				}
			}
		}
	}

	if packageID.Valid {
		if _, err := tx.Exec(`
			UPDATE product_packages
			SET name = ?, description = ?, price = ?
			WHERE package_id = ?
		`, req.Name, req.Description, req.ItemCostPerDay, packageID.Int64); err != nil {
			log.Printf("Failed to update linked product package for product %d: %v", id, err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update linked product package"})
			return
		}
	}

	// BEFORE commit: Recalculate stock_quantity from product_locations (within transaction)
	if req.IsConsumable || req.IsAccessory {
		_, err := tx.Exec(`
			UPDATE products
			SET stock_quantity = (
				SELECT COALESCE(SUM(quantity), 0)
				FROM product_locations
				WHERE product_id = ?
			)
			WHERE productID = ?
		`, id, id)
		if err != nil {
			log.Printf("Error: Failed to recalculate stock_quantity: %v", err)
		} else {
			log.Printf("Recalculated stock_quantity for product %d from product_locations", id)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit product update transaction: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update product"})
		return
	}

	message := "Product updated successfully"
	if packageID.Valid {
		message = "Package product updated successfully"
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": message})
}

// DeleteProduct deletes a product and cascades to delete all associated devices
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	db := repository.GetSQLDB()

	// Check if this product is a package product (managed by product_packages)
	var packageID int
	err = db.QueryRow("SELECT package_id FROM product_packages WHERE product_id = ?", id).Scan(&packageID)
	if err == nil {
		// Product is managed by a package - cannot be deleted directly
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "Package product cannot be deleted directly",
			"message": "This product is managed by a package. Please delete it via the Packages tab.",
		})
		return
	} else if err != sql.ErrNoRows {
		log.Printf("Failed to check if product is a package: %v", err)
	}

	// Get product name for logging
	var productName string
	err = db.QueryRow("SELECT name FROM products WHERE productID = ?", id).Scan(&productName)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch product"})
		return
	}

	// Count devices to be deleted
	var deviceCount int
	err = db.QueryRow("SELECT COUNT(*) FROM devices WHERE productID = ?", id).Scan(&deviceCount)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to check product devices"})
		return
	}

	log.Printf("[PRODUCT DELETE] Deleting product %d (%s) with %d associated device(s)", id, productName, deviceCount)

	// Cascade delete: Delete all associated devices first
	if deviceCount > 0 {
		// Get device IDs for detailed logging
		var deviceIDs []string
		rows, err := db.Query("SELECT deviceID FROM devices WHERE productID = ?", id)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var deviceID string
				if err := rows.Scan(&deviceID); err == nil {
					deviceIDs = append(deviceIDs, deviceID)
				}
			}
		}

		log.Printf("[PRODUCT DELETE] Deleting %d devices: %v", len(deviceIDs), deviceIDs)

		// Delete all devices for this product
		result, err := db.Exec("DELETE FROM devices WHERE productID = ?", id)
		if err != nil {
			log.Printf("[PRODUCT DELETE ERROR] Failed to delete devices for product %d: %v", id, err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "Failed to delete associated devices",
				"message": fmt.Sprintf("Error deleting %d device(s) before product deletion", deviceCount),
			})
			return
		}

		deletedDevices, _ := result.RowsAffected()
		log.Printf("[PRODUCT DELETE] Successfully deleted %d devices for product %d", deletedDevices, id)
	}

	// Now delete the product
	result, err := db.Exec("DELETE FROM products WHERE productID = ?", id)
	if err != nil {
		log.Printf("[PRODUCT DELETE ERROR] Failed to delete product %d: %v", id, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete product"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	log.Printf("[PRODUCT DELETE] Successfully deleted product %d (%s)", id, productName)

	// Include device count in response
	message := "Product deleted successfully"
	if deviceCount > 0 {
		message = fmt.Sprintf("Product deleted successfully along with %d device(s)", deviceCount)
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message":         message,
		"deleted_devices": fmt.Sprintf("%d", deviceCount),
	})
}

// CreateDevicesForProduct creates multiple devices for a product
func CreateDevicesForProduct(w http.ResponseWriter, r *http.Request) {
	// Extract product ID from URL path
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	var req DeviceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Override ProductID with the one from URL path
	req.ProductID = productID

	log.Printf("[DEVICE CREATE] Starting device creation for product %d, quantity: %d, prefix: %v",
		req.ProductID, req.Quantity, req.Prefix)

	if req.Quantity <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Valid quantity is required"})
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
	err = db.QueryRow(`
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

	// Auto-generate labels for all created devices in the background
	// This prevents blocking the API response if label generation is slow or fails
	go func() {
		labelService := services.NewLabelService()
		for _, deviceID := range createdDeviceIDs {
			if err := labelService.AutoGenerateDeviceLabel(deviceID); err != nil {
				log.Printf("[LABEL CREATE ERROR] Failed to generate label for device %s: %v", deviceID, err)
			} else {
				log.Printf("[LABEL CREATE SUCCESS] Generated label for device %s using default template", deviceID)
			}
		}
	}()

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

// GetProductDevices retrieves all devices for a specific product
func GetProductDevices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	db := repository.GetSQLDB()

	query := `
		WITH latest_job AS (
			SELECT jd.deviceID, MAX(jd.jobID) AS jobID
			FROM jobdevices jd
			GROUP BY jd.deviceID
		)
		SELECT d.deviceID, d.productID, d.serialnumber, d.barcode, d.qr_code, d.status,
		       d.current_location, d.zone_id,
		       d.condition_rating, d.usage_hours, d.purchaseDate, d.lastmaintenance, d.nextmaintenance,
		       d.notes, d.label_path,
		       COALESCE(p.name, '') AS product_name,
		       COALESCE(cat.name, '') AS product_category,
		       COALESCE(z.name, '') AS zone_name,
		       COALESCE(z.code, '') AS zone_code,
		       dc.caseID,
		       COALESCE(c.name, '') AS case_name,
		       lj.jobID,
		       COALESCE(j.job_code, '') AS job_number
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN categories cat ON p.categoryID = cat.categoryID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		LEFT JOIN devicescases dc ON d.deviceID = dc.deviceID
		LEFT JOIN cases c ON dc.caseID = c.caseID
		LEFT JOIN latest_job lj ON lj.deviceID = d.deviceID
		LEFT JOIN jobs j ON lj.jobID = j.jobID
		WHERE d.productID = ?
		ORDER BY d.deviceID ASC
	`

	rows, err := db.Query(query, productID)
	if err != nil {
		log.Printf("[PRODUCT DEVICES] Failed to query devices for product %d: %v", productID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch product devices"})
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
			&device.Status,
			&device.CurrentLocation,
			&device.ZoneID,
			&device.ConditionRating,
			&device.UsageHours,
			&device.PurchaseDate,
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
			&device.CurrentJobID,
			&device.JobNumber,
		)
		if err != nil {
			log.Printf("[PRODUCT DEVICES] Failed to scan device: %v", err)
			continue
		}

		responses = append(responses, toDeviceAdminResponse(&device))
	}

	respondJSON(w, http.StatusOK, responses)
}

// GetLowStockAlerts returns products with stock below minimum level
func GetLowStockAlerts(w http.ResponseWriter, r *http.Request) {
	db := repository.GetDB()

	type LowStockAlert struct {
		ProductID      int     `json:"product_id"`
		Name           string  `json:"name"`
		StockQuantity  float64 `json:"stock_quantity"`
		MinStockLevel  float64 `json:"min_stock_level"`
		CountTypeName  string  `json:"count_type_name"`
		CountTypeAbbr  string  `json:"count_type_abbr"`
		GenericBarcode string  `json:"generic_barcode"`
		IsAccessory    bool    `json:"is_accessory"`
		IsConsumable   bool    `json:"is_consumable"`
	}

	var alerts []LowStockAlert
	err := db.Raw(`
		SELECT
			p.productID,
			p.name,
			COALESCE(p.stock_quantity, 0) as stock_quantity,
			COALESCE(p.min_stock_level, 0) as min_stock_level,
			COALESCE(ct.name, 'Units') as count_type_name,
			COALESCE(ct.abbreviation, 'Stk') as count_type_abbr,
			COALESCE(p.generic_barcode, '') as generic_barcode,
			COALESCE(p.is_accessory, false) as is_accessory,
			COALESCE(p.is_consumable, false) as is_consumable
		FROM products p
		LEFT JOIN count_types ct ON p.count_type_id = ct.count_type_id
		WHERE (p.is_consumable = 1 OR p.is_accessory = 1)
		  AND p.min_stock_level IS NOT NULL
		  AND p.min_stock_level > 0
		  AND COALESCE(p.stock_quantity, 0) < p.min_stock_level
		ORDER BY (COALESCE(p.stock_quantity, 0) / p.min_stock_level) ASC
	`).Scan(&alerts).Error
	if err != nil {
		log.Printf("Failed to fetch low stock alerts: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch low stock alerts",
		})
		return
	}

	if alerts == nil {
		alerts = []LowStockAlert{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
	})
}
