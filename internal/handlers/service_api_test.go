package handlers_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"

	"warehousecore/internal/handlers"
	"warehousecore/internal/middleware"
	"warehousecore/internal/repository"
)

func withServiceAPIKey(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv("SERVICE_API_KEY")
	if key == "" {
		_ = os.Unsetenv("SERVICE_API_KEY")
	} else {
		_ = os.Setenv("SERVICE_API_KEY", key)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("SERVICE_API_KEY", prev)
			return
		}
		_ = os.Unsetenv("SERVICE_API_KEY")
	})
}

// serviceRouter builds a minimal *mux.Router that mirrors the service subrouter
// registered in main.go, enabling both http.Handler use and direct router.Match
// calls in tests.
func serviceRouter() *mux.Router {
	router := mux.NewRouter()
	service := router.PathPrefix("/api/v1/service").Subrouter()
	service.Use(middleware.ServiceKeyMiddleware)
	service.HandleFunc("/devices/{id}", handlers.GetDevice).Methods("GET")
	service.HandleFunc("/products", handlers.GetProducts).Methods("GET")
	service.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
	return router
}

// TestServiceAPI_MissingAPIKey verifies that all service endpoints return 401
// when no API key is supplied.
func TestServiceAPI_MissingAPIKey(t *testing.T) {
	withServiceAPIKey(t, "service-key")
	router := serviceRouter()

	paths := []string{
		"/api/v1/service/devices/DEV001",
		"/api/v1/service/products",
		"/api/v1/service/products/42",
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

func TestServiceAPI_InvalidAPIKey_Returns401(t *testing.T) {
	withServiceAPIKey(t, "service-key")
	router := serviceRouter()

	paths := []string{
		"/api/v1/service/devices/DEV001",
		"/api/v1/service/products",
		"/api/v1/service/products/42",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("X-API-Key", "wrong-key")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("path %s: expected 401 for invalid API key, got %d", path, rr.Code)
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("path %s: expected Content-Type application/json, got %q", path, ct)
			}
		})
	}
}

func TestServiceAPI_MissingServiceKeyConfig_Returns503(t *testing.T) {
	withServiceAPIKey(t, "")
	router := serviceRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/service/products", nil)
	req.Header.Set("X-API-Key", "any-key")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when SERVICE_API_KEY is unset, got %d", rr.Code)
	}
}

func TestServiceAPI_MissingServiceKeyConfig_WithoutHeader_Returns503(t *testing.T) {
	withServiceAPIKey(t, "")
	router := serviceRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/service/products", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when SERVICE_API_KEY is unset even without header, got %d", rr.Code)
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
