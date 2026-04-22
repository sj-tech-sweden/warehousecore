package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"warehousecore/internal/repository"

	"github.com/gorilla/mux"
)

// UTF-8 BOM for Excel compatibility
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ExportCSV handles CSV export requests based on export type
func ExportCSV(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	exportType := vars["type"]

	var csvData []byte
	var filename string
	var err error

	timestamp := time.Now().Format("2006-01-02")

	switch exportType {
	case "products":
		csvData, err = exportProducts()
		filename = fmt.Sprintf("products_%s.csv", timestamp)
	case "products-with-count":
		csvData, err = exportProductsWithDeviceCount()
		filename = fmt.Sprintf("products_with_count_%s.csv", timestamp)
	case "products-with-brand":
		csvData, err = exportProductsWithBrandManufacturer()
		filename = fmt.Sprintf("products_with_brand_%s.csv", timestamp)
	case "devices":
		csvData, err = exportAllDevices()
		filename = fmt.Sprintf("devices_%s.csv", timestamp)
	case "manufacturers":
		csvData, err = exportManufacturers()
		filename = fmt.Sprintf("manufacturers_%s.csv", timestamp)
	case "manufacturers-with-brands":
		csvData, err = exportManufacturersWithBrands()
		filename = fmt.Sprintf("manufacturers_with_brands_%s.csv", timestamp)
	case "brands":
		csvData, err = exportBrands()
		filename = fmt.Sprintf("brands_%s.csv", timestamp)
	case "zones":
		csvData, err = exportStorageZones()
		filename = fmt.Sprintf("storage_zones_%s.csv", timestamp)
	case "cables":
		csvData, err = exportProductsWithCableFields()
		filename = fmt.Sprintf("cables_%s.csv", timestamp)
	case "jobs":
		csvData, err = exportJobs()
		filename = fmt.Sprintf("jobs_%s.csv", timestamp)
	default:
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid export type"})
		return
	}

	if err != nil {
		log.Printf("[EXPORT] Error generating CSV for type %s: %v", exportType, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(csvData)))

	// Write UTF-8 BOM + CSV data
	w.Write(csvData)
}

// Helper function to create CSV with BOM
func createCSV(headers []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer

	// Write UTF-8 BOM for Excel compatibility
	buf.Write(utf8BOM)

	writer := csv.NewWriter(&buf)
	writer.Comma = ';' // Use semicolon as CSV separator

	// Write headers
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	// Write rows
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// formatBool converts boolean to Yes/No
func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// formatNullString handles NULL string values
func formatNullString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// formatNullInt handles NULL int values
func formatNullInt(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}

// formatNullFloat handles NULL float values
func formatNullFloat(f *float64) string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *f)
}

// formatDate formats time to date format (DD.MM.YYYY HH:MM)
func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("02.01.2006 15:04")
}

// ============================
// EXPORT FUNCTIONS
// ============================

// exportProducts exports all products with basic information
func exportProducts() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			p.productID,
			p.name,
			p.description,
			c.name as category_name,
			sc.name as subcategory_name,
			p.is_accessory,
			p.is_consumable,
			p.item_cost_per_day,
			p.weight,
			p.height,
			p.width,
			p.depth,
			p.power_consumption,
			p.generic_barcode
		FROM products p
		LEFT JOIN categories c ON p.categoryID = c.categoryid
		LEFT JOIN subcategories sc ON p.subcategoryID = sc.subcategoryid
		ORDER BY p.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"Product-ID", "Name", "Description", "Category", "Subcategory",
		"Accessory", "Consumable", "Cost/Day", "Weight (kg)", "Height (cm)",
		"Width (cm)", "Depth (cm)", "Power Consumption (W)", "Barcode",
	}

	var csvRows [][]string

	for rows.Next() {
		var productID int
		var name string
		var description, categoryName, subcategoryName, genericBarcode *string
		var isAccessory, isConsumable bool
		var itemCostPerDay, weight, height, width, depth, powerConsumption *float64

		err := rows.Scan(
			&productID, &name, &description, &categoryName, &subcategoryName,
			&isAccessory, &isConsumable, &itemCostPerDay, &weight, &height,
			&width, &depth, &powerConsumption, &genericBarcode,
		)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			strconv.Itoa(productID),
			name,
			formatNullString(description),
			formatNullString(categoryName),
			formatNullString(subcategoryName),
			formatBool(isAccessory),
			formatBool(isConsumable),
			formatNullFloat(itemCostPerDay),
			formatNullFloat(weight),
			formatNullFloat(height),
			formatNullFloat(width),
			formatNullFloat(depth),
			formatNullFloat(powerConsumption),
			formatNullString(genericBarcode),
		})
	}

	return createCSV(headers, csvRows)
}

