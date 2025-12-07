package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

// GetProductPackages retrieves all product packages with optional search
func GetProductPackages(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")

	db := repository.GetSQLDB()
	if err := ensurePackageCodeSupport(db); err != nil {
		log.Printf("Failed to ensure package code support: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to prepare package storage"})
		return
	}

	query := `
		SELECT
			pp.package_id,
			pp.product_id,
			pp.package_code,
			pp.name,
			pp.description,
			pp.price,
			pp.website_visible,
			pp.created_at,
			pp.updated_at,
			COALESCE(SUM(ppi.quantity), 0) as total_items,
			p.categoryID,
			c.name as category_name,
			p.subcategoryID
		FROM product_packages pp
		LEFT JOIN product_package_items ppi ON pp.package_id = ppi.package_id
		LEFT JOIN products p ON pp.product_id = p.productID
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		WHERE 1=1
	`

	var args []interface{}

	if search != "" {
		query += " AND (pp.name LIKE ? OR pp.description LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	query += " GROUP BY pp.package_id ORDER BY pp.name"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to query product packages: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch product packages"})
		return
	}
	defer rows.Close()

	var packages []models.ProductPackageWithItems
	for rows.Next() {
		var pkg models.ProductPackageWithItems
		err := rows.Scan(
			&pkg.PackageID,
			&pkg.ProductID,
			&pkg.PackageCode,
			&pkg.Name,
			&pkg.Description,
			&pkg.Price,
			&pkg.WebsiteVisible,
			&pkg.CreatedAt,
			&pkg.UpdatedAt,
			&pkg.TotalItems,
			&pkg.CategoryID,
			&pkg.CategoryName,
			&pkg.SubcategoryID,
		)
		if err != nil {
			log.Printf("Failed to scan product package: %v", err)
			continue
		}
		packages = append(packages, pkg)
	}

	respondJSON(w, http.StatusOK, packages)
}

// GetProductPackage retrieves a single product package by ID with all items
func GetProductPackage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid package ID"})
		return
	}

	db := repository.GetSQLDB()
	if err := ensurePackageCodeSupport(db); err != nil {
		log.Printf("Failed to ensure package code support: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to prepare package storage"})
		return
	}

	// Get package details with product category information
	var pkg models.ProductPackageWithItems
	err = db.QueryRow(`
		SELECT
			pp.package_id,
			pp.product_id,
			pp.package_code,
			pp.name,
			pp.description,
			pp.price,
			pp.website_visible,
			pp.created_at,
			pp.updated_at,
			p.categoryID,
			c.name as category_name,
			p.subcategoryID
		FROM product_packages pp
		LEFT JOIN products p ON pp.product_id = p.productID
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		WHERE pp.package_id = ?
	`, id).Scan(
		&pkg.PackageID,
		&pkg.ProductID,
		&pkg.PackageCode,
		&pkg.Name,
		&pkg.Description,
		&pkg.Price,
		&pkg.WebsiteVisible,
		&pkg.CreatedAt,
		&pkg.UpdatedAt,
		&pkg.CategoryID,
		&pkg.CategoryName,
		&pkg.SubcategoryID,
	)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product package not found"})
		return
	} else if err != nil {
		log.Printf("Failed to query product package: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch product package"})
		return
	}

	// Get package items
	itemRows, err := db.Query(`
		SELECT
			ppi.package_item_id,
			ppi.product_id,
			p.name as product_name,
			ppi.quantity,
			c.name as category_name,
			b.name as brand_name
		FROM product_package_items ppi
		JOIN products p ON ppi.product_id = p.productID
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		LEFT JOIN brands b ON p.brandID = b.brandID
		WHERE ppi.package_id = ?
		ORDER BY p.name
	`, id)

	if err != nil {
		log.Printf("Failed to query package items: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch package items"})
		return
	}
	defer itemRows.Close()

	var items []models.PackageItemDetail
	for itemRows.Next() {
		var item models.PackageItemDetail
		err := itemRows.Scan(
			&item.PackageItemID,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.CategoryName,
			&item.BrandName,
		)
		if err != nil {
			log.Printf("Failed to scan package item: %v", err)
			continue
		}
		items = append(items, item)
		pkg.TotalItems += item.Quantity
	}

	pkg.Items = items
	aliases, err := fetchPackageAliases(db, id)
	if err != nil {
		log.Printf("Failed to fetch package aliases: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch package aliases"})
		return
	}
	pkg.Aliases = aliases

	respondJSON(w, http.StatusOK, pkg)
}

