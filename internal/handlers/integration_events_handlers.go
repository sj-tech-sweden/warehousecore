package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"warehousecore/internal/repository"
)

type integrationEventEnvelope struct {
	EventID        string                 `json:"eventId"`
	SchemaVersion  int                    `json:"schemaVersion"`
	Source         string                 `json:"source"`
	EntityType     string                 `json:"entityType"`
	Action         string                 `json:"action"`
	OccurredAt     string                 `json:"occurredAt"`
	CorrelationID  string                 `json:"correlationId"`
	IdempotencyKey string                 `json:"idempotencyKey"`
	Entity         integrationEventEntity `json:"entity"`
}

type integrationEventEntity struct {
	ExternalID  string                 `json:"externalId"`
	WarehouseID interface{}            `json:"warehouseId"`
	Version     *int                   `json:"version"`
	Fields      map[string]interface{} `json:"fields"`
}

var allowedIntegrationSources = map[string]bool{
	"twenty":        true,
	"warehousecore": true,
}

var allowedIntegrationEntityTypes = map[string]bool{
	"customer":    true,
	"job":         true,
	"requirement": true,
	"product":     true,
}

var allowedIntegrationActions = map[string]bool{
	"upsert": true,
	"delete": true,
}

type integrationApplyResult struct {
	Status      string
	Reason      string
	WarehouseID string
}

// IngestTwentyEvent handles inbound integration events from Twenty.
// Phase 1 behavior:
//   - Validate envelope fields
//   - Deduplicate by idempotency key
//   - Persist raw receipt payload
//   - Upsert ID links when entity IDs are present
func IngestTwentyEvent(w http.ResponseWriter, r *http.Request) {
	var event integrationEventEnvelope
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&event); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := validateIntegrationEvent(event); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	db := repository.GetSQLDB()
	if db == nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "database unavailable"})
		return
	}

	payloadJSON, err := json.Marshal(event)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encode event payload"})
		return
	}

	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to open transaction"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var receiptID int64
	err = tx.QueryRowContext(
		r.Context(),
		`INSERT INTO integration_event_receipts (
			idempotency_key, event_id, source, entity_type, action,
			correlation_id, payload_json, received_at, processed_at, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, NOW(), NOW(), 'accepted')
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING receipt_id`,
		event.IdempotencyKey,
		event.EventID,
		event.Source,
		event.EntityType,
		event.Action,
		nullIfEmpty(event.CorrelationID),
		string(payloadJSON),
	).Scan(&receiptID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist event receipt"})
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		if commitErr := tx.Commit(); commitErr != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to finalize duplicate receipt"})
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"ok":             true,
			"status":         "duplicate_ignored",
			"idempotencyKey": event.IdempotencyKey,
			"eventId":        event.EventID,
			"entityType":     event.EntityType,
			"action":         event.Action,
		})
		return
	}

	twentyID := strings.TrimSpace(event.Entity.ExternalID)
	warehouseID := normalizeWarehouseID(event.Entity.WarehouseID)

	if twentyID != "" || warehouseID != "" {
		if upsertErr := upsertIntegrationLink(tx, event, warehouseID, twentyID); upsertErr != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to upsert integration link"})
			return
		}
	}

	applyResult, applyErr := applyIntegrationEvent(tx, event, warehouseID, twentyID)
	if applyErr != nil {
		log.Printf("[INTEGRATION] apply error for event %s: %v", event.EventID, applyErr)
		_ = updateIntegrationReceiptStatus(tx, receiptID, "failed_apply", applyErr.Error())
		if commitErr := tx.Commit(); commitErr != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to commit failed_apply receipt"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to apply integration event"})
		return
	}

	if applyResult.WarehouseID != "" {
		warehouseID = applyResult.WarehouseID
		if twentyID != "" || warehouseID != "" {
			if upsertErr := upsertIntegrationLink(tx, event, warehouseID, twentyID); upsertErr != nil {
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update integration link with applied id"})
				return
			}
		}
	}

	if updateErr := updateIntegrationReceiptStatus(tx, receiptID, applyResult.Status, applyResult.Reason); updateErr != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update receipt status"})
		return
	}

	if commitErr := tx.Commit(); commitErr != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to commit event receipt"})
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"ok":             true,
		"status":         applyResult.Status,
		"reason":         applyResult.Reason,
		"receiptId":      receiptID,
		"eventId":        event.EventID,
		"idempotencyKey": event.IdempotencyKey,
		"entityType":     event.EntityType,
		"action":         event.Action,
		"warehouseId":    warehouseID,
	})
}

