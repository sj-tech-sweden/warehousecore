package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"warehousecore/internal/models"
)

// stub implementation of the label service used to inject deterministic
// responses for handler tests.
type testLabelService struct{}

func (t *testLabelService) GenerateQRCode(content string, size int) (string, error) {
	return "data:image/png;base64,TESTQR", nil
}
func (t *testLabelService) GenerateBarcode(content string, w, h int) (string, error) {
	return "data:image/png;base64,TESTBC", nil
}
func (t *testLabelService) GetAllTemplates() ([]models.LabelTemplate, error) {
	// return nil to ensure handler encodes an empty array
	return nil, nil
}
func (t *testLabelService) GetTemplateByID(id int) (*models.LabelTemplate, error) {
	return &models.LabelTemplate{}, nil
}
func (t *testLabelService) CreateTemplate(tpl *models.LabelTemplate) error              { return nil }
func (t *testLabelService) UpdateTemplate(id int, updates map[string]interface{}) error { return nil }
func (t *testLabelService) DeleteTemplate(id int) error                                 { return nil }
func (t *testLabelService) GenerateLabelForDevice(deviceID string, templateID int) (map[string]interface{}, error) {
	return map[string]interface{}{"image_data": "data:image/png;base64,DEV"}, nil
}
func (t *testLabelService) GenerateLabelForCase(caseID, templateID int) (map[string]interface{}, error) {
	return map[string]interface{}{"image_data": "data:image/png;base64,CASE"}, nil
}
func (t *testLabelService) GenerateLabelForZone(zoneID int64, templateID int) (map[string]interface{}, error) {
	return map[string]interface{}{"image_data": "data:image/png;base64,ZONE"}, nil
}
func (t *testLabelService) SaveLabelImage(deviceID, imageData string) (string, error) {
	return "/labels/ok.png", nil
}
func (t *testLabelService) SaveCaseLabelImage(caseID int, imageData string) (string, error) {
	return "/labels/case.png", nil
}
func (t *testLabelService) SaveZoneLabelImage(zoneID int64, imageData string) (string, error) {
	return "/labels/zone.png", nil
}

func TestGenerateQRCode_Success(t *testing.T) {
	t.Cleanup(SetLabelService(&testLabelService{}))

	body := `{"content":"hello","size":128}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/labels/qrcode", strings.NewReader(body))
	rr := httptest.NewRecorder()
	GenerateQRCode(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp["image_data"] == "" {
		t.Fatal("expected image_data in response")
	}
}

func TestGenerateQRCode_BadRequest(t *testing.T) {
	t.Cleanup(SetLabelService(&testLabelService{}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/labels/qrcode", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	GenerateQRCode(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGenerateBarcode_BadRequest(t *testing.T) {
	t.Cleanup(SetLabelService(&testLabelService{}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/labels/barcode", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	GenerateBarcode(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGetLabelTemplates_EmptyArray(t *testing.T) {
	t.Cleanup(SetLabelService(&testLabelService{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/labels/templates", nil)
	rr := httptest.NewRecorder()
	GetLabelTemplates(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var arr []models.LabelTemplate
	if err := json.NewDecoder(rr.Body).Decode(&arr); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if arr == nil {
		t.Fatal("expected empty array, got nil")
	}
}

func TestGenerateDeviceLabel_MissingTemplateID(t *testing.T) {
	t.Cleanup(SetLabelService(&testLabelService{}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/labels/device/DEV1", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	// need to set mux vars for device_id
	req = muxSetVars(req, map[string]string{"device_id": "DEV1"})
	GenerateDeviceLabel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestSaveDeviceLabel_Success(t *testing.T) {
	t.Cleanup(SetLabelService(&testLabelService{}))
	body := `{"device_id":"DEV1","image_data":"data:image/png;base64,AAA"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/labels/save", strings.NewReader(body))
	rr := httptest.NewRecorder()
	SaveDeviceLabel(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

// helper to set mux.Vars on a request (since handlers use mux.Vars)
func muxSetVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}
