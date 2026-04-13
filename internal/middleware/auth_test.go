package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractAPIKey_XAPIKeyHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("X-API-Key", "test-key-123")

	got := extractAPIKey(req)
	if got != "test-key-123" {
		t.Errorf("expected %q, got %q", "test-key-123", got)
	}
}

func TestExtractAPIKey_BearerHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("Authorization", "Bearer my-bearer-token")

	got := extractAPIKey(req)
	if got != "my-bearer-token" {
		t.Errorf("expected %q, got %q", "my-bearer-token", got)
	}
}

func TestExtractAPIKey_XAPIKeyTakesPrecedence(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("X-API-Key", "key-from-header")
	req.Header.Set("Authorization", "Bearer key-from-bearer")

	got := extractAPIKey(req)
	if got != "key-from-header" {
		t.Errorf("X-API-Key should take precedence; expected %q, got %q", "key-from-header", got)
	}
}

func TestExtractAPIKey_EmptyHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/zone-types", nil)

	got := extractAPIKey(req)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExtractAPIKey_BearerWithWhitespace(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("Authorization", "Bearer   spaced-token  ")

	got := extractAPIKey(req)
	if got != "spaced-token" {
		t.Errorf("expected %q, got %q", "spaced-token", got)
	}
}

func TestExtractAPIKey_BearerEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("Authorization", "Bearer ")

	got := extractAPIKey(req)
	if got != "" {
		t.Errorf("expected empty string for empty Bearer token, got %q", got)
	}
}

func TestExtractAPIKey_NonBearerAuth(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	got := extractAPIKey(req)
	if got != "" {
		t.Errorf("expected empty string for Basic auth, got %q", got)
	}
}

// TestAuthMiddleware_NoCredentials_NoDB verifies that when the database is
// unavailable and no credentials are provided, a 500 is returned.
func TestAuthMiddleware_NoCredentials_NoDB(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// DB is nil in test environment → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when DB is nil, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "{\"error\":\"Database unavailable\"}\n" {
		t.Errorf("unexpected body: %s", body)
	}
}

// TestAuthMiddleware_InvalidAPIKey_NoDB verifies that an API key still gets
// 500 when the database is unavailable (not a misleading 401).
func TestAuthMiddleware_InvalidAPIKey_NoDB(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("X-API-Key", "some-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// DB is nil → 500 even with an API key
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when DB is nil, got %d", rr.Code)
	}
}

// TestAuthMiddleware_InvalidBearerToken_NoDB verifies that a Bearer token
// still gets 500 when the database is unavailable.
func TestAuthMiddleware_InvalidBearerToken_NoDB(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("Authorization", "Bearer some-invalid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// DB is nil → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when DB is nil, got %d", rr.Code)
	}
}