func applyIntegrationEvent(tx *sql.Tx, event integrationEventEnvelope, warehouseID, twentyID string) (integrationApplyResult, error) {
	switch event.EntityType {
	case "customer":
		return applyCustomerEvent(tx, event, warehouseID, twentyID)
	case "job":
		return applyJobEvent(tx, event, warehouseID, twentyID)
	case "requirement":
		return applyRequirementEvent(tx, event, warehouseID, twentyID)
	case "product":
		return integrationApplyResult{Status: "accepted_noop", Reason: "product_owner_is_warehousecore", WarehouseID: warehouseID}, nil
	default:
		return integrationApplyResult{}, fmt.Errorf("unsupported entityType: %s", event.EntityType)
	}
}

func applyCustomerEvent(tx *sql.Tx, event integrationEventEnvelope, warehouseID, twentyID string) (integrationApplyResult, error) {
	cols, err := getTableColumns(tx, "customers")
	if err != nil {
		return integrationApplyResult{}, err
	}
	if len(cols) == 0 {
		return applyCustomerLinkFallback(event, warehouseID, twentyID), nil
	}
	if !cols["customerid"] {
		return applyCustomerLinkFallback(event, warehouseID, twentyID), nil
	}

	firstName, lastName := splitDisplayName(asString(event.Entity.Fields["name"]))
	if firstName == "" && lastName == "" {
		firstName = "Unknown"
	}

	if event.Action == "delete" {
		id, ok := parsePositiveInt64(warehouseID)
		if !ok {
			return integrationApplyResult{Status: "accepted_noop", Reason: "customer_delete_missing_warehouse_id", WarehouseID: warehouseID}, nil
		}
		_, delErr := tx.Exec(`DELETE FROM customers WHERE customerid = $1`, id)
		if delErr != nil {
			return integrationApplyResult{}, delErr
		}
		return integrationApplyResult{Status: "applied", Reason: "customer_deleted", WarehouseID: warehouseID}, nil
	}

	var resolvedID int64
	id, idOK := parsePositiveInt64(warehouseID)
	if idOK {
		updates := []string{}
		args := []interface{}{}
		argN := 1
		if cols["firstname"] {
			updates = append(updates, fmt.Sprintf("firstname = $%d", argN))
			args = append(args, firstName)
			argN++
		}
		if cols["lastname"] {
			updates = append(updates, fmt.Sprintf("lastname = $%d", argN))
			args = append(args, lastName)
			argN++
		}
		if cols["email"] && event.Entity.Fields != nil {
			if email := asString(event.Entity.Fields["email"]); email != "" {
				updates = append(updates, fmt.Sprintf("email = $%d", argN))
				args = append(args, email)
				argN++
			}
		}
		if cols["phone"] && event.Entity.Fields != nil {
			if phone := asString(event.Entity.Fields["phone"]); phone != "" {
				updates = append(updates, fmt.Sprintf("phone = $%d", argN))
				args = append(args, phone)
				argN++
			}
		}
		if len(updates) > 0 {
			args = append(args, id)
			result, updErr := tx.Exec(
				fmt.Sprintf("UPDATE customers SET %s WHERE customerid = $%d", strings.Join(updates, ", "), argN),
				args...,
			)
			if updErr != nil {
				return integrationApplyResult{}, updErr
			}
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				resolvedID = id
			}
		}
	}

	if resolvedID == 0 {
		insertCols := []string{}
		insertVals := []string{}
		args := []interface{}{}
		if idOK {
			insertCols = append(insertCols, "customerid")
			insertVals = append(insertVals, "$1")
			args = append(args, id)
		}
		if cols["firstname"] {
			insertCols = append(insertCols, "firstname")
			insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, firstName)
		}
		if cols["lastname"] {
			insertCols = append(insertCols, "lastname")
			insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, lastName)
		}
		if cols["email"] {
			if email := asString(event.Entity.Fields["email"]); email != "" {
				insertCols = append(insertCols, "email")
				insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, email)
			}
		}
		if cols["phone"] {
			if phone := asString(event.Entity.Fields["phone"]); phone != "" {
				insertCols = append(insertCols, "phone")
				insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, phone)
			}
		}

		query := fmt.Sprintf(
			"INSERT INTO customers (%s) VALUES (%s) RETURNING customerid",
			strings.Join(insertCols, ", "),
			strings.Join(insertVals, ", "),
		)
		if insErr := tx.QueryRow(query, args...).Scan(&resolvedID); insErr != nil {
			return integrationApplyResult{}, insErr
		}
	}

	return integrationApplyResult{Status: "applied", Reason: "customer_upserted", WarehouseID: strconv.FormatInt(resolvedID, 10)}, nil
}

