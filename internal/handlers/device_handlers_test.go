package handlers_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"

	"warehousecore/internal/handlers"
)

// deviceRouter builds minimal device routes used in tests
func deviceRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/devices/{id}", handlers.GetDevice).Methods("GET")
	r.HandleFunc("/api/v1/devices/{id}/movements", handlers.GetDeviceMovements).Methods("GET")
	return r
}

// Reuses shared withMockDB from other test files.

func TestGetDevice_NotFound(t *testing.T) {
	mock := withMockDB(t)
	router := deviceRouter()

	mock.ExpectQuery(`SELECT`).WithArgs("NOPE").WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/NOPE", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestGetDevice_Success(t *testing.T) {
	mock := withMockDB(t)
	router := deviceRouter()

	cols := []string{"deviceid", "productid", "name", "description", "product_category", "subcategory", "manufacturer_name", "brand_name", "weight", "width", "height", "depth", "maintenance_interval", "power_consumption", "serialnumber", "rfid", "barcode", "qr_code", "status", "zone_id", "condition_rating", "usage_hours", "label_path", "purchase_date", "notes", "zone_name", "zone_code", "case_name", "job_number"}
	row := sqlmock.NewRows(cols).AddRow("DEV1", nil, "PName", "PDesc", "Cat", "Sub", "Mfg", "Brand", 0, 0, 0, 0, 0, 0, nil, nil, nil, nil, "in_storage", nil, 0, 0, nil, "", "", "Z1", "ZCODE", "CaseA", "")

	mock.ExpectQuery(`SELECT`).WithArgs("DEV1").WillReturnRows(row)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/DEV1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["device_id"] != "DEV1" {
		t.Fatalf("unexpected device_id: %v", resp["device_id"])
	}
}
