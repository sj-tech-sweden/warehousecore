package services

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"storagecore/internal/repository"
)

// ZoneService handles zone-related business logic
type ZoneService struct {
	db *sql.DB
}

// NewZoneService creates a new zone service
func NewZoneService() *ZoneService {
	return &ZoneService{
		db: repository.GetDB(),
	}
}

// GenerateZoneCode generates a smart, hierarchical zone code
// Examples:
// - Warehouse: WDB (from "Weidelbach")
// - Rack in WDB: WDB-RG-01
// - Shelf in Rack: WDB-RG-01-F-01
// - Vehicle: WDB-VH-01
func (s *ZoneService) GenerateZoneCode(zoneName, zoneType string, parentZoneID *int64) (string, error) {
	// If there's a parent, get its code and build hierarchically
	if parentZoneID != nil && *parentZoneID > 0 {
		var parentCode string
		err := s.db.QueryRow(`SELECT code FROM storage_zones WHERE zone_id = ?`, *parentZoneID).Scan(&parentCode)
		if err != nil {
			return "", fmt.Errorf("parent zone not found: %v", err)
		}

		// Get next number for this type under this parent
		typePrefix := getTypePrefix(zoneType)
		pattern := fmt.Sprintf("%s-%s-%%", parentCode, typePrefix)

		var maxNum int
		err = s.db.QueryRow(`
			SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(code, '-', -1) AS UNSIGNED)), 0)
			FROM storage_zones
			WHERE code LIKE ? AND parent_zone_id = ?
		`, pattern, *parentZoneID).Scan(&maxNum)

		if err != nil && err != sql.ErrNoRows {
			return "", err
		}

		return fmt.Sprintf("%s-%s-%02d", parentCode, typePrefix, maxNum+1), nil
	}

	// Root level zone (warehouse)
	prefix := generatePrefix(zoneName)

	// Check if this prefix exists and add number if needed
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM storage_zones WHERE code LIKE ?
	`, prefix+"%").Scan(&count)

	if err != nil {
		return "", err
	}

	if count == 0 {
		return prefix, nil
	}

	return fmt.Sprintf("%s-%02d", prefix, count+1), nil
}

// generatePrefix creates a 3-letter prefix from a zone name
func generatePrefix(name string) string {
	// Remove special characters and convert to uppercase
	reg := regexp.MustCompile("[^a-zA-Z]+")
	cleaned := reg.ReplaceAllString(name, "")
	cleaned = strings.ToUpper(cleaned)

	if len(cleaned) <= 3 {
		return cleaned
	}

	// Take first 3 consonants or first 3 letters
	consonants := regexp.MustCompile("[BCDFGHJKLMNPQRSTVWXYZ]")
	matches := consonants.FindAllString(cleaned, -1)

	if len(matches) >= 3 {
		return strings.Join(matches[:3], "")
	}

	return cleaned[:3]
}

// getTypePrefix returns the type prefix for zone codes
func getTypePrefix(zoneType string) string {
	prefixes := map[string]string{
		"warehouse": "LGR", // Lager
		"rack":      "RG",  // Regal
		"gitterbox": "GB",  // Gitterbox
	}

	if prefix, ok := prefixes[zoneType]; ok {
		return prefix
	}
	return "OT"
}

// GetZoneDetails returns zone with subzones and device count
func (s *ZoneService) GetZoneDetails(zoneID int64) (map[string]interface{}, error) {
	// Get zone info
	var (
		code, name, zoneType string
		description          sql.NullString
		parentZoneID         sql.NullInt64
		capacity             sql.NullInt64
		isActive             bool
	)

	err := s.db.QueryRow(`
		SELECT zone_id, code, name, type, description, parent_zone_id, capacity, is_active
		FROM storage_zones
		WHERE zone_id = ? AND is_active = TRUE
	`, zoneID).Scan(&zoneID, &code, &name, &zoneType, &description, &parentZoneID, &capacity, &isActive)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("zone not found")
	}
	if err != nil {
		return nil, err
	}

	// Get subzones
	subzones, err := s.getSubzones(zoneID)
	if err != nil {
		return nil, err
	}

	// Get devices in this zone
	var deviceCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM devices WHERE zone_id = ? AND status = 'in_storage'`, zoneID).Scan(&deviceCount)

	// Get breadcrumb path
	breadcrumb, err := s.getBreadcrumb(zoneID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"zone_id":      zoneID,
		"code":         code,
		"name":         name,
		"type":         zoneType,
		"is_active":    isActive,
		"device_count": deviceCount,
		"subzones":     subzones,
		"breadcrumb":   breadcrumb,
	}

	if description.Valid {
		result["description"] = description.String
	}
	if capacity.Valid {
		result["capacity"] = capacity.Int64
	}
	if parentZoneID.Valid {
		result["parent_zone_id"] = parentZoneID.Int64
	}

	return result, nil
}

// getSubzones returns all child zones
func (s *ZoneService) getSubzones(parentID int64) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(`
		SELECT zone_id, code, name, type, capacity,
		       (SELECT COUNT(*) FROM devices WHERE zone_id = sz.zone_id AND status = 'in_storage') as device_count,
		       (SELECT COUNT(*) FROM storage_zones WHERE parent_zone_id = sz.zone_id AND is_active = TRUE) as subzone_count
		FROM storage_zones sz
		WHERE parent_zone_id = ? AND is_active = TRUE
		ORDER BY code
	`, parentID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subzones := []map[string]interface{}{}
	for rows.Next() {
		var (
			zoneID, deviceCount, subzoneCount int64
			code, name, zoneType              string
			capacity                          sql.NullInt64
		)

		if err := rows.Scan(&zoneID, &code, &name, &zoneType, &capacity, &deviceCount, &subzoneCount); err != nil {
			continue
		}

		zone := map[string]interface{}{
			"zone_id":       zoneID,
			"code":          code,
			"name":          name,
			"type":          zoneType,
			"device_count":  deviceCount,
			"subzone_count": subzoneCount,
		}

		if capacity.Valid {
			zone["capacity"] = capacity.Int64
		}

		subzones = append(subzones, zone)
	}

	return subzones, nil
}

// getBreadcrumb returns the hierarchical path to a zone
func (s *ZoneService) getBreadcrumb(zoneID int64) ([]map[string]interface{}, error) {
	breadcrumb := []map[string]interface{}{}
	currentID := zoneID

	for {
		var (
			id, code, name string
			parentID       sql.NullInt64
		)

		err := s.db.QueryRow(`
			SELECT zone_id, code, name, parent_zone_id
			FROM storage_zones
			WHERE zone_id = ?
		`, currentID).Scan(&id, &code, &name, &parentID)

		if err != nil {
			break
		}

		breadcrumb = append([]map[string]interface{}{{
			"zone_id": id,
			"code":    code,
			"name":    name,
		}}, breadcrumb...)

		if !parentID.Valid {
			break
		}
		currentID = parentID.Int64
	}

	return breadcrumb, nil
}