func applyCustomerLinkFallback(event integrationEventEnvelope, warehouseID, twentyID string) integrationApplyResult {
	resolved := strings.TrimSpace(warehouseID)
	if resolved == "" {
		resolved = strings.TrimSpace(twentyID)
	}
	if event.Action == "delete" {
		return integrationApplyResult{Status: "applied", Reason: "customer_deleted", WarehouseID: resolved}
	}
	return integrationApplyResult{Status: "applied", Reason: "customer_upserted", WarehouseID: resolved}
}

func applyJobEvent(tx *sql.Tx, event integrationEventEnvelope, warehouseID, _ string) (integrationApplyResult, error) {
	cols, err := getTableColumns(tx, "jobs")
	if err != nil {
		return integrationApplyResult{}, err
	}
	if len(cols) == 0 {
		return integrationApplyResult{Status: "accepted_noop", Reason: "jobs_table_missing", WarehouseID: warehouseID}, nil
	}

	pk := ""
	if cols["jobid"] {
		pk = "jobid"
	} else if cols["id"] {
		pk = "id"
	} else {
		return integrationApplyResult{Status: "accepted_noop", Reason: "jobs_primary_key_missing", WarehouseID: warehouseID}, nil
	}

	if event.Action == "delete" {
		id, ok := parsePositiveInt64(warehouseID)
		if !ok {
			return integrationApplyResult{Status: "accepted_noop", Reason: "job_delete_missing_warehouse_id", WarehouseID: warehouseID}, nil
		}
		_, delErr := tx.Exec(fmt.Sprintf("DELETE FROM jobs WHERE %s = $1", pk), id)
		if delErr != nil {
			return integrationApplyResult{}, delErr
		}
		return integrationApplyResult{Status: "applied", Reason: "job_deleted", WarehouseID: warehouseID}, nil
	}

	jobCode := asString(event.Entity.Fields["jobCode"])
	if jobCode == "" {
		jobCode = asString(event.Entity.Fields["job_code"])
	}
	if jobCode == "" {
		jobCode = asString(event.Entity.Fields["name"])
	}
	if jobCode == "" {
		jobCode = "JOB-UNSET"
	}
	status := asString(event.Entity.Fields["status"])
	if status == "" {
		status = "open"
	}

	customerID := int64(0)
	if cols["customerid"] {
		if val, ok := parsePositiveInt64(normalizeWarehouseID(event.Entity.Fields["customerWarehouseId"])); ok {
			customerID = val
		} else if jobCustomerExternal := asString(event.Entity.Fields["customerExternalId"]); jobCustomerExternal != "" {
			if linkedCustomerID, linkErr := resolveWarehouseIDFromLink(tx, "customer", jobCustomerExternal); linkErr == nil {
				customerID = linkedCustomerID
			}
		}
	}

	resolvedID := int64(0)
	id, idOK := parsePositiveInt64(warehouseID)
	if idOK {
		updates := []string{}
		args := []interface{}{}
		argN := 1
		if cols["job_code"] {
			updates = append(updates, fmt.Sprintf("job_code = $%d", argN))
			args = append(args, jobCode)
			argN++
		}
		if cols["status"] {
			updates = append(updates, fmt.Sprintf("status = $%d", argN))
			args = append(args, status)
			argN++
		}
		if cols["description"] {
			if description := asString(event.Entity.Fields["description"]); description != "" {
				updates = append(updates, fmt.Sprintf("description = $%d", argN))
				args = append(args, description)
				argN++
			}
		}
		if cols["startdate"] {
			if startDate := asString(event.Entity.Fields["startDate"]); startDate != "" {
				updates = append(updates, fmt.Sprintf("startdate = $%d", argN))
				args = append(args, startDate)
				argN++
			}
		}
		if cols["enddate"] {
			if endDate := asString(event.Entity.Fields["endDate"]); endDate != "" {
				updates = append(updates, fmt.Sprintf("enddate = $%d", argN))
				args = append(args, endDate)
				argN++
			}
		}
		if cols["customerid"] && customerID > 0 {
			updates = append(updates, fmt.Sprintf("customerid = $%d", argN))
			args = append(args, customerID)
			argN++
		}

		if len(updates) > 0 {
			args = append(args, id)
			result, updErr := tx.Exec(
				fmt.Sprintf("UPDATE jobs SET %s WHERE %s = $%d", strings.Join(updates, ", "), pk, argN),
				args...,
			)
			if updErr != nil {
				return integrationApplyResult{}, updErr
			}
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				resolvedID = id
			}
		}
	}

	if resolvedID == 0 {
		insertCols := []string{}
		insertVals := []string{}
		args := []interface{}{}

		if idOK {
			insertCols = append(insertCols, pk)
			insertVals = append(insertVals, "$1")
			args = append(args, id)
		}
		if cols["job_code"] {
			insertCols = append(insertCols, "job_code")
			insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, jobCode)
		}
		if cols["status"] {
			insertCols = append(insertCols, "status")
			insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		if cols["description"] {
			if description := asString(event.Entity.Fields["description"]); description != "" {
				insertCols = append(insertCols, "description")
				insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, description)
			}
		}
		if cols["startdate"] {
			if startDate := asString(event.Entity.Fields["startDate"]); startDate != "" {
				insertCols = append(insertCols, "startdate")
				insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, startDate)
			}
		}
		if cols["enddate"] {
			if endDate := asString(event.Entity.Fields["endDate"]); endDate != "" {
				insertCols = append(insertCols, "enddate")
				insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, endDate)
			}
		}
		if cols["customerid"] && customerID > 0 {
			insertCols = append(insertCols, "customerid")
			insertVals = append(insertVals, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, customerID)
		}

		if len(insertCols) == 0 {
			return integrationApplyResult{Status: "accepted_noop", Reason: "jobs_insert_columns_missing", WarehouseID: warehouseID}, nil
		}

		query := fmt.Sprintf(
			"INSERT INTO jobs (%s) VALUES (%s) RETURNING %s",
			strings.Join(insertCols, ", "),
			strings.Join(insertVals, ", "),
			pk,
		)
		if insErr := tx.QueryRow(query, args...).Scan(&resolvedID); insErr != nil {
			return integrationApplyResult{}, insErr
		}
	}

	return integrationApplyResult{Status: "applied", Reason: "job_upserted", WarehouseID: strconv.FormatInt(resolvedID, 10)}, nil
}

