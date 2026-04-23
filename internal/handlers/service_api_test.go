package handlers_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"

	"warehousecore/internal/handlers"
	"warehousecore/internal/middleware"
	"warehousecore/internal/repository"
)

// withErrorDB injects a sqlmock DB that returns connection errors, simulating
// a database that is unavailable. It returns the mock so callers can register
// exactly as many expectations as they need and optionally verify them.
// A t.Cleanup is registered that restores the original DB (via the repository
// mutex helper), verifies all expectations were met, and closes the mock DB.
func withErrorDB(t *testing.T) sqlmock.Sqlmock {
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
	return mock
}

// serviceRouter builds a minimal *mux.Router that mirrors the service subrouter
// registered in main.go, enabling both http.Handler use and direct router.Match
// calls in tests.
func serviceRouter() *mux.Router {
	router := mux.NewRouter()
	service := router.PathPrefix("/api/v1/service").Subrouter()
	service.Use(middleware.APIKeyMiddleware)
	service.HandleFunc("/devices/{id}", handlers.GetDevice).Methods("GET")
	return router
}

// TestServiceAPI_MissingAPIKey verifies that all service endpoints return 401
// when no API key is supplied.
func TestServiceAPI_MissingAPIKey(t *testing.T) {
	router := serviceRouter()

	paths := []string{
		"/api/v1/service/devices/DEV001",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("path %s: expected 401 without API key, got %d", path, rr.Code)
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("path %s: expected Content-Type application/json, got %q", path, ct)
			}
		})
	}
}

// TestServiceAPI_APIKey_DBUnavailable_Returns500 verifies that when a key is
// present but the database is unavailable, the middleware returns 500 (not a
// misleading 401 "invalid key"). One sqlmock expectation is registered per
// request so that ExpectationsWereMet() confirms the middleware actually hit
// the DB on every call.
func TestServiceAPI_APIKey_DBUnavailable_Returns500(t *testing.T) {
	mock := withErrorDB(t)
	router := serviceRouter()

	paths := []string{
		"/api/v1/service/devices/DEV001",
	}

	for _, path := range paths {
		// Register one expectation before each request so sqlmock can verify
		// the middleware is actually querying the DB on every call.
		mock.ExpectQuery(`SELECT id FROM api_keys`).WillReturnError(errors.New("connection refused"))

		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("X-API-Key", "some-key")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("path %s: expected 500 when DB unavailable with API key, got %d", path, rr.Code)
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("path %s: expected Content-Type application/json, got %q", path, ct)
			}
		})
	}
}

// TestServiceAPI_Routes_NotFoundWithoutAuth checks that an unknown path under
// the service prefix is not registered as a route in the router. Uses
// router.Match directly to avoid relying on status codes, which can be
// influenced by the subrouter middleware even for unregistered paths.
func TestServiceAPI_Routes_NotFoundWithoutAuth(t *testing.T) {
	router := serviceRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/service/unknown-endpoint", nil)
	var routeMatch mux.RouteMatch
	if router.Match(req, &routeMatch) {
		t.Error("unknown service path unexpectedly matched a registered route")
	}
}

// TestGetDevice_ResponseHasNoCableID verifies that the GetDevice handler no
// longer includes a cable_id field in its response. Cables were removed as a
// separate entity and migrated into products with custom field values.
func TestGetDevice_ResponseHasNoCableID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	// Inject the mock DB into the repository using the mutex-protected helper
	// to avoid cross-package data races when packages run in parallel.
	// restore() is called first so the global handle is valid again before
	// the sqlmock connection is closed.
	restore := repository.WithTestSQLDB(db)
	t.Cleanup(func() {
		restore()
		db.Close()
	})

	// The GetDevice query selects 29 columns (cable_id was removed in the
	// product custom fields migration).
	rows := sqlmock.NewRows([]string{
		"deviceID", "productID",
		"product_name", "product_description", "product_category", "subcategory",
		"manufacturer_name", "brand_name",
		"product_weight", "product_width", "product_height", "product_depth",
		"maintenance_interval", "power_consumption",
		"serialnumber", "rfid", "barcode", "qr_code",
		"status", "zone_id", "condition_rating", "usage_hours", "label_path",
		"purchase_date", "notes",
		"zone_name", "zone_code", "case_name", "job_number",
	}).AddRow(
		"DEV001", sql.NullInt64{Int64: 1, Valid: true},
		"Test Product", "A test device", "Audio", "",
		"Shure", "Shure",
		float64(0), float64(0), float64(0), float64(0),
		0, float64(0),
		sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{},
		"in_storage", sql.NullInt64{}, float64(4.5), float64(10.0), sql.NullString{},
		sql.NullString{}, sql.NullString{},
		"Shelf A", "WDL-01", "", "",
	)

	mock.ExpectQuery(`SELECT d\.deviceID`).WillReturnRows(rows)

	// Build a router that routes to GetDevice without any auth middleware.
	router := mux.NewRouter()
	router.HandleFunc("/devices/{id}", handlers.GetDevice).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/devices/DEV001", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := body["cable_id"]; ok {
		t.Error("cable_id field should not be present in GetDevice response after cable migration")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}
