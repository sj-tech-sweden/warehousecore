package services

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
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
	// /inventory-rentals should have been tried first and returned 404
	if callCounts["/inventory-rentals"] == 0 {
		t.Error("expected /inventory-rentals to have been tried")
	}
	// legacy endpoints should also have been tried
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

// ===========================
// /inventory-rentals tree tests
// ===========================

// TestFetchInventoryRentals_Tree verifies that fetchInventoryRentals correctly
// parses a hierarchical category/item tree and enriches each leaf with price and
// description fetched from /rentals/{id}.
func TestFetchInventoryRentals_Tree(t *testing.T) {
	inventoryBody := `[
		{
			"id": "cat-1",
			"name": "Sound",
			"children": [
				{
					"id": "item-1",
					"name": "Mixer",
					"articleNumber": "S001",
					"stockLevel": 2,
					"is_pack": false
				}
			]
		},
		{
			"id": "cat-2",
			"name": "Lights",
			"children": [
				{
					"id": "cat-2-1",
					"name": "LED",
					"children": [
						{
							"id": "item-2",
							"name": "LED Bar",
							"articleNumber": "L001",
							"stockLevel": null,
							"is_pack": false
						}
					]
				}
			]
		}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/inventory-rentals":
			w.Write([]byte(inventoryBody)) //nolint:errcheck
		case "/rentals/item-1":
			w.Write([]byte(`{"rental":{"id":"item-1","name":"Mixer","description":"Audio mixer","dailyRate":150}}`)) //nolint:errcheck
		case "/rentals/item-2":
			w.Write([]byte(`{"rental":{"id":"item-2","name":"LED Bar","description":"Stage light","dailyRate":50}}`)) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL}
	products, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d: %+v", len(products), products)
	}

	// Check Mixer
	var mixer, ledBar EventoryProduct
	for _, p := range products {
		switch p.Name {
		case "Mixer":
			mixer = p
		case "LED Bar":
			ledBar = p
		}
	}
	if mixer.Name != "Mixer" {
		t.Errorf("expected Mixer product, got %+v", mixer)
	}
	if mixer.Category != "Sound" {
		t.Errorf("expected Mixer category 'Sound', got %q", mixer.Category)
	}
	if mixer.Description != "Audio mixer" {
		t.Errorf("expected Mixer description 'Audio mixer', got %q", mixer.Description)
	}
	if mixer.Price != 150 {
		t.Errorf("expected Mixer price 150, got %f", mixer.Price)
	}

	// Check LED Bar (nested category path)
	if ledBar.Name != "LED Bar" {
		t.Errorf("expected LED Bar product, got %+v", ledBar)
	}
	if ledBar.Category != "Lights > LED" {
		t.Errorf("expected LED Bar category 'Lights > LED', got %q", ledBar.Category)
	}
	if ledBar.Description != "Stage light" {
		t.Errorf("expected LED Bar description 'Stage light', got %q", ledBar.Description)
	}
	if ledBar.Price != 50 {
		t.Errorf("expected LED Bar price 50, got %f", ledBar.Price)
	}
}

// TestFetchInventoryRentals_EmptyCategory verifies that category nodes with no
// leaf children do not produce spurious products.
func TestFetchInventoryRentals_EmptyCategory(t *testing.T) {
	inventoryBody := `[
		{"id": "cat-empty", "name": "Empty Category", "children": []}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/inventory-rentals" {
			w.Write([]byte(inventoryBody)) //nolint:errcheck
		} else {
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL}
	products, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("expected 0 products from empty category, got %d: %+v", len(products), products)
	}
}