// CreateProductPackage creates a new product package
func CreateProductPackage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string                      `json:"name"`
		Description       *string                     `json:"description"`
		Price             *float64                    `json:"price"`
		Items             []models.ProductPackageItem `json:"items"`
		Aliases           []string                    `json:"aliases"`
		CategoryID        *int                        `json:"category_id"`        // NEW: Product category
		SubcategoryID     *string                     `json:"subcategory_id"`     // NEW: Product subcategory
		SubbiercategoryID *string                     `json:"subbiercategory_id"` // NEW: Product sub-subcategory
		WebsiteVisible    bool                        `json:"website_visible"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Package name is required"})
		return
	}

	db := repository.GetSQLDB()

	if err := ensurePackageCodeSupport(db); err != nil {
		log.Printf("Failed to ensure package code support: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to prepare package storage"})
		return
	}

	packageCode, err := generatePackageCode(db)
	if err != nil {
		log.Printf("Failed to generate package code: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to assign package code"})
		return
	}

	normalizedAliases := normalizeAliases(req.Aliases)

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create package"})
		return
	}
	defer tx.Rollback()

	// Step 1: Create a Product entry for this package
	productResult, err := tx.Exec(`
		INSERT INTO products (name, categoryID, subcategoryID, subbiercategoryID, itemcostperday, description, website_visible)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, req.Name, req.CategoryID, req.SubcategoryID, req.SubbiercategoryID, req.Price, req.Description, req.WebsiteVisible)

	if err != nil {
		log.Printf("Failed to create package product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create package product"})
		return
	}

	productID, _ := productResult.LastInsertId()
	log.Printf("[PACKAGE CREATE] Created product ID %d for package '%s'", productID, req.Name)

	// Step 2: Create package linked to the product
	result, err := tx.Exec(`
		INSERT INTO product_packages (package_code, product_id, name, description, price, website_visible)
		VALUES (?, ?, ?, ?, ?, ?)
	`, packageCode, productID, req.Name, req.Description, req.Price, req.WebsiteVisible)

	if err != nil {
		log.Printf("Failed to create product package: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create package"})
		return
	}

	packageID, _ := result.LastInsertId()

	// Add items to package
	if len(req.Items) > 0 {
		for _, item := range req.Items {
			if item.Quantity <= 0 {
				continue
			}

			_, err := tx.Exec(`
				INSERT INTO product_package_items (package_id, product_id, quantity)
				VALUES (?, ?, ?)
			`, packageID, item.ProductID, item.Quantity)

			if err != nil {
				log.Printf("Failed to add item to package: %v", err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add items to package"})
				return
			}
		}
	}

	if err := replacePackageAliases(tx, packageID, normalizedAliases); err != nil {
		log.Printf("Failed to save package aliases: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save package aliases"})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create package"})
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"package_id":   packageID,
		"package_code": packageCode,
		"message":      "Product package created successfully",
	})
}

