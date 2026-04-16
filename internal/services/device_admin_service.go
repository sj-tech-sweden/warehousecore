package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

type DeviceAdminService struct {
	db           *sql.DB
	labelService *LabelService
}

// NewDeviceAdminService constructs a device admin service using the global repositories.
func NewDeviceAdminService() *DeviceAdminService {
	return &DeviceAdminService{
		db:           repository.GetSQLDB(),
		labelService: NewLabelService(),
	}
}

// CreateDevices inserts one or multiple devices and returns their hydrated representations.
func (s *DeviceAdminService) CreateDevices(ctx context.Context, input *models.DeviceCreateInput) ([]*models.DeviceWithDetails, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}
	if input.ProductID <= 0 {
		return nil, errors.New("product_id is required")
	}

	if input.Quantity <= 0 {
		input.Quantity = 1
	}
	if input.Quantity > 100 {
		return nil, errors.New("quantity cannot exceed 100")
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status == "" || status == "free" {
		status = string(models.StatusInStorage)
	}

	autoGenerateLabel := true
	if input.AutoGenerateLabel != nil {
		autoGenerateLabel = *input.AutoGenerateLabel
	}

	regenerateCodes := false
	if input.RegenerateCodes != nil {
		regenerateCodes = *input.RegenerateCodes
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Determine the device ID prefix and the next available counter via the
	// shared helpers in device_id.go (see that file for full documentation):
	//   - DeriveDeviceIDPrefix honours a caller-supplied device_prefix or derives
	//     one from subcategory abbreviation + pos_in_category, with a
	//     "P{productID}" fallback for products without an abbreviation.
	//   - AllocateDeviceCounter acquires a pg_advisory_xact_lock keyed on an
	//     FNV-32a hash of the prefix and scans existing IDs with an index-friendly
	//     LIKE predicate to find the next counter value.
	manualPrefix := ""
	if input.DevicePrefix != nil {
		manualPrefix = *input.DevicePrefix
	}
	prefix, err := DeriveDeviceIDPrefix(ctx, tx, input.ProductID, manualPrefix)
	if err != nil {
		return nil, err
	}

	nextCounter, err := AllocateDeviceCounter(ctx, tx, prefix)
	if err != nil {
		return nil, err
	}

	createdIDs := make([]string, 0, input.Quantity)
	providedBarcode := input.Barcode != nil && strings.TrimSpace(*input.Barcode) != ""
	providedQRCode := input.QRCode != nil && strings.TrimSpace(*input.QRCode) != ""

	for i := 0; i < input.Quantity; i++ {
		deviceID := fmt.Sprintf("%s%03d", prefix, nextCounter+int64(i))
		serialValue := serialForIndex(input.SerialNumber, input.StartingSerial, input.IncrementSerial, i)

		_, err := tx.ExecContext(ctx, `
			INSERT INTO devices (
				deviceID, productID, serialnumber, rfid, status, current_location, zone_id,
				condition_rating, usage_hours, purchaseDate, retire_date, warranty_end_date,
				lastmaintenance, nextmaintenance,
				notes, barcode, qr_code
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		`,
			deviceID,
			input.ProductID,
			nullableString(serialValue),
			nullableString(trimPtr(input.RFID)),
			status,
			nullableString(trimPtr(input.CurrentLocation)),
			nullableInt(input.ZoneID),
			derefFloatOr(input.ConditionRating, 0),
			derefFloatOr(input.UsageHours, 0),
			parseDatePtr(input.PurchaseDate),
			parseDatePtr(input.RetireDate),
			parseDatePtr(input.WarrantyEndDate),
			parseDatePtr(input.LastMaintenance),
			parseDatePtr(input.NextMaintenance),
			nullableText(input.Notes),
			nullableString(trimPtr(input.Barcode)),
			nullableString(trimPtr(input.QRCode)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert device: %w", err)
		}

		if err := s.ensureDeviceCodes(ctx, tx, deviceID, providedBarcode, providedQRCode, regenerateCodes); err != nil {
			return nil, err
		}

		createdIDs = append(createdIDs, deviceID)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit device creation: %w", err)
	}

	devices := make([]*models.DeviceWithDetails, 0, len(createdIDs))
	for _, id := range createdIDs {
		device, err := s.FetchDevice(ctx, id)
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}

	// Trigger async label generation after commit
	if autoGenerateLabel || input.LabelTemplateID != nil {
		templateID := input.LabelTemplateID
		for i := range createdIDs {
			deviceID := createdIDs[i]
			go func(id string) {
				if err := s.generateLabelForDevice(id, templateID); err != nil {
					log.Printf("[DEVICE LABEL] Failed to generate label for %s: %v", id, err)
				}
			}(deviceID)
		}
	}

	return devices, nil
}

// UpdateDevice updates an existing device and returns the updated record.
func (s *DeviceAdminService) UpdateDevice(ctx context.Context, deviceID string, input *models.DeviceUpdateInput) (*models.DeviceWithDetails, error) {
	if deviceID == "" {
		return nil, errors.New("deviceID required")
	}
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	setClauses := make([]string, 0, 16)
	args := make([]interface{}, 0, 16)

	addArg := func(clause string, value interface{}) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf(clause, len(args)))
	}

	if input.ProductID.Set {
		if input.ProductID.Valid {
			addArg("productID = $%d", input.ProductID.Value)
		} else {
			addArg("productID = $%d", nil)
		}
	}

	if input.Status.Set {
		if input.Status.Valid {
			normalizedStatus := strings.ToLower(strings.TrimSpace(input.Status.Value))
			if normalizedStatus == "free" {
				normalizedStatus = string(models.StatusInStorage)
			}
			addArg("status = $%d", normalizedStatus)
		} else {
			addArg("status = $%d", nil)
		}
	}

	if input.SerialNumber.Set {
		if input.SerialNumber.Valid {
			addArg("serialnumber = $%d", nullableStringPtr(&input.SerialNumber.Value))
		} else {
			addArg("serialnumber = $%d", nil)
		}
	}

	if input.RFID.Set {
		if input.RFID.Valid {
			addArg("rfid = $%d", nullableStringPtr(&input.RFID.Value))
		} else {
			addArg("rfid = $%d", nil)
		}
	}

	if input.Barcode.Set {
		if input.Barcode.Valid {
			addArg("barcode = $%d", nullableStringPtr(&input.Barcode.Value))
		} else {
			addArg("barcode = $%d", nil)
		}
	}

	if input.QRCode.Set {
		if input.QRCode.Valid {
			addArg("qr_code = $%d", nullableStringPtr(&input.QRCode.Value))
		} else {
			addArg("qr_code = $%d", nil)
		}
	}

	if input.CurrentLocation.Set {
		if input.CurrentLocation.Valid {
			addArg("current_location = $%d", nullableStringPtr(&input.CurrentLocation.Value))
		} else {
			addArg("current_location = $%d", nil)
		}
	}

	if input.ZoneID.Set {
		if input.ZoneID.Valid {
			id := input.ZoneID.Value
			addArg("zone_id = $%d", &id)
		} else {
			addArg("zone_id = $%d", nil)
		}
	}

	if input.ConditionRating.Set {
		if input.ConditionRating.Valid {
			addArg("condition_rating = $%d", input.ConditionRating.Value)
		} else {
			addArg("condition_rating = $%d", 0.0)
		}
	}

	if input.UsageHours.Set {
		if input.UsageHours.Valid {
			addArg("usage_hours = $%d", input.UsageHours.Value)
		} else {
			addArg("usage_hours = $%d", 0.0)
		}
	}

	if input.PurchaseDate.Set {
		if input.PurchaseDate.Valid {
			addArg("purchaseDate = $%d", parseDateValue(input.PurchaseDate.Value))
		} else {
			addArg("purchaseDate = $%d", nil)
		}
	}

	if input.RetireDate.Set {
		if input.RetireDate.Valid {
			addArg("retire_date = $%d", parseDateValue(input.RetireDate.Value))
		} else {
			addArg("retire_date = $%d", nil)
		}
	}

	if input.WarrantyEndDate.Set {
		if input.WarrantyEndDate.Valid {
			addArg("warranty_end_date = $%d", parseDateValue(input.WarrantyEndDate.Value))
		} else {
			addArg("warranty_end_date = $%d", nil)
		}
	}

	if input.LastMaintenance.Set {
		if input.LastMaintenance.Valid {
			addArg("lastmaintenance = $%d", parseDateValue(input.LastMaintenance.Value))
		} else {
			addArg("lastmaintenance = $%d", nil)
		}
	}

	if input.NextMaintenance.Set {
		if input.NextMaintenance.Valid {
			addArg("nextmaintenance = $%d", parseDateValue(input.NextMaintenance.Value))
		} else {
			addArg("nextmaintenance = $%d", nil)
		}
	}

	if input.Notes.Set {
		if input.Notes.Valid {
			addArg("notes = $%d", nullableStringPtr(&input.Notes.Value))
		} else {
			addArg("notes = $%d", nil)
		}
	}

	// Handle device ID rename
	newDeviceID := deviceID
	if input.NewDeviceID.Set && input.NewDeviceID.Valid {
		trimmed := strings.TrimSpace(input.NewDeviceID.Value)
		if trimmed != "" && trimmed != deviceID {
			addArg("deviceID = $%d", trimmed)
			newDeviceID = trimmed
		}
	}

	if len(setClauses) == 0 {
		return nil, errors.New("no fields provided for update")
	}

	args = append(args, deviceID)
	query := fmt.Sprintf("UPDATE devices SET %s WHERE deviceID = $%d", strings.Join(setClauses, ", "), len(args))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	// If device ID is being renamed, update all child tables that reference it
	if newDeviceID != deviceID {
		childUpdates := []string{
			"UPDATE device_movements SET device_id = $1 WHERE device_id = $2",
			"UPDATE scan_events SET device_id = $1 WHERE device_id = $2",
			"UPDATE defect_reports SET device_id = $1 WHERE device_id = $2",
			"UPDATE inspection_schedules SET device_id = $1 WHERE device_id = $2",
			"UPDATE jobdevices SET deviceID = $1 WHERE deviceID = $2",
			"UPDATE devicescases SET deviceID = $1 WHERE deviceID = $2",
		}
		for _, stmt := range childUpdates {
			if _, err := tx.ExecContext(ctx, stmt, newDeviceID, deviceID); err != nil {
				return nil, fmt.Errorf("failed to update device references: %w", err)
			}
		}
	}

	shouldResetCodes := input.RegenerateCodes.Set && input.RegenerateCodes.Valid && input.RegenerateCodes.Value
	if shouldResetCodes {
		if err := s.resetDeviceCodes(ctx, tx, newDeviceID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit device update: %w", err)
	}

	if shouldResetCodes {
		log.Printf("[DEVICE] Regenerated codes for %s", newDeviceID)
	}

	if input.RegenerateLabel.Set && input.RegenerateLabel.Valid && input.RegenerateLabel.Value {
		templateID := input.LabelTemplateID.Ptr()
		go func() {
			if err := s.generateLabelForDevice(newDeviceID, templateID); err != nil {
				log.Printf("[DEVICE LABEL] Failed to regenerate label for %s: %v", newDeviceID, err)
			}
		}()
	}

	return s.FetchDevice(ctx, newDeviceID)
}

// deleteDeviceInTx deletes a single device within the provided transaction.
// It returns the device's label_path (if any) for post-commit cleanup.
// Uses DELETE ... RETURNING to atomically delete and retrieve the label path.
func (s *DeviceAdminService) deleteDeviceInTx(ctx context.Context, tx *sql.Tx, deviceID string) (string, error) {
	var labelPath sql.NullString
	err := tx.QueryRowContext(ctx, `DELETE FROM devices WHERE deviceID = $1 RETURNING label_path`, deviceID).Scan(&labelPath)
	if errors.Is(err, sql.ErrNoRows) {
		return "", repository.ErrNotFound
	}
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" { // foreign_key_violation
			return "", fmt.Errorf("device %s is still linked to cases, jobs, or history entries", deviceID)
		}
		return "", fmt.Errorf("failed to delete device: %w", err)
	}

	if labelPath.Valid {
		return labelPath.String, nil
	}
	return "", nil
}