// TestFetchInventoryRentals_DetailFetchFailure verifies that when /rentals/{id}
// fails for a leaf item, the product is still returned with zeroed price and
// empty description (the sync must not abort due to a single detail failure).
func TestFetchInventoryRentals_DetailFetchFailure(t *testing.T) {
	inventoryBody := `[
		{
			"id": "cat-1",
			"name": "Sound",
			"children": [
				{"id": "item-1", "name": "Speaker", "articleNumber": "S001", "stockLevel": 1, "is_pack": false}
			]
		}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/inventory-rentals" {
			w.Write([]byte(inventoryBody)) //nolint:errcheck
		} else {
			// All detail fetches fail
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL}
	products, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product even when detail fetch fails, got %d", len(products))
	}
	if products[0].Name != "Speaker" {
		t.Errorf("expected Speaker, got %q", products[0].Name)
	}
	if products[0].Price != 0 {
		t.Errorf("expected price 0 when detail unavailable, got %f", products[0].Price)
	}
	if products[0].Description != "" {
		t.Errorf("expected empty description when detail unavailable, got %q", products[0].Description)
	}
}

// TestFetchInventoryRentals_AuthHeadersForwarded verifies that the Bearer token
// and X-API-Key are forwarded to both /inventory-rentals and /rentals/{id}.
func TestFetchInventoryRentals_AuthHeadersForwarded(t *testing.T) {
	inventoryBody := `[
		{"id": "item-1", "name": "Widget", "articleNumber": "W001", "stockLevel": 1, "is_pack": false}
	]`

	authOnInventory := ""
	authOnDetail := ""
	apiKeyOnInventory := ""
	apiKeyOnDetail := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/inventory-rentals":
			authOnInventory = r.Header.Get("Authorization")
			apiKeyOnInventory = r.Header.Get("X-API-Key")
			w.Write([]byte(inventoryBody)) //nolint:errcheck
		case "/rentals/item-1":
			authOnDetail = r.Header.Get("Authorization")
			apiKeyOnDetail = r.Header.Get("X-API-Key")
			w.Write([]byte(`{"rental":{"id":"item-1","name":"Widget","description":"","dailyRate":0}}`)) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL, APIKey: "my-api-key"}
	_, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if authOnInventory != "Bearer my-api-key" {
		t.Errorf("inventory-rentals: Authorization = %q, want %q", authOnInventory, "Bearer my-api-key")
	}
	if apiKeyOnInventory != "my-api-key" {
		t.Errorf("inventory-rentals: X-API-Key = %q, want %q", apiKeyOnInventory, "my-api-key")
	}
	if authOnDetail != "Bearer my-api-key" {
		t.Errorf("rentals/{id}: Authorization = %q, want %q", authOnDetail, "Bearer my-api-key")
	}
	if apiKeyOnDetail != "my-api-key" {
		t.Errorf("rentals/{id}: X-API-Key = %q, want %q", apiKeyOnDetail, "my-api-key")
	}
}

// TestFetchInventoryRentals_FallsBackToLegacy verifies that when /inventory-rentals
// returns 404 the legacy flat-list endpoints are tried.
func TestFetchInventoryRentals_FallsBackToLegacy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/products" {
			json.NewEncoder(w).Encode([]EventoryProduct{{Name: "Legacy Product"}}) //nolint:errcheck
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
	if len(products) != 1 || products[0].Name != "Legacy Product" {
		t.Errorf("unexpected products from legacy fallback: %+v", products)
	}
}

// TestFetchInventoryRentals_NoFallbackOn401 verifies that a 401 from
// /inventory-rentals is returned directly to the caller and does NOT trigger
// the legacy endpoint fallback — preventing masked misconfigurations.
func TestFetchInventoryRentals_NoFallbackOn401(t *testing.T) {
	callCounts := map[string]int{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCounts[r.URL.Path]++
		if r.URL.Path == "/inventory-rentals" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Legacy endpoints return products — should never be reached.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EventoryProduct{{Name: "Legacy Product"}}) //nolint:errcheck
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL}
	_, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err == nil {
		t.Fatal("expected error for 401 from inventory-rentals, got nil")
	}
	// Legacy endpoints must not have been tried.
	for _, path := range []string{"/api/v1/products", "/api/products", "/products"} {
		if callCounts[path] > 0 {
			t.Errorf("legacy endpoint %s should not have been called on 401, but got %d calls", path, callCounts[path])
		}
	}
}

// TestFetchInventoryRentals_NoFallbackOn500 verifies that a 5xx from
// /inventory-rentals is returned directly to the caller and does NOT trigger
// the legacy endpoint fallback.
func TestFetchInventoryRentals_NoFallbackOn500(t *testing.T) {
	callCounts := map[string]int{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCounts[r.URL.Path]++
		if r.URL.Path == "/inventory-rentals" {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EventoryProduct{{Name: "Legacy Product"}}) //nolint:errcheck
	}))
	defer srv.Close()

	cfg := &EventoryConfig{APIURL: srv.URL}
	_, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err == nil {
		t.Fatal("expected error for 500 from inventory-rentals, got nil")
	}
	for _, path := range []string{"/api/v1/products", "/api/products", "/products"} {
		if callCounts[path] > 0 {
			t.Errorf("legacy endpoint %s should not have been called on 500, but got %d calls", path, callCounts[path])
		}
	}
}

// TestFetchRentalDetail_PathEscapesID verifies that a rental ID containing
// path-special characters is properly escaped before being placed in the URL,
// preventing path traversal or host-override attacks.
func TestFetchRentalDetail_PathEscapesID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RawPath // use RawPath to see the encoded form
		if gotPath == "" {
			gotPath = r.URL.Path // fallback when no special chars need encoding
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"rental":{"description":"ok","dailyRate":10}}`)) //nolint:errcheck
	}))
	defer srv.Close()

	// An ID with slashes that, if unescaped, could traverse path boundaries.
	id := "../../etc/passwd"
	fetchRentalDetail(srv.Client(), srv.URL, id, "", "")

	// The server should see the slashes as percent-encoded %2F, not raw /.
	want := "/rentals/" + neturl.PathEscape(id)
	if gotPath != want {
		t.Errorf("expected path %q, got %q (raw slashes would indicate path traversal)", want, gotPath)
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

// TestFetchEventoryProducts_OAuthPasswordGrant verifies the OAuth2 Resource Owner
// Password Credentials flow end-to-end:
//   - the token request POSTs grant_type=password, username, and password as form fields
//   - the returned access token is used as Authorization: Bearer on product requests
//   - when an API key is also configured, X-API-Key is still sent with the API key
//     (not the OAuth token) on product requests
func TestFetchEventoryProducts_OAuthPasswordGrant(t *testing.T) {
	const wantToken = "oauth-access-token-abc"
	const apiKey = "static-api-key"

	var (
		gotGrantType, gotUsername, gotPassword string
		gotProductAuth, gotProductAPIKey       string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/oauth/token":
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			gotGrantType = r.FormValue("grant_type")
			gotUsername = r.FormValue("username")
			gotPassword = r.FormValue("password")
			json.NewEncoder(w).Encode(map[string]string{"access_token": wantToken})
		default:
			gotProductAuth = r.Header.Get("Authorization")
			gotProductAPIKey = r.Header.Get("X-API-Key")
			json.NewEncoder(w).Encode([]EventoryProduct{{Name: "Widget"}})
		}
	}))
	defer srv.Close()

	cfg := &EventoryConfig{
		APIURL:   srv.URL,
		Username: "user@example.com",
		Password: "s3cr3t",
		APIKey:   apiKey,
	}
	products, err := fetchEventoryProductsWith(cfg, srv.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) == 0 {
		t.Fatal("expected at least one product")
	}

	// Token request assertions.
	if gotGrantType != "password" {
		t.Errorf("token: grant_type = %q, want %q", gotGrantType, "password")
	}
	if gotUsername != cfg.Username {
		t.Errorf("token: username = %q, want %q", gotUsername, cfg.Username)
	}
	if gotPassword != cfg.Password {
		t.Errorf("token: password = %q, want %q", gotPassword, cfg.Password)
	}

	// Product request assertions.
	wantBearer := "Bearer " + wantToken
	if gotProductAuth != wantBearer {
		t.Errorf("product: Authorization = %q, want %q", gotProductAuth, wantBearer)
	}
	if gotProductAPIKey != apiKey {
		t.Errorf("product: X-API-Key = %q, want %q", gotProductAPIKey, apiKey)
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

// TestEncryptCredential_PrefixCollision verifies that a plaintext API key that
// starts with "enc:" is safely round-tripped when encryption is disabled. Without
// the rawEscapePrefix guard, decryptCredential would attempt (and fail) to decrypt
// what is actually plain text.
func TestEncryptCredential_PrefixCollision(t *testing.T) {
	colliding := "enc:looks-like-encrypted-but-isnt"
	// Encryption disabled (nil key): should be escaped, not returned as-is.
	out, err := encryptCredential(colliding, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, rawEscapePrefix) {
		t.Fatalf("expected rawEscapePrefix %q on collision value, got %q", rawEscapePrefix, out)
	}
	// Decrypt (with nil key) should recover original.
	dec, err := decryptCredential(out, nil)
	if err != nil {
		t.Fatalf("unexpected error on decrypt: %v", err)
	}
	if dec != colliding {
		t.Errorf("expected %q, got %q", colliding, dec)
	}
}

// TestEventoryCredentialKey_Base64WrongLength verifies that a base64 string that
// decodes to a length other than 32 returns an error referencing the decoded
// length rather than silently falling back to a raw-byte length check.
func TestEventoryCredentialKey_Base64WrongLength(t *testing.T) {
	// 16 raw bytes base64-encoded → decodes to 16, not 32
	shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16))
	t.Setenv("EVENTORY_CREDENTIAL_KEY", shortKey)
	_, err := eventoryCredentialKey()
	if err == nil {
		t.Fatal("expected error for base64-decoded key of wrong length, got nil")
	}
	// Error should mention the decoded byte count (16), not the raw string length.
	if !strings.Contains(err.Error(), "16") {
		t.Errorf("expected error to mention decoded length 16, got: %v", err)
	}
}
