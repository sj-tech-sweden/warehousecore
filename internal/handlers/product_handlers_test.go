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

// productRouter builds minimal routes used in tests
func productRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/products", handlers.GetProducts).Methods("GET")
	r.HandleFunc("/api/v1/products/{id}", handlers.GetProduct).Methods("GET")
	return r
}

// Reuses shared withMockDB from other test files.

func TestGetProducts_EmptyList(t *testing.T) {
	mock := withMockDB(t)
	router := productRouter()

	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"productid"}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	var out []interface{}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty list, got %d", len(out))
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	mock := withMockDB(t)
	router := productRouter()

	mock.ExpectQuery(`SELECT`).WithArgs(999).WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/999", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}
}