// UpdateProductPackage updates an existing product package
func UpdateProductPackage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid package ID"})
		return
	}

	var req struct {
		Name              string                      `json:"name"`
		Description       *string                     `json:"description"`
		Price             *float64                    `json:"price"`
		Items             []models.ProductPackageItem `json:"items"`
		Aliases           []string                    `json:"aliases"`
		CategoryID        *int                        `json:"category_id"`        // NEW: Product category
		SubcategoryID     *string                     `json:"subcategory_id"`     // NEW: Product subcategory
		SubbiercategoryID *string                     `json:"subbiercategory_id"` // NEW: Product sub-subcategory
		WebsiteVisible    bool                        `json:"website_visible"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Package name is required"})
		return
	}

	db := repository.GetSQLDB()
	normalizedAliases := normalizeAliases(req.Aliases)

	if err := ensurePackageCodeSupport(db); err != nil {
		log.Printf("Failed to ensure package code support: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to prepare package storage"})
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package"})
		return
	}
	defer tx.Rollback()

	// Get the product_id for this package
	var productID int
	err = tx.QueryRow("SELECT product_id FROM product_packages WHERE package_id = ?", id).Scan(&productID)
	if err != nil {
		log.Printf("Failed to get product_id for package: %v", err)
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Package not found"})
		return
	}

	// Update the linked product
	_, err = tx.Exec(`
		UPDATE products
		SET name = ?, categoryID = ?, subcategoryID = ?, subbiercategoryID = ?, itemcostperday = ?, description = ?, website_visible = ?
		WHERE productID = ?
	`, req.Name, req.CategoryID, req.SubcategoryID, req.SubbiercategoryID, req.Price, req.Description, req.WebsiteVisible, productID)

	if err != nil {
		log.Printf("Failed to update package product: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package product"})
		return
	}

	// Update package
	result, err := tx.Exec(`
		UPDATE product_packages
		SET name = ?, description = ?, price = ?, website_visible = ?
		WHERE package_id = ?
	`, req.Name, req.Description, req.Price, req.WebsiteVisible, id)

	if err != nil {
		log.Printf("Failed to update product package: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		var exists bool
		if err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM product_packages WHERE package_id = ?)", id).Scan(&exists); err != nil {
			log.Printf("Failed to confirm product package existence: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package"})
			return
		}
		if !exists {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product package not found"})
			return
		}
	}

	// Delete existing items
	_, err = tx.Exec("DELETE FROM product_package_items WHERE package_id = ?", id)
	if err != nil {
		log.Printf("Failed to delete package items: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package items"})
		return
	}

	// Add new items
	if len(req.Items) > 0 {
		for _, item := range req.Items {
			if item.Quantity <= 0 {
				continue
			}

			_, err := tx.Exec(`
				INSERT INTO product_package_items (package_id, product_id, quantity)
				VALUES (?, ?, ?)
			`, id, item.ProductID, item.Quantity)

			if err != nil {
				log.Printf("Failed to add item to package: %v", err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package items"})
				return
			}
		}
	}

	if err := replacePackageAliases(tx, int64(id), normalizedAliases); err != nil {
		log.Printf("Failed to update package aliases: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package aliases"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Product package updated successfully"})
}

// DeleteProductPackage deletes a product package
func DeleteProductPackage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid package ID"})
		return
	}

	db := repository.GetSQLDB()
	if err := ensurePackageCodeSupport(db); err != nil {
		log.Printf("Failed to ensure package code support: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to prepare package storage"})
		return
	}

	result, err := db.Exec("DELETE FROM product_packages WHERE package_id = ?", id)
	if err != nil {
		log.Printf("Failed to delete product package: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete package"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product package not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Product package deleted successfully"})
}

type PackageAliasEntry struct {
	Alias       string   `json:"alias"`
	PackageID   int      `json:"package_id"`
	PackageCode string   `json:"package_code"`
	PackageName string   `json:"package_name"`
	Price       *float64 `json:"price,omitempty"`
}

// GetProductPackageAliasMap returns all alias mappings for OCR integrations
func GetProductPackageAliasMap(w http.ResponseWriter, r *http.Request) {
	db := repository.GetSQLDB()
	if err := ensurePackageCodeSupport(db); err != nil {
		log.Printf("Failed to ensure package code support: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to prepare package storage"})
		return
	}

	rows, err := db.Query(`
		SELECT
			ppa.alias,
			pp.package_id,
			pp.package_code,
			pp.name,
			pp.price
		FROM product_package_aliases ppa
		JOIN product_packages pp ON ppa.package_id = pp.package_id
		ORDER BY ppa.alias
	`)
	if err != nil {
		log.Printf("Failed to query package aliases: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch package aliases"})
		return
	}
	defer rows.Close()

	var entries []PackageAliasEntry
	for rows.Next() {
		var entry PackageAliasEntry
		var price sql.NullFloat64
		if err := rows.Scan(&entry.Alias, &entry.PackageID, &entry.PackageCode, &entry.PackageName, &price); err != nil {
			log.Printf("Failed to scan alias row: %v", err)
			continue
		}
		if price.Valid {
			val := price.Float64
			entry.Price = &val
		}
		entries = append(entries, entry)
	}

	if entries == nil {
		entries = []PackageAliasEntry{}
	}

	respondJSON(w, http.StatusOK, entries)
}

func ensurePackageCodeSupport(db *sql.DB) error {
	if db == nil {
		return errors.New("database connection is nil")
	}

	if err := ensureProductPackageTables(db); err != nil {
		return err
	}
	if err := ensurePackageCodeColumn(db); err != nil {
		return err
	}
	return ensureAliasTable(db)
}

func ensureProductPackageTables(db *sql.DB) error {
	const tableCheck = `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
		  AND table_name = ?
	`

	// Ensure product_packages exists with final schema
	var count int
	if err := db.QueryRow(tableCheck, "product_packages").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS product_packages (
				package_id INT AUTO_INCREMENT PRIMARY KEY,
				package_code VARCHAR(32) NOT NULL,
				name VARCHAR(255) NOT NULL,
				description TEXT,
				price DECIMAL(10,2),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				UNIQUE KEY uq_product_package_code (package_code),
				INDEX idx_name (name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
		`); err != nil {
			return err
		}
	}

	// Ensure product_package_items exists
	count = 0
	if err := db.QueryRow(tableCheck, "product_package_items").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS product_package_items (
				package_item_id INT AUTO_INCREMENT PRIMARY KEY,
				package_id INT NOT NULL,
				product_id INT NOT NULL,
				quantity INT NOT NULL DEFAULT 1,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (package_id) REFERENCES product_packages(package_id) ON DELETE CASCADE,
				FOREIGN KEY (product_id) REFERENCES products(productID) ON DELETE CASCADE,
				UNIQUE KEY unique_package_product (package_id, product_id),
				INDEX idx_package_id (package_id),
				INDEX idx_product_id (product_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
		`); err != nil {
			return err
		}
	}

	return nil
}

func ensurePackageCodeColumn(db *sql.DB) error {
	const columnCheck = `
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'product_packages'
		  AND column_name = 'package_code'
	`
	var count int
	if err := db.QueryRow(columnCheck).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if _, err := db.Exec(`ALTER TABLE product_packages ADD COLUMN package_code VARCHAR(32) NULL AFTER package_id`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE product_packages SET package_code = CONCAT('PKG-', LPAD(package_id, 6, '0')) WHERE package_code IS NULL OR package_code = ''`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE product_packages MODIFY COLUMN package_code VARCHAR(32) NOT NULL`); err != nil {
		return err
	}
	const indexCheck = `
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'product_packages'
		  AND index_name = 'uq_product_package_code'
	`
	var idxCount int
	if err := db.QueryRow(indexCheck).Scan(&idxCount); err != nil {
		return err
	}
	if idxCount == 0 {
		if _, err := db.Exec(`ALTER TABLE product_packages ADD UNIQUE KEY uq_product_package_code (package_code)`); err != nil {
			return err
		}
	}
	return nil
}

func ensureAliasTable(db *sql.DB) error {
	const tableCheck = `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
		  AND table_name = 'product_package_aliases'
	`
	var count int
	if err := db.QueryRow(tableCheck).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS product_package_aliases (
			alias_id INT AUTO_INCREMENT PRIMARY KEY,
			package_id INT NOT NULL,
			alias VARCHAR(191) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uq_package_alias (package_id, alias),
			INDEX idx_alias (alias),
			CONSTRAINT fk_package_alias_package
				FOREIGN KEY (package_id) REFERENCES product_packages(package_id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	return err
}

func normalizeAliases(values []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, alias := range values {
		trimmed := strings.TrimSpace(alias)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func replacePackageAliases(tx *sql.Tx, packageID int64, aliases []string) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}

	if _, err := tx.Exec("DELETE FROM product_package_aliases WHERE package_id = ?", packageID); err != nil {
		return err
	}

	for _, alias := range aliases {
		if _, err := tx.Exec("INSERT INTO product_package_aliases (package_id, alias) VALUES (?, ?)", packageID, alias); err != nil {
			return err
		}
	}
	return nil
}

func fetchPackageAliases(db *sql.DB, packageID int) ([]string, error) {
	rows, err := db.Query("SELECT alias FROM product_package_aliases WHERE package_id = ? ORDER BY alias", packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []string
	for rows.Next() {
		var alias string
		if err := rows.Scan(&alias); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	if aliases == nil {
		aliases = []string{}
	}
	return aliases, nil
}

const packageCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomPackageCodeSegment(length int) (string, error) {
	var result strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(packageCodeCharset))))
		if err != nil {
			return "", err
		}
		result.WriteByte(packageCodeCharset[n.Int64()])
	}
	return result.String(), nil
}

func generatePackageCode(db *sql.DB) (string, error) {
	if db == nil {
		return "", errors.New("database connection is nil")
	}

	for attempts := 0; attempts < 20; attempts++ {
		segment, err := randomPackageCodeSegment(5)
		if err != nil {
			return "", err
		}
		code := fmt.Sprintf("PKG-%s", segment)

		var exists bool
		if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM product_packages WHERE package_code = ?)", code).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}

	return "", errors.New("could not generate unique package code")
}

// AddItemToPackage adds a product to an existing package
func AddItemToPackage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packageID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid package ID"})
		return
	}

	var req struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.ProductID <= 0 || req.Quantity <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Valid product ID and quantity are required"})
		return
	}

	db := repository.GetSQLDB()
	if err := ensurePackageCodeSupport(db); err != nil {
		log.Printf("Failed to ensure package code support: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to prepare package storage"})
		return
	}

	// Check if package exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM product_packages WHERE package_id = ?)", packageID).Scan(&exists)
	if err != nil || !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product package not found"})
		return
	}

	// Check if product exists
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE productID = ?)", req.ProductID).Scan(&exists)
	if err != nil || !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	// Add or update item
	_, err = db.Exec(`
		INSERT INTO product_package_items (package_id, product_id, quantity)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE quantity = ?
	`, packageID, req.ProductID, req.Quantity, req.Quantity)

	if err != nil {
		log.Printf("Failed to add item to package: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add item to package"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Item added to package successfully"})
}

// RemoveItemFromPackage removes a product from a package
func RemoveItemFromPackage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	packageID, err := strconv.Atoi(vars["package_id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid package ID"})
		return
	}

	itemID, err := strconv.Atoi(vars["item_id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid item ID"})
		return
	}

	db := repository.GetSQLDB()
	if err := ensurePackageCodeSupport(db); err != nil {
		log.Printf("Failed to ensure package code support: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to prepare package storage"})
		return
	}

	result, err := db.Exec(`
		DELETE FROM product_package_items
		WHERE package_id = ? AND package_item_id = ?
	`, packageID, itemID)

	if err != nil {
		log.Printf("Failed to remove item from package: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to remove item from package"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Package item not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Item removed from package successfully"})
}
