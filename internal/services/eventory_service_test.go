package services

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	// Use a public routable IP that is not private (8.8.8.8 = Google DNS)
	// to avoid DNS lookups in environments without network access.
	if err := ValidateEventoryURL("https://8.8.8.8/api"); err != nil {
		t.Errorf("unexpected error for valid public IP URL: %v", err)
	}
}

func TestValidateEventoryURL_ValidHTTP(t *testing.T) {
	if err := ValidateEventoryURL("http://1.1.1.1/path"); err != nil {
		t.Errorf("unexpected error for valid http URL: %v", err)
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

func TestValidateEventoryURL_RejectsLoopbackIP(t *testing.T) {
	if err := ValidateEventoryURL("http://127.0.0.1/"); err == nil {
		t.Error("expected error for loopback IP URL, got nil")
	}
}

func TestValidateEventoryURL_RejectsLinkLocal(t *testing.T) {
	if err := ValidateEventoryURL("http://169.254.169.254/latest/meta-data/"); err == nil {
		t.Error("expected error for link-local IP URL, got nil")
	}
}

func TestValidateEventoryURL_RejectsEmbeddedCredentials(t *testing.T) {
	if err := ValidateEventoryURL("http://user:pass@1.1.1.1"); err == nil {
		t.Error("expected error for URL with credentials, got nil")
	}
}

func TestValidateEventoryURL_RejectsNonHTTP(t *testing.T) {
	if err := ValidateEventoryURL("ftp://1.1.1.1"); err == nil {
		t.Error("expected error for ftp:// URL, got nil")
	}
}

func TestValidateEventoryURL_RejectsMulticast(t *testing.T) {
	// 224.0.0.1 is a multicast address — not a valid outbound target.
	if err := ValidateEventoryURL("http://224.0.0.1/api"); err == nil {
		t.Error("expected error for multicast IP URL, got nil")
	}
}

func TestValidateEventoryURL_RejectsUnspecified(t *testing.T) {
	// 0.0.0.0 is unspecified — should be blocked.
	if err := ValidateEventoryURL("http://0.0.0.0/api"); err == nil {
		t.Error("expected error for unspecified (0.0.0.0) IP URL, got nil")
	}
}

func TestValidateEventoryURL_RejectsSharedAddressSpace(t *testing.T) {
	// 100.64.0.0/10 is RFC 6598 shared address space (CGNAT) — not publicly routable.
	if err := ValidateEventoryURL("http://100.64.0.1/api"); err == nil {
		t.Error("expected error for shared-address-space IP URL (100.64.0.0/10), got nil")
	}
}

func TestValidateEventoryURL_RejectsBenchmarking(t *testing.T) {
	// 198.18.0.0/15 is RFC 2544 benchmarking — not a valid outbound target.
	if err := ValidateEventoryURL("http://198.18.0.1/api"); err == nil {
		t.Error("expected error for benchmarking IP URL (198.18.0.0/15), got nil")
	}
}

func TestValidateEventoryURL_RejectsTestNet(t *testing.T) {
	// 192.0.2.0/24 is RFC 5737 TEST-NET-1 — documentation only, not routable.
	if err := ValidateEventoryURL("http://192.0.2.1/api"); err == nil {
		t.Error("expected error for TEST-NET-1 IP URL (192.0.2.0/24), got nil")
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
	products, err := fetchEventoryProductsWith(cfg, srv.Client())
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
	_, err := fetchEventoryProductsWith(cfg, srv.Client())
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
	_, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer my-token" {
		t.Errorf("expected Authorization: Bearer my-token, got %q", gotAuth)
	}
}

// TestFetchEventoryProducts_APIKeyHeaders verifies that when only an API key is
// configured, X-API-Key is set to the API key (not the OAuth token).
func TestFetchEventoryProducts_APIKeyHeaders(t *testing.T) {
	var gotAPIKey, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EventoryProduct{{Name: "Product"}})
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL, APIKey: "static-api-key"}
	_, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer static-api-key" {
		t.Errorf("expected Authorization: Bearer static-api-key, got %q", gotAuth)
	}
	if gotAPIKey != "static-api-key" {
		t.Errorf("expected X-API-Key: static-api-key, got %q", gotAPIKey)
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

// ===========================
// Credential encryption tests
// ===========================

func TestEncryptDecryptCredential_Roundtrip(t *testing.T) {
	key := []byte("12345678901234567890123456789012") // 32 bytes
	plaintext := "super-secret-api-key"

	enc, err := encryptCredential(plaintext, key)
	if err != nil {
		t.Fatalf("encryptCredential: %v", err)
	}
	if enc == plaintext {
		t.Fatal("expected encrypted value to differ from plaintext")
	}
	if !strings.HasPrefix(enc, encryptedPrefix) {
		t.Fatalf("expected encrypted prefix %q, got %q", encryptedPrefix, enc[:len(encryptedPrefix)])
	}

	dec, err := decryptCredential(enc, key)
	if err != nil {
		t.Fatalf("decryptCredential: %v", err)
	}
	if dec != plaintext {
		t.Errorf("expected %q, got %q", plaintext, dec)
	}
}

func TestEncryptCredential_NilKeyPassthrough(t *testing.T) {
	plaintext := "api-key"
	out, err := encryptCredential(plaintext, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != plaintext {
		t.Errorf("expected passthrough without key, got %q", out)
	}
}

func TestDecryptCredential_PlaintextPassthrough(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	plain := "not-encrypted"
	out, err := decryptCredential(plain, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != plain {
		t.Errorf("expected passthrough for non-prefixed value, got %q", out)
	}
}

func TestEncryptCredential_EmptyPassthrough(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	out, err := encryptCredential("", key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty string passthrough, got %q", out)
	}
}

func TestEncryptDecryptCredential_Base64Key(t *testing.T) {
	// Simulate a base64-encoded key as generated by `openssl rand -base64 32`
	rawKey := make([]byte, 32)
	for i := range rawKey {
		rawKey[i] = byte(i + 1)
	}
	base64Key := base64.StdEncoding.EncodeToString(rawKey)

	t.Setenv("EVENTORY_CREDENTIAL_KEY", base64Key)
	key, err := eventoryCredentialKey()
	if err != nil {
		t.Fatalf("eventoryCredentialKey: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}

	plaintext := "my-oauth-password"
	enc, err := encryptCredential(plaintext, key)
	if err != nil {
		t.Fatalf("encryptCredential: %v", err)
	}
	dec, err := decryptCredential(enc, key)
	if err != nil {
		t.Fatalf("decryptCredential: %v", err)
	}
	if dec != plaintext {
		t.Errorf("expected %q, got %q", plaintext, dec)
	}
}