// exportProductsWithDeviceCount exports products with their device counts
func exportProductsWithDeviceCount() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			p.productID,
			p.name,
			c.name as category_name,
			COUNT(d.deviceID) as device_count,
			SUM(CASE WHEN d.status = 'available' THEN 1 ELSE 0 END) as available_count,
			SUM(CASE WHEN d.status = 'in_use' THEN 1 ELSE 0 END) as in_use_count,
			SUM(CASE WHEN d.status = 'defect' THEN 1 ELSE 0 END) as defect_count
		FROM products p
		LEFT JOIN categories c ON p.categoryID = c.categoryid
		LEFT JOIN devices d ON p.productID = d.productID
		GROUP BY p.productID, p.name, c.name
		ORDER BY p.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"Product-ID", "Name", "Category", "Total Devices",
		"Available", "In Use", "Defective",
	}

	var csvRows [][]string

	for rows.Next() {
		var productID int
		var name string
		var categoryName *string
		var deviceCount, availableCount, inUseCount, defectCount int

		err := rows.Scan(
			&productID, &name, &categoryName, &deviceCount,
			&availableCount, &inUseCount, &defectCount,
		)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			strconv.Itoa(productID),
			name,
			formatNullString(categoryName),
			strconv.Itoa(deviceCount),
			strconv.Itoa(availableCount),
			strconv.Itoa(inUseCount),
			strconv.Itoa(defectCount),
		})
	}

	return createCSV(headers, csvRows)
}

// exportProductsWithBrandManufacturer exports products with brand and manufacturer info
func exportProductsWithBrandManufacturer() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			p.productID,
			p.name,
			c.name as category_name,
			m.name as manufacturer_name,
			b.name as brand_name,
			p.description,
			p.itemcostperday
		FROM products p
		LEFT JOIN categories c ON p.categoryID = c.categoryid
		LEFT JOIN manufacturer m ON p.manufacturerID = m.manufacturerid
		LEFT JOIN brands b ON p.brandID = b.brandid
		ORDER BY p.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"Product-ID", "Name", "Category", "Manufacturer", "Brand",
		"Description", "Cost/Day",
	}

	var csvRows [][]string

	for rows.Next() {
		var productID int
		var name string
		var categoryName, manufacturerName, brandName, description *string
		var itemCostPerDay *float64

		err := rows.Scan(
			&productID, &name, &categoryName, &manufacturerName,
			&brandName, &description, &itemCostPerDay,
		)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			strconv.Itoa(productID),
			name,
			formatNullString(categoryName),
			formatNullString(manufacturerName),
			formatNullString(brandName),
			formatNullString(description),
			formatNullFloat(itemCostPerDay),
		})
	}

	return createCSV(headers, csvRows)
}

