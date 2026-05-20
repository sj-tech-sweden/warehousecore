package handlers_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"

	"warehousecore/internal/handlers"
)

// TestUploadProductPictures verifies multipart upload path and DB update.
func TestUploadProductPictures_Success(t *testing.T) {
	mock := withMockDB(t)

	// product name lookup
	mock.ExpectQuery(`SELECT name FROM products`).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("TestProduct"))
	// UPDATE products for website images
	mock.ExpectExec(`UPDATE products`).WillReturnResult(sqlmock.NewResult(1, 1))

	// inject fake service
	t.Cleanup(handlers.SetProductPictureService(&fakePictureService{enabled: true}))

	// build multipart request
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("files", "test.png")
	if err != nil {
		t.Fatalf("failed create form file: %v", err)
	}
	_, _ = io.Copy(fw, bytes.NewBufferString("PNGDATA"))
	w.Close()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/products/{id}/pictures", handlers.UploadProductPictures).Methods("POST")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products/1/pictures", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

// TestDeleteProductPicture_NotConfigured returns 503 when pictures disabled.
func TestDeleteProductPicture_NotConfigured(t *testing.T) {
	// inject disabled service
	t.Cleanup(handlers.SetProductPictureService(&fakePictureService{enabled: false}))

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/products/{id}/pictures/{filename}", handlers.DeleteProductPicture).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/products/1/pictures/foo.png", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when pictures disabled, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

// TestDeleteProductPicture_Success deletes when configured and product exists.
func TestDeleteProductPicture_Success(t *testing.T) {
	mock := withMockDB(t)
	mock.ExpectQuery(`SELECT name FROM products`).WithArgs(2).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("P2"))

	t.Cleanup(handlers.SetProductPictureService(&fakePictureService{enabled: true}))

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/products/{id}/pictures/{filename}", handlers.DeleteProductPicture).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/products/2/pictures/foo.png", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
}
