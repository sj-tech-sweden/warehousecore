package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"

	"warehousecore/internal/handlers"
)

func TestGetZoneDevices_Success(t *testing.T) {
	mock := withMockDB(t)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/zones/{id}/devices", handlers.GetZoneDevices).Methods("GET")

	// columns as scanned in handler
	cols := []string{"deviceid", "productid", "serialnumber", "status", "barcode", "qr_code", "condition_rating", "usage_hours", "product_name", "manufacturer", "model", "zone_code"}
	rows := sqlmock.NewRows(cols).
		AddRow("DEV1", nil, "SN1", "in_storage", "B1", "Q1", 0.0, 0.0, "Prod", "Mfg", "ModelX", "ZC")
	mock.ExpectQuery(`SELECT d.deviceID`).WithArgs(int64(10)).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/zones/10/devices", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	var out []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 device, got %d", len(out))
	}
}

func TestAssignDevicesToZone_PartialFailure(t *testing.T) {
	mock := withMockDB(t)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/zones/{id}/devices", handlers.AssignDevicesToZone).Methods("POST")

	// zone exists
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// first device update succeeds, second fails (rowsAffected 0)
	mock.ExpectExec(`UPDATE devices`).WithArgs(int64(20), "DEV1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO device_movements`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE devices`).WithArgs(int64(20), "DEV2").WillReturnResult(sqlmock.NewResult(0, 0))

	body := `{"device_ids":["DEV1","DEV2"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/zones/20/devices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["success"].(float64) != 1 {
		t.Fatalf("expected success 1, got %v", resp["success"])
	}
}

func TestAssignDevicesToZone_ZoneNotFound(t *testing.T) {
	mock := withMockDB(t)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/zones/{id}/devices", handlers.AssignDevicesToZone).Methods("POST")

	// zone does not exist
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(999)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	body := `{"device_ids":["DEVX"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/zones/999/devices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}
}