// exportAllDevices exports all devices with full details
func exportAllDevices() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			d.deviceID,
			p.name as product_name,
			d.serialnumber,
			d.status,
			d.purchasedate,
			NULL::numeric AS purchase_price,
			d.lastmaintenance,
			d.notes,
			z.name as zone_name,
			c.name as case_name
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		LEFT JOIN cases c ON d.current_case_id = c.caseid
		ORDER BY d.deviceID
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"Device-ID", "Product", "Serial Number", "Status", "Purchase Date",
		"Purchase Price", "Last Maintenance", "Notes", "Storage Area", "Case",
	}

	var csvRows [][]string

	for rows.Next() {
		var deviceID, productName string
		var serialNumber, notes, zoneName, caseName *string
		var status string
		var purchaseDate, lastMaintenanceDate *time.Time
		var purchasePrice *float64

		err := rows.Scan(
			&deviceID, &productName, &serialNumber, &status, &purchaseDate,
			&purchasePrice, &lastMaintenanceDate, &notes, &zoneName, &caseName,
		)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			deviceID,
			productName,
			formatNullString(serialNumber),
			status,
			formatDate(purchaseDate),
			formatNullFloat(purchasePrice),
			formatDate(lastMaintenanceDate),
			formatNullString(notes),
			formatNullString(zoneName),
			formatNullString(caseName),
		})
	}

	return createCSV(headers, csvRows)
}

// exportManufacturers exports all manufacturers
func exportManufacturers() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			manufacturerid,
			name,
			website
		FROM manufacturer
		ORDER BY name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"ID", "Name", "Website",
	}

	var csvRows [][]string

	for rows.Next() {
		var id int
		var name string
		var website *string

		err := rows.Scan(&id, &name, &website)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			strconv.Itoa(id),
			name,
			formatNullString(website),
		})
	}

	return createCSV(headers, csvRows)
}

// exportManufacturersWithBrands exports manufacturers with their brands
func exportManufacturersWithBrands() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			m.manufacturerid,
			m.name as manufacturer_name,
			STRING_AGG(b.name, ', ') as brands
		FROM manufacturer m
		LEFT JOIN brands b ON m.manufacturerid = b.manufacturerid
		GROUP BY m.manufacturerid, m.name
		ORDER BY m.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"ID", "Manufacturer", "Brands",
	}

	var csvRows [][]string

	for rows.Next() {
		var id int
		var manufacturerName string
		var brands *string

		err := rows.Scan(&id, &manufacturerName, &brands)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			strconv.Itoa(id),
			manufacturerName,
			formatNullString(brands),
		})
	}

	return createCSV(headers, csvRows)
}

// exportBrands exports all brands
func exportBrands() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			b.brandid,
			b.name,
			m.name as manufacturer_name
		FROM brands b
		LEFT JOIN manufacturer m ON b.manufacturerid = m.manufacturerid
		ORDER BY b.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"ID", "Brand Name", "Manufacturer",
	}

	var csvRows [][]string

	for rows.Next() {
		var id int
		var name string
		var manufacturerName *string

		err := rows.Scan(&id, &name, &manufacturerName)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			strconv.Itoa(id),
			name,
			formatNullString(manufacturerName),
		})
	}

	return createCSV(headers, csvRows)
}

// exportStorageZones exports all storage zones with details
func exportStorageZones() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			z.zone_id,
			z.name,
			zt.label as zone_type,
			z.barcode,
			z.capacity,
			z.location,
			z.notes,
			COUNT(DISTINCT d.deviceID) as device_count
		FROM storage_zones z
		LEFT JOIN zone_types zt ON z.zone_type = zt.key
		LEFT JOIN devices d ON z.zone_id = d.current_zone_id
		GROUP BY z.zone_id, z.name, zt.label, z.barcode, z.capacity, z.location, z.notes
		ORDER BY z.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"Zone-ID", "Name", "Type", "Barcode", "Capacity",
		"Location", "Notes", "Device Count",
	}

	var csvRows [][]string

	for rows.Next() {
		var zoneID int
		var name string
		var zoneType, barcode, location, notes *string
		var capacity *int
		var deviceCount int

		err := rows.Scan(
			&zoneID, &name, &zoneType, &barcode, &capacity,
			&location, &notes, &deviceCount,
		)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			strconv.Itoa(zoneID),
			name,
			formatNullString(zoneType),
			formatNullString(barcode),
			formatNullInt(capacity),
			formatNullString(location),
			formatNullString(notes),
			strconv.Itoa(deviceCount),
		})
	}

	return createCSV(headers, csvRows)
}