func applyRequirementEvent(tx *sql.Tx, event integrationEventEnvelope, warehouseID, _ string) (integrationApplyResult, error) {
	cols, err := getTableColumns(tx, "job_product_requirements")
	if err != nil {
		return integrationApplyResult{}, err
	}

	jobWarehouseID := normalizeWarehouseID(event.Entity.Fields["jobWarehouseId"])
	if jobWarehouseID == "" {
		if parsed, ok := parsePositiveInt64(warehouseID); ok {
			jobWarehouseID = strconv.FormatInt(parsed, 10)
		}
	}
	if jobWarehouseID == "" {
		if jobExternalID := asString(event.Entity.Fields["jobExternalId"]); jobExternalID != "" {
			if linkedJobID, linkErr := resolveWarehouseIDFromLink(tx, "job", jobExternalID); linkErr == nil && linkedJobID > 0 {
				jobWarehouseID = strconv.FormatInt(linkedJobID, 10)
			}
		}
	}
	jobID, ok := parsePositiveInt64(jobWarehouseID)
	if !ok {
		return integrationApplyResult{Status: "accepted_noop", Reason: "requirement_job_unresolved", WarehouseID: warehouseID}, nil
	}

	productID, productOK := parsePositiveInt64(normalizeWarehouseID(event.Entity.Fields["warehouseProductId"]))
	if !productOK {
		return integrationApplyResult{Status: "accepted_noop", Reason: "requirement_product_unresolved", WarehouseID: warehouseID}, nil
	}

	if len(cols) == 0 {
		return applyRequirementViaJobDevicesFallback(tx, event.Action, jobID, productID)
	}

	jobCol := ""
	if cols["job_id"] {
		jobCol = "job_id"
	} else if cols["jobid"] {
		jobCol = "jobid"
	} else {
		return applyRequirementViaJobDevicesFallback(tx, event.Action, jobID, productID)
	}

	productCol := ""
	if cols["product_id"] {
		productCol = "product_id"
	} else if cols["productid"] {
		productCol = "productid"
	} else {
		return applyRequirementViaJobDevicesFallback(tx, event.Action, jobID, productID)
	}

	quantityCol := ""
	if cols["quantity"] {
		quantityCol = "quantity"
	}
	if quantityCol == "" {
		return applyRequirementViaJobDevicesFallback(tx, event.Action, jobID, productID)
	}

	if event.Action == "delete" {
		_, delErr := tx.Exec(
			fmt.Sprintf("DELETE FROM job_product_requirements WHERE %s = $1 AND %s = $2", jobCol, productCol),
			jobID,
			productID,
		)
		if delErr != nil {
			return integrationApplyResult{}, delErr
		}
		return integrationApplyResult{Status: "applied", Reason: "requirement_deleted", WarehouseID: strconv.FormatInt(jobID, 10)}, nil
	}

	quantity := int64(1)
	if parsed, qOK := parsePositiveInt64(normalizeWarehouseID(event.Entity.Fields["quantity"])); qOK {
		quantity = parsed
	}

	updateQuery := fmt.Sprintf(
		"UPDATE job_product_requirements SET %s = $3 WHERE %s = $1 AND %s = $2",
		quantityCol,
		jobCol,
		productCol,
	)
	result, updErr := tx.Exec(updateQuery, jobID, productID, quantity)
	if updErr != nil {
		return integrationApplyResult{}, updErr
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		insertQuery := fmt.Sprintf(
			"INSERT INTO job_product_requirements (%s, %s, %s) VALUES ($1, $2, $3)",
			jobCol,
			productCol,
			quantityCol,
		)
		if _, insErr := tx.Exec(insertQuery, jobID, productID, quantity); insErr != nil {
			return integrationApplyResult{}, insErr
		}
	}

	return integrationApplyResult{Status: "applied", Reason: "requirement_upserted", WarehouseID: strconv.FormatInt(jobID, 10)}, nil
}

