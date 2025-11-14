package models

import (
	"database/sql"
	"time"
)

// ProductPackage represents a package of products with a fixed price
type ProductPackage struct {
	PackageID   int             `json:"package_id" db:"package_id"`
	PackageCode string          `json:"package_code" db:"package_code"`
	Name        string          `json:"name" db:"name"`
	Description sql.NullString  `json:"description" db:"description"`
	Price       sql.NullFloat64 `json:"price" db:"price"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// ProductPackageItem represents the relationship between a package and a product
type ProductPackageItem struct {
	PackageItemID int       `json:"package_item_id" db:"package_item_id"`
	PackageID     int       `json:"package_id" db:"package_id"`
	ProductID     int       `json:"product_id" db:"product_id"`
	Quantity      int       `json:"quantity" db:"quantity"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ProductPackageWithItems includes the items in the package
type ProductPackageWithItems struct {
	ProductPackage
	Items      []PackageItemDetail `json:"items,omitempty"`
	TotalItems int                 `json:"total_items"`
	Aliases    []string            `json:"aliases,omitempty"`
}

// PackageItemDetail provides detailed information about a product in the package
type PackageItemDetail struct {
	PackageItemID int     `json:"package_item_id"`
	ProductID     int     `json:"product_id"`
	ProductName   string  `json:"product_name"`
	Quantity      int     `json:"quantity"`
	CategoryName  *string `json:"category_name,omitempty"`
	BrandName     *string `json:"brand_name,omitempty"`
}
