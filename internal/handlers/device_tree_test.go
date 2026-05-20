package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"

	"warehousecore/internal/handlers"
)

func TestGetDeviceTree_SimpleRow(t *testing.T) {
	mock := withMockDB(t)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/devices/tree", handlers.GetDeviceTree).Methods("GET")

	cols := []string{"categoryid", "category_name", "subcategoryid", "subcategory_name", "subbiercategoryid", "subbiercategory_name", "productid", "product_name", "is_consumable", "is_accessory", "stock_quantity", "unit", "deviceid", "status", "barcode", "qr_code", "serialnumber", "zone_id", "zone_name", "zone_code", "case_id", "case_name", "current_job_id", "job_number", "condition_rating", "usage_hours", "label_path", "purchaseDate", "lastmaintenance", "nextmaintenance", "notes"}

	rows := sqlmock.NewRows(cols).AddRow(
		int64(1), "Cat", "sc1", "Sub", "sbc1", "SubB", int64(10), "Prod", 0, 0, 0.0, "", "DEV1", "in_storage", "", "", "", int64(2), "ZName", "ZCode", nil, "", nil, "", 0.0, 0.0, "", "", "", "", "",
	)

	mock.ExpectQuery(`WITH latest_job AS`).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tree", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["treeData"]; !ok {
		t.Fatalf("expected treeData in response")
	}
}