func applyRequirementViaJobDevicesFallback(tx *sql.Tx, action string, jobID, productID int64) (integrationApplyResult, error) {
	jobDeviceCols, err := getTableColumns(tx, "jobdevices")
	if err != nil {
		return integrationApplyResult{}, err
	}
	if len(jobDeviceCols) == 0 || !jobDeviceCols["jobid"] || !jobDeviceCols["deviceid"] {
		return integrationApplyResult{Status: "accepted_noop", Reason: "job_product_requirements_table_missing", WarehouseID: strconv.FormatInt(jobID, 10)}, nil
	}

	deviceCols, err := getTableColumns(tx, "devices")
	if err != nil {
		return integrationApplyResult{}, err
	}
	if len(deviceCols) == 0 || !deviceCols["deviceid"] || !deviceCols["productid"] {
		return integrationApplyResult{Status: "accepted_noop", Reason: "devices_product_mapping_missing", WarehouseID: strconv.FormatInt(jobID, 10)}, nil
	}

	if action == "delete" {
		_, delErr := tx.Exec(
			`DELETE FROM jobdevices
			 WHERE jobid = $1
			   AND deviceid IN (
			     SELECT d.deviceid FROM devices d WHERE d.productid = $2
			   )`,
			jobID,
			productID,
		)
		if delErr != nil {
			return integrationApplyResult{}, delErr
		}
		return integrationApplyResult{Status: "applied", Reason: "requirement_deleted", WarehouseID: strconv.FormatInt(jobID, 10)}, nil
	}

	var existingCount int64
	if countErr := tx.QueryRow(
		`SELECT COUNT(*)
		 FROM jobdevices jd
		 JOIN devices d ON d.deviceid = jd.deviceid
		 WHERE jd.jobid = $1 AND d.productid = $2`,
		jobID,
		productID,
	).Scan(&existingCount); countErr != nil {
		return integrationApplyResult{}, countErr
	}

	quantity := int64(1)
	if quantity < existingCount {
		quantity = existingCount
	}

	if existingCount == 0 {
		_, insErr := tx.Exec(
			`INSERT INTO jobdevices (deviceid, jobid, pack_status, pack_ts)
			 SELECT d.deviceid, $1, 'pending', NOW()
			 FROM devices d
			 LEFT JOIN jobdevices jd ON jd.deviceid = d.deviceid
			 WHERE d.productid = $2
			   AND jd.deviceid IS NULL
			 LIMIT 1`,
			jobID,
			productID,
		)
		if insErr != nil {
			return integrationApplyResult{}, insErr
		}
	}

	return integrationApplyResult{Status: "applied", Reason: "requirement_upserted", WarehouseID: strconv.FormatInt(jobID, 10)}, nil
}