// labelBaseDir is the base directory for label files.
// It defaults to "web/dist" and can be overridden in tests.
var labelBaseDir = filepath.Join("web", "dist")

// RemoveLabelFile removes a device label file from disk.
// Paths are sanitized to prevent path traversal outside the label base directory.
func RemoveLabelFile(labelPath string) {
	if labelPath == "" {
		return
	}
	baseDir, err := filepath.Abs(labelBaseDir)
	if err != nil {
		log.Printf("[DEVICE] Failed to resolve base dir for label cleanup: %v", err)
		return
	}
	cleaned := filepath.Clean(strings.TrimPrefix(labelPath, "/"))
	fullPath := filepath.Join(baseDir, cleaned)
	if !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
		log.Printf("[DEVICE] Skipping label path outside base dir: %s", labelPath)
		return
	}
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("[DEVICE] Failed to remove label %s: %v", fullPath, err)
	}
}

// DeleteDevice removes a device and its label file if no dependencies exist.
func (s *DeviceAdminService) DeleteDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return errors.New("deviceID required")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	labelPath, err := s.deleteDeviceInTx(ctx, tx, deviceID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit delete: %w", err)
	}

	RemoveLabelFile(labelPath)
	return nil
}

// sanitizeDeviceDeleteError converts deleteDeviceInTx errors into user-friendly messages
// suitable for API responses. Known error types (not found, FK violation) already have
// safe messages; anything else is replaced with a generic message to avoid leaking
// internal DB/driver details (table names, constraint names, etc.).
func sanitizeDeviceDeleteError(err error) string {
	if errors.Is(err, repository.ErrNotFound) {
		return "device not found"
	}
	msg := err.Error()
	// The FK-violation branch in deleteDeviceInTx produces a known user-friendly message.
	if strings.Contains(msg, "still linked to") {
		return msg
	}
	return "internal error deleting device"
}

