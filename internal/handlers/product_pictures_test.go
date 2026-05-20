package handlers_test

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"warehousecore/internal/handlers"
	"warehousecore/internal/services"
)

// fake picture service implements the ProductPictureServiceInterface for tests
type fakePictureService struct {
	enabled bool
}

func (f *fakePictureService) Enabled() bool                              { return f.enabled }
func (f *fakePictureService) MaxFileSize() int64                         { return 1024 }
func (f *fakePictureService) FolderForProduct(productName string) string { return "test/folder" }
func (f *fakePictureService) ListPictures(productName string) ([]services.ProductPictureInfo, error) {
	return []services.ProductPictureInfo{}, nil
}
func (f *fakePictureService) UploadPicture(productName string, file multipart.File, header *multipart.FileHeader) (string, error) {
	return "stored.png", nil
}
func (f *fakePictureService) DeletePicture(productName, fileName string) error { return nil }
func (f *fakePictureService) DownloadPictureWithVariant(productName, fileName, variant, format string) (io.ReadCloser, string, error) {
	return nil, "image/png", nil
}
func (f *fakePictureService) ClearCachedVariants(productName, fileName string) {}
func (f *fakePictureService) WarmPictureVariants(productName, fileName string) {}

func TestGetProductPictures_Empty(t *testing.T) {
	mock := withMockDB(t)
	// product name lookup
	mock.ExpectQuery(`SELECT name FROM products`).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("TestProduct"))

	// inject fake service
	t.Cleanup(handlers.SetProductPictureService(&fakePictureService{enabled: true}))

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/products/{id}/pictures", handlers.GetProductPictures).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/products/1/pictures", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["pictures"]; !ok {
		t.Fatalf("expected pictures key in response")
	}
}
