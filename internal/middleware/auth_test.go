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

// TestAuthMiddleware_NoCredentials verifies that requests without any
// credentials receive a 401 with the "No session" message.
func TestAuthMiddleware_NoCredentials(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "{\"error\":\"Unauthorized - No session\"}\n" {
		t.Errorf("unexpected body: %s", body)
	}
}

// TestAuthMiddleware_InvalidAPIKey verifies that an invalid API key gets a 401
// with the "Invalid API key" message (not "No session").
func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("X-API-Key", "definitely-not-a-valid-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "{\"error\":\"Unauthorized - Invalid API key\"}\n" {
		t.Errorf("unexpected body: %s", body)
	}
}

// TestAuthMiddleware_InvalidBearerToken verifies that an invalid Bearer token
// also produces the "Invalid API key" message.
func TestAuthMiddleware_InvalidBearerToken(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/admin/zone-types", nil)
	req.Header.Set("Authorization", "Bearer some-invalid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "{\"error\":\"Unauthorized - Invalid API key\"}\n" {
		t.Errorf("unexpected body: %s", body)
	}
}
