package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ===========================
// parseEventoryProductsResponse tests
// ===========================

func TestParseEventoryProductsResponse_DirectArray(t *testing.T) {
	body := `[
		{"id":1,"name":"Speaker","description":"PA Speaker","category":"Audio","price":29.99},
		{"id":2,"name":"Stand","description":"Mic stand","category":"Accessories","price":5.00}
	]`

	products, err := parseEventoryProductsResponse([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	if products[0].Name != "Speaker" {
		t.Errorf("expected first product name 'Speaker', got %q", products[0].Name)
	}
}

func TestParseEventoryProductsResponse_DataWrapper(t *testing.T) {
	body := `{"data":[{"id":1,"name":"Fogger","category":"Effects","price":50.0}]}`

	products, err := parseEventoryProductsResponse([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 || products[0].Name != "Fogger" {
		t.Errorf("unexpected result: %+v", products)
	}
}

func TestParseEventoryProductsResponse_ProductsWrapper(t *testing.T) {
	body := `{"products":[{"id":42,"name":"Truss","category":"Structure","price":100.0}]}`

	products, err := parseEventoryProductsResponse([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 || products[0].Name != "Truss" {
		t.Errorf("unexpected result: %+v", products)
	}
}

func TestParseEventoryProductsResponse_ItemsWrapper(t *testing.T) {
	body := `{"items":[{"id":7,"name":"Cable 10m","category":"Cables","price":3.5}]}`

	products, err := parseEventoryProductsResponse([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 || products[0].Name != "Cable 10m" {
		t.Errorf("unexpected result: %+v", products)
	}
}

func TestParseEventoryProductsResponse_ResultsWrapper(t *testing.T) {
	body := `{"results":[{"id":3,"name":"LED Bar","price":25.0}]}`

	products, err := parseEventoryProductsResponse([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 || products[0].Name != "LED Bar" {
		t.Errorf("unexpected result: %+v", products)
	}
}

func TestParseEventoryProductsResponse_EmptyArray(t *testing.T) {
	products, err := parseEventoryProductsResponse([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("expected 0 products, got %d", len(products))
	}
}

func TestParseEventoryProductsResponse_Unknown(t *testing.T) {
	_, err := parseEventoryProductsResponse([]byte(`{"unknown_key":"value"}`))
	if err == nil {
		t.Fatal("expected error for unrecognised shape, got nil")
	}
}

func TestParseEventoryProductsResponse_InvalidJSON(t *testing.T) {
	_, err := parseEventoryProductsResponse([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ===========================
// ValidateEventoryURL tests
// ===========================

func TestValidateEventoryURL_ValidHTTPS(t *testing.T) {
	if err := ValidateEventoryURL("https://api.eventory.se"); err != nil {
		t.Errorf("unexpected error for valid URL: %v", err)
	}
}

func TestValidateEventoryURL_ValidHTTP(t *testing.T) {
	if err := ValidateEventoryURL("http://example.com"); err != nil {
		t.Errorf("unexpected error for http URL: %v", err)
	}
}

func TestValidateEventoryURL_RejectsLocalhost(t *testing.T) {
	if err := ValidateEventoryURL("http://localhost:8080"); err == nil {
		t.Error("expected error for localhost URL, got nil")
	}
}

func TestValidateEventoryURL_RejectsPrivateIP(t *testing.T) {
	if err := ValidateEventoryURL("http://192.168.1.1/api"); err == nil {
		t.Error("expected error for private IP URL, got nil")
	}
}

func TestValidateEventoryURL_RejectsEmbeddedCredentials(t *testing.T) {
	if err := ValidateEventoryURL("http://user:pass@example.com"); err == nil {
		t.Error("expected error for URL with credentials, got nil")
	}
}

func TestValidateEventoryURL_RejectsNonHTTP(t *testing.T) {
	if err := ValidateEventoryURL("ftp://files.example.com"); err == nil {
		t.Error("expected error for ftp:// URL, got nil")
	}
}

// ===========================
// Endpoint fallback tests
// ===========================

func TestFetchEventoryProducts_EndpointFallback(t *testing.T) {
	callCounts := map[string]int{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCounts[r.URL.Path]++
		if r.URL.Path == "/api/products" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]EventoryProduct{{Name: "From fallback"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL}
	products, err := FetchEventoryProducts(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 || products[0].Name != "From fallback" {
		t.Errorf("unexpected products: %+v", products)
	}
	// /api/v1/products should have been tried first and returned 404
	if callCounts["/api/v1/products"] == 0 {
		t.Error("expected /api/v1/products to have been tried")
	}
	if callCounts["/api/products"] == 0 {
		t.Error("expected /api/products to have been tried")
	}
}

func TestFetchEventoryProducts_AllFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL}
	_, err := FetchEventoryProducts(cfg)
	if err == nil {
		t.Fatal("expected error when all endpoints fail, got nil")
	}
}

func TestFetchEventoryProducts_BearerAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EventoryProduct{{Name: "Secured"}})
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL, APIKey: "my-token"}
	_, err := FetchEventoryProducts(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer my-token" {
		t.Errorf("expected Authorization: Bearer my-token, got %q", gotAuth)
	}
}

// ===========================
// EffectiveSupplierName tests
// ===========================

func TestEffectiveSupplierName_Default(t *testing.T) {
	cfg := &EventoryConfig{}
	if got := cfg.EffectiveSupplierName(); got != "Eventory" {
		t.Errorf("expected 'Eventory', got %q", got)
	}
}

func TestEffectiveSupplierName_Custom(t *testing.T) {
	cfg := &EventoryConfig{SupplierName: "My Events GmbH"}
	if got := cfg.EffectiveSupplierName(); got != "My Events GmbH" {
		t.Errorf("expected 'My Events GmbH', got %q", got)
	}
}

func TestEffectiveSupplierName_Whitespace(t *testing.T) {
	cfg := &EventoryConfig{SupplierName: "   "}
	if got := cfg.EffectiveSupplierName(); got != "Eventory" {
		t.Errorf("expected 'Eventory' for whitespace, got %q", got)
	}
}