// BulkDeleteDeviceResult contains the results of a bulk device deletion.
type BulkDeleteDeviceResult struct {
	Deleted      int
	Failed       int
	FailedIDs    []string
	FailedErrors map[string]string // per-device failure reasons (device ID → error message)
}

// BulkDeleteDevices deletes multiple devices within a single transaction.
// Each device deletion uses a SAVEPOINT so individual failures (FK violations, not found)
// do not abort the entire transaction. Label files are removed after a successful commit.
func (s *DeviceAdminService) BulkDeleteDevices(ctx context.Context, ids []string) (*BulkDeleteDeviceResult, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	labelPaths := make([]string, 0)
	failedIDs := make([]string, 0)
	failedErrors := make(map[string]string)
	deleted := 0

	for i, id := range ids {
		sp := fmt.Sprintf("device_delete_%d", i)
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SAVEPOINT %s", sp)); err != nil {
			log.Printf("[BULK DEVICE DELETE] Failed to create savepoint for %s: %v", id, err)
			failedIDs = append(failedIDs, id)
			failedErrors[id] = "internal error processing device"
			continue
		}

		lp, err := s.deleteDeviceInTx(ctx, tx, id)
		if err != nil {
			log.Printf("[BULK DEVICE DELETE] Failed for %s: %v", id, err)
			failedIDs = append(failedIDs, id)
			failedErrors[id] = sanitizeDeviceDeleteError(err)
			if _, rbErr := tx.ExecContext(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", sp)); rbErr != nil {
				log.Printf("[BULK DEVICE DELETE] Failed to rollback savepoint for %s: %v", id, rbErr)
			} else if _, relErr := tx.ExecContext(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", sp)); relErr != nil {
				log.Printf("[BULK DEVICE DELETE] Failed to release savepoint after rollback for %s: %v", id, relErr)
			}
			continue
		}

		if _, err := tx.ExecContext(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", sp)); err != nil {
			log.Printf("[BULK DEVICE DELETE] Failed to release savepoint for %s: %v", id, err)
		}
		deleted++
		if lp != "" {
			labelPaths = append(labelPaths, lp)
		}
	}

	if deleted > 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit bulk delete: %w", err)
		}
		// Clean up label files after successful commit
		for _, lp := range labelPaths {
			RemoveLabelFile(lp)
		}
	}

	return &BulkDeleteDeviceResult{
		Deleted:      deleted,
		Failed:       len(failedIDs),
		FailedIDs:    failedIDs,
		FailedErrors: failedErrors,
	}, nil
}