func updateIntegrationReceiptStatus(tx *sql.Tx, receiptID int64, status, reason string) error {
	_, err := tx.Exec(
		`UPDATE integration_event_receipts
		 SET status = $1,
		     status_reason = NULLIF($2, ''),
		     processed_at = NOW()
		 WHERE receipt_id = $3`,
		status,
		reason,
		receiptID,
	)
	return err
}

func getTableColumns(tx *sql.Tx, tableName string) (map[string]bool, error) {
	rows, err := tx.Query(
		`SELECT LOWER(column_name)
		 FROM information_schema.columns
		 WHERE table_schema = current_schema() AND table_name = $1`,
		strings.ToLower(tableName),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := map[string]bool{}
	for rows.Next() {
		var col string
		if scanErr := rows.Scan(&col); scanErr != nil {
			return nil, scanErr
		}
		cols[col] = true
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	return cols, nil
}

func resolveWarehouseIDFromLink(tx *sql.Tx, entityType, twentyID string) (int64, error) {
	var raw sql.NullString
	err := tx.QueryRow(
		`SELECT warehouse_id
		 FROM integration_links
		 WHERE system = 'twenty' AND entity_type = $1 AND twenty_id = $2
		 LIMIT 1`,
		entityType,
		twentyID,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	if !raw.Valid {
		return 0, nil
	}
	parsed, ok := parsePositiveInt64(strings.TrimSpace(raw.String))
	if !ok {
		return 0, nil
	}
	return parsed, nil
}

func parsePositiveInt64(raw string) (int64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func asString(raw interface{}) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func splitDisplayName(name string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func validateIntegrationEvent(event integrationEventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" {
		return fmt.Errorf("eventId is required")
	}
	if strings.TrimSpace(event.IdempotencyKey) == "" {
		return fmt.Errorf("idempotencyKey is required")
	}
	if !allowedIntegrationSources[strings.TrimSpace(event.Source)] {
		return fmt.Errorf("invalid source")
	}
	if !allowedIntegrationEntityTypes[strings.TrimSpace(event.EntityType)] {
		return fmt.Errorf("invalid entityType")
	}
	if !allowedIntegrationActions[strings.TrimSpace(event.Action)] {
		return fmt.Errorf("invalid action")
	}
	if strings.TrimSpace(event.OccurredAt) != "" {
		if _, err := time.Parse(time.RFC3339, strings.TrimSpace(event.OccurredAt)); err != nil {
			return fmt.Errorf("occurredAt must be RFC3339")
		}
	}
	return nil
}

func normalizeWarehouseID(raw interface{}) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(v, 'f', -1, 64))
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case json.Number:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func upsertIntegrationLink(tx *sql.Tx, event integrationEventEnvelope, warehouseID, twentyID string) error {
	// Prefer twenty_id as stable key when available.
	if twentyID != "" {
		_, err := tx.Exec(
			`INSERT INTO integration_links (
				system, entity_type, warehouse_id, twenty_id,
				last_source, last_event_id, last_synced_at, created_at, updated_at
			) VALUES ('twenty', $1, NULLIF($2, ''), $3, $4, $5, NOW(), NOW(), NOW())
			ON CONFLICT (system, entity_type, twenty_id)
			WHERE twenty_id IS NOT NULL
			DO UPDATE SET
				warehouse_id = COALESCE(NULLIF(EXCLUDED.warehouse_id, ''), integration_links.warehouse_id),
				last_source = EXCLUDED.last_source,
				last_event_id = EXCLUDED.last_event_id,
				last_synced_at = NOW(),
				updated_at = NOW()`,
			event.EntityType,
			warehouseID,
			twentyID,
			event.Source,
			event.EventID,
		)
		return err
	}

	// Fallback path: only warehouse side ID present.
	_, err := tx.Exec(
		`INSERT INTO integration_links (
			system, entity_type, warehouse_id, twenty_id,
			last_source, last_event_id, last_synced_at, created_at, updated_at
		) VALUES ('twenty', $1, $2, NULL, $3, $4, NOW(), NOW(), NOW())
		ON CONFLICT (system, entity_type, warehouse_id)
		WHERE warehouse_id IS NOT NULL
		DO UPDATE SET
			last_source = EXCLUDED.last_source,
			last_event_id = EXCLUDED.last_event_id,
			last_synced_at = NOW(),
			updated_at = NOW()`,
		event.EntityType,
		warehouseID,
		event.Source,
		event.EventID,
	)
	return err
}

func nullIfEmpty(v string) interface{} {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