// exportProductsWithCableFields exports products that have cable-related custom field values.
// Cables are now modelled as products with product_field_values entries.
func exportProductsWithCableFields() ([]byte, error) {
	db := repository.GetSQLDB()

	// Fetch products that have at least one cable-related field value
	query := `
		SELECT DISTINCT
			p.productID,
			p.name,
			MAX(CASE WHEN pfd.name = 'connector_1' THEN pfv.value END) AS connector_1,
			MAX(CASE WHEN pfd.name = 'connector_2' THEN pfv.value END) AS connector_2,
			MAX(CASE WHEN pfd.name = 'cable_type'  THEN pfv.value END) AS cable_type,
			MAX(CASE WHEN pfd.name = 'cable_length' THEN pfv.value END) AS cable_length,
			MAX(CASE WHEN pfd.name = 'cable_mm2'   THEN pfv.value END) AS cable_mm2
		FROM products p
		JOIN product_field_values pfv ON pfv.product_id = p.productID
		JOIN product_field_definitions pfd ON pfd.id = pfv.field_definition_id
		WHERE pfd.name IN ('connector_1', 'connector_2', 'cable_type', 'cable_length', 'cable_mm2')
		GROUP BY p.productID, p.name
		ORDER BY p.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"Product ID", "Name", "Connector 1", "Connector 2", "Cable Type",
		"Length (m)", "Cross-section (mm²)",
	}

	var csvRows [][]string
	for rows.Next() {
		var id int
		var name string
		var connector1, connector2, cableType, cableLength, cableMM2 *string
		if err := rows.Scan(&id, &name, &connector1, &connector2, &cableType, &cableLength, &cableMM2); err != nil {
			return nil, err
		}
		csvRows = append(csvRows, []string{
			strconv.Itoa(id),
			name,
			formatNullString(connector1),
			formatNullString(connector2),
			formatNullString(cableType),
			formatNullString(cableLength),
			formatNullString(cableMM2),
		})
	}

	return createCSV(headers, csvRows)
}

// exportJobs exports all jobs with complete information
func exportJobs() ([]byte, error) {
	db := repository.GetSQLDB()

	query := `
		SELECT
			j.jobID,
			j.job_number,
			j.title,
			j.customer_name,
			j.start_date,
			j.end_date,
			j.status,
			j.location,
			j.notes,
			COUNT(DISTINCT jd.deviceID) as device_count,
			SUM(jd.quantity) as total_quantity
		FROM jobs j
		LEFT JOIN jobdevices jd ON j.jobID = jd.jobID
		GROUP BY j.jobID, j.job_number, j.title, j.customer_name,
		         j.start_date, j.end_date, j.status, j.location, j.notes
		ORDER BY j.start_date DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := []string{
		"Job-ID", "Job-Number", "Title", "Customer", "Start Date",
		"End Date", "Status", "Location", "Notes", "Device Type Count", "Total Quantity",
	}

	var csvRows [][]string

	for rows.Next() {
		var jobID int
		var jobNumber, title string
		var customerName, status, location, notes *string
		var startDate, endDate *time.Time
		var deviceCount, totalQuantity int

		err := rows.Scan(
			&jobID, &jobNumber, &title, &customerName, &startDate,
			&endDate, &status, &location, &notes, &deviceCount, &totalQuantity,
		)
		if err != nil {
			return nil, err
		}

		csvRows = append(csvRows, []string{
			strconv.Itoa(jobID),
			jobNumber,
			title,
			formatNullString(customerName),
			formatDate(startDate),
			formatDate(endDate),
			formatNullString(status),
			formatNullString(location),
			formatNullString(notes),
			strconv.Itoa(deviceCount),
			strconv.Itoa(totalQuantity),
		})
	}

	return createCSV(headers, csvRows)
}