// BulkUpdateDeviceInput holds the optional fields that can be bulk-updated on devices.
type BulkUpdateDeviceInput struct {
	Status          *string
	ZoneID          *int
	CurrentLocation *string
	ConditionRating *float64
}

// BulkUpdateDeviceResult contains the results of a bulk device update.
type BulkUpdateDeviceResult struct {
	Updated int
}

// BulkUpdateDevices updates common fields on multiple devices within a single transaction.
// It applies the same normalization as UpdateDevice (e.g. nullableStringPtr for current_location,
// status trimming / "free" → "in_storage" consolidation) to keep behavior consistent.
func (s *DeviceAdminService) BulkUpdateDevices(ctx context.Context, ids []string, input *BulkUpdateDeviceInput) (*BulkUpdateDeviceResult, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	var setClauses []string
	var args []interface{}
	paramCount := 0

	if input.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*input.Status))
		if status == "" {
			return nil, errors.New("status cannot be empty")
		}
		if status == "free" {
			status = string(models.StatusInStorage)
		}
		paramCount++
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", paramCount))
		args = append(args, status)
	}
	if input.ZoneID != nil {
		paramCount++
		setClauses = append(setClauses, fmt.Sprintf("zone_id = $%d", paramCount))
		args = append(args, *input.ZoneID)
	}
	if input.CurrentLocation != nil {
		paramCount++
		setClauses = append(setClauses, fmt.Sprintf("current_location = $%d", paramCount))
		args = append(args, nullableStringPtr(input.CurrentLocation))
	}
	if input.ConditionRating != nil {
		paramCount++
		setClauses = append(setClauses, fmt.Sprintf("condition_rating = $%d", paramCount))
		args = append(args, *input.ConditionRating)
	}

	if len(setClauses) == 0 {
		return nil, errors.New("no fields to update")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	totalUpdated := 0
	for _, id := range ids {
		updateArgs := make([]interface{}, len(args))
		copy(updateArgs, args)
		updateArgs = append(updateArgs, id)
		query := fmt.Sprintf("UPDATE devices SET %s WHERE deviceID = $%d",
			strings.Join(setClauses, ", "), paramCount+1)
		result, err := tx.ExecContext(ctx, query, updateArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to update device %s: %w", id, err)
		}
		affected, _ := result.RowsAffected()
		totalUpdated += int(affected)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[BULK DEVICE UPDATE] Updated %d devices", totalUpdated)
	return &BulkUpdateDeviceResult{Updated: totalUpdated}, nil
}

// RegenerateLabel allows manual regeneration of a device label using the default or provided template.
func (s *DeviceAdminService) RegenerateLabel(deviceID string, templateID *int) error {
	if deviceID == "" {
		return errors.New("deviceID required")
	}
	return s.generateLabelForDevice(deviceID, templateID)
}

// FetchDevice loads a device with related metadata for API responses.
func (s *DeviceAdminService) FetchDevice(ctx context.Context, deviceID string) (*models.DeviceWithDetails, error) {
	var device models.DeviceWithDetails
	err := s.db.QueryRowContext(ctx, `
		SELECT d.deviceID, d.productID, d.serialnumber, d.rfid, d.barcode, d.qr_code, d.status,
		       d.current_location, d.zone_id,
		       COALESCE(d.condition_rating, 0), COALESCE(d.usage_hours, 0), d.purchaseDate, d.retire_date, d.warranty_end_date,
		       d.lastmaintenance, d.nextmaintenance,
		       d.notes, d.label_path,
		       COALESCE(p.name, '') AS product_name,
		       COALESCE(cat.name, '') AS product_category,
		       COALESCE(z.name, '') AS zone_name,
		       COALESCE(z.code, '') AS zone_code,
		       dc.caseID,
		       COALESCE(c.name, '') AS case_name,
		       jd.jobID,
		       COALESCE(CAST(jd.jobID AS TEXT), '') AS job_number
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN categories cat ON p.categoryID = cat.categoryID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		LEFT JOIN devicescases dc ON d.deviceID = dc.deviceID
		LEFT JOIN cases c ON dc.caseID = c.caseID
		LEFT JOIN jobdevices jd ON d.deviceID = jd.deviceID
		WHERE d.deviceID = $1
		LIMIT 1
	`, deviceID).Scan(
		&device.DeviceID,
		&device.ProductID,
		&device.SerialNumber,
		&device.RFID,
		&device.Barcode,
		&device.QRCode,
		&device.Status,
		&device.CurrentLocation,
		&device.ZoneID,
		&device.ConditionRating,
		&device.UsageHours,
		&device.PurchaseDate,
		&device.RetireDate,
		&device.WarrantyEndDate,
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
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load device %s: %w", deviceID, err)
	}

	return &device, nil
}

func (s *DeviceAdminService) ensureDeviceCodes(ctx context.Context, tx *sql.Tx, deviceID string, hadBarcode bool, hadQR bool, regenerate bool) error {
	if regenerate || !hadBarcode || !hadQR {
		barcode := ""
		qr := ""
		if regenerate || !hadBarcode {
			barcode = deviceID
		}
		if regenerate || !hadQR {
			qr = fmt.Sprintf("QR-%s", deviceID)
		}

		columns := make([]string, 0, 2)
		args := make([]interface{}, 0, 2)

		if regenerate || !hadBarcode {
			args = append(args, barcode)
			columns = append(columns, fmt.Sprintf("barcode = $%d", len(args)))
		}
		if regenerate || !hadQR {
			args = append(args, qr)
			columns = append(columns, fmt.Sprintf("qr_code = $%d", len(args)))
		}
		if len(columns) == 0 {
			return nil
		}
		args = append(args, deviceID)
		_, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE devices SET %s WHERE deviceID = $%d", strings.Join(columns, ", "), len(args)), args...)
		if err != nil {
			return fmt.Errorf("failed to backfill device codes: %w", err)
		}
	}
	return nil
}

func (s *DeviceAdminService) resetDeviceCodes(ctx context.Context, tx *sql.Tx, deviceID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE devices
		SET barcode = $1, qr_code = $2
		WHERE deviceID = $3
	`, deviceID, fmt.Sprintf("QR-%s", deviceID), deviceID)
	if err != nil {
		return fmt.Errorf("failed to regenerate device codes: %w", err)
	}
	return nil
}

func (s *DeviceAdminService) generateLabelForDevice(deviceID string, templateID *int) error {
	if templateID == nil || *templateID <= 0 {
		return s.labelService.AutoGenerateDeviceLabel(deviceID)
	}

	labelData, err := s.labelService.GenerateLabelForDevice(deviceID, *templateID)
	if err != nil {
		return err
	}

	labelDataJSON, err := json.Marshal(labelData)
	if err != nil {
		return fmt.Errorf("failed to marshal label data: %w", err)
	}

	htmlTemplate, err := os.ReadFile("./internal/services/label_template.html")
	if err != nil {
		return fmt.Errorf("failed to load label template: %w", err)
	}

	htmlContent := strings.Replace(string(htmlTemplate), "{{LABEL_DATA_JSON}}", string(labelDataJSON), 1)

	base64PNG, err := s.labelService.renderLabelWithHeadlessBrowser(htmlContent)
	if err != nil {
		return fmt.Errorf("failed to render label: %w", err)
	}

	_, err = s.labelService.SaveLabelImage(deviceID, "data:image/png;base64,"+base64PNG)
	if err != nil {
		return fmt.Errorf("failed to save label image: %w", err)
	}

	return nil
}

// Helper conversions ---------------------------------------------------------

func serialForIndex(base *string, starting *int, increment bool, index int) *string {
	if base == nil || strings.TrimSpace(*base) == "" {
		return nil
	}
	value := strings.TrimSpace(*base)
	if !increment {
		return &value
	}
	start := 1
	if starting != nil && *starting > 0 {
		start = *starting
	}
	serial := fmt.Sprintf("%s-%02d", value, start+index)
	return &serial
}

func parseDatePtr(value *string) interface{} {
	if value == nil {
		return nil
	}
	return parseDateValue(*value)
}

func parseDateValue(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil
	}
	return t
}

func trimPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
