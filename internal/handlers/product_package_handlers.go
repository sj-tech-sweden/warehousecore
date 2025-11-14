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

	query := `
		SELECT
			pp.package_id,
			pp.package_code,
			pp.name,
			pp.description,
			pp.price,
			pp.created_at,
			pp.updated_at,
			COALESCE(SUM(ppi.quantity), 0) as total_items
		FROM product_packages pp
		LEFT JOIN product_package_items ppi ON pp.package_id = ppi.package_id
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
			&pkg.PackageCode,
			&pkg.Name,
			&pkg.Description,
			&pkg.Price,
			&pkg.CreatedAt,
			&pkg.UpdatedAt,
			&pkg.TotalItems,
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

	// Get package details
	var pkg models.ProductPackageWithItems
	err = db.QueryRow(`
		SELECT
			package_id,
			package_code,
			name,
			description,
			price,
			created_at,
			updated_at
		FROM product_packages
		WHERE package_id = ?
	`, id).Scan(
		&pkg.PackageID,
		&pkg.PackageCode,
		&pkg.Name,
		&pkg.Description,
		&pkg.Price,
		&pkg.CreatedAt,
		&pkg.UpdatedAt,
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
		Name        string                      `json:"name"`
		Description *string                     `json:"description"`
		Price       *float64                    `json:"price"`
		Items       []models.ProductPackageItem `json:"items"`
		Aliases     []string                    `json:"aliases"`
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

	// Create package
	result, err := tx.Exec(`
		INSERT INTO product_packages (package_code, name, description, price)
		VALUES (?, ?, ?, ?)
	`, packageCode, req.Name, req.Description, req.Price)

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
		Name        string                      `json:"name"`
		Description *string                     `json:"description"`
		Price       *float64                    `json:"price"`
		Items       []models.ProductPackageItem `json:"items"`
		Aliases     []string                    `json:"aliases"`
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

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package"})
		return
	}
	defer tx.Rollback()

	// Update package
	result, err := tx.Exec(`
		UPDATE product_packages
		SET name = ?, description = ?, price = ?
		WHERE package_id = ?
	`, req.Name, req.Description, req.Price, id)

	if err != nil {
		log.Printf("Failed to update product package: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update package"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Product package not found"})
		return
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
