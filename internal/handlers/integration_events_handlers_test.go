package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"warehousecore/internal/repository"
)

func withIntegrationMockSQLDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	restore := repository.WithTestSQLDB(db)
	t.Cleanup(func() {
		restore()
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		db.Close()
	})

	return db, mock
}

func TestIngestTwentyEvent_DuplicateIgnored(t *testing.T) {
	_, mock := withIntegrationMockSQLDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO integration_event_receipts`).WillReturnRows(sqlmock.NewRows([]string{"receipt_id"}))
	mock.ExpectCommit()

	payload := []byte(`{
    "eventId":"evt-dup-1",
    "schemaVersion":1,
    "source":"twenty",
    "entityType":"product",
    "action":"upsert",
    "occurredAt":"2026-01-01T00:00:00Z",
    "correlationId":"corr-dup-1",
    "idempotencyKey":"idem-dup-1",
    "entity":{
      "externalId":"prod-dup-1",
      "warehouseId":null,
      "version":1,
      "fields":{"name":"dup"}
    }
  }`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/twenty/events", bytes.NewReader(payload))
	rr := httptest.NewRecorder()

	IngestTwentyEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for duplicate event, got %d; body=%s", rr.Code, rr.Body.String())
	}

	var res map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if got := res["status"]; got != "duplicate_ignored" {
		t.Fatalf("expected duplicate_ignored status, got %v", got)
	}
}

func TestUpsertIntegrationLink_UsesTwentyIDPartialConflictPredicate(t *testing.T) {
	db, mock := withIntegrationMockSQLDB(t)

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	mock.ExpectExec(`(?s)INSERT INTO integration_links .*ON CONFLICT \(system, entity_type, twenty_id\)\s+WHERE twenty_id IS NOT NULL\s+DO UPDATE SET`).
		WithArgs("customer", "123", "twenty-customer-1", "twenty", "evt-customer-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = upsertIntegrationLink(tx, integrationEventEnvelope{
		EventID:    "evt-customer-1",
		Source:     "twenty",
		EntityType: "customer",
	}, "123", "twenty-customer-1")
	if err != nil {
		t.Fatalf("unexpected upsert error: %v", err)
	}

	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}
}

func TestUpsertIntegrationLink_UsesWarehouseIDPartialConflictPredicate(t *testing.T) {
	db, mock := withIntegrationMockSQLDB(t)

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	mock.ExpectExec(`(?s)INSERT INTO integration_links .*ON CONFLICT \(system, entity_type, warehouse_id\)\s+WHERE warehouse_id IS NOT NULL\s+DO UPDATE SET`).
		WithArgs("job", "55", "twenty", "evt-job-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = upsertIntegrationLink(tx, integrationEventEnvelope{
		EventID:    "evt-job-1",
		Source:     "twenty",
		EntityType: "job",
	}, "55", "")
	if err != nil {
		t.Fatalf("unexpected upsert error: %v", err)
	}

	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}
}
