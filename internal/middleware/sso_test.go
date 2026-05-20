package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"warehousecore/internal/repository"
)

func signHS256(claims ssoClaims, key []byte) (string, error) {
	hdr := map[string]string{"alg": "HS256", "typ": "JWT"}
	hdrB, _ := json.Marshal(hdr)
	plB, _ := json.Marshal(claims)
	h := base64.RawURLEncoding.EncodeToString(hdrB)
	p := base64.RawURLEncoding.EncodeToString(plB)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(h + "." + p))
	sig := mac.Sum(nil)
	s := base64.RawURLEncoding.EncodeToString(sig)
	return h + "." + p + "." + s, nil
}

func TestSSOMiddleware_CookieCreatesUserContext(t *testing.T) {
	os.Setenv("SSO_JWT_SECRET", "test-secret-sso")
	defer os.Unsetenv("SSO_JWT_SECRET")

	claims := ssoClaims{
		UserID:   77,
		Username: "alice",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}
	s, err := signHS256(claims, ssoSigningKey())
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sso_token", Value: s})
	rr := httptest.NewRecorder()

	handler := SSOMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r)
		if !ok || user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
}

func TestSSOMiddleware_InvalidTokenDoesNotPanic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sso_token", Value: "bad.token.here"})
	rr := httptest.NewRecorder()

	handler := SSOMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// should continue without user
		user, ok := GetUserFromContext(r)
		if ok && user != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
}

func TestSSOMiddleware_DoesNotAuthenticateWithoutSigningKey(t *testing.T) {
	_ = os.Unsetenv("SSO_JWT_SECRET")
	_ = os.Unsetenv("ENCRYPTION_KEY")

	claims := ssoClaims{
		UserID:   123,
		Username: "forged",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}
	s, err := signHS256(claims, []byte(""))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sso_token", Value: s})
	rr := httptest.NewRecorder()

	handler := SSOMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r)
		if ok && user != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
}

func TestSSOMiddleware_TokenWithoutExpIsRejected(t *testing.T) {
	os.Setenv("SSO_JWT_SECRET", "test-secret-sso")
	defer os.Unsetenv("SSO_JWT_SECRET")

	claims := ssoClaims{
		UserID:   456,
		Username: "no-exp",
		Iat:      time.Now().Unix(),
	}
	s, err := signHS256(claims, ssoSigningKey())
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sso_token", Value: s})
	rr := httptest.NewRecorder()

	handler := SSOMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r)
		if ok && user != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
}

func TestSSOMiddleware_PreservesPasswordHashFromDBUser(t *testing.T) {
	os.Setenv("SSO_JWT_SECRET", "test-secret-sso")
	defer os.Unsetenv("SSO_JWT_SECRET")

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DriverName:           "sqlmock",
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		sqlDB.Close()
		t.Fatalf("failed to create gorm db: %v", err)
	}
	restore := repository.WithTestDatabases(nil, gormDB)
	defer func() {
		restore()
		sqlDB.Close()
	}()

	claims := ssoClaims{
		UserID:   99,
		Username: "alice",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}
	s, err := signHS256(claims, ssoSigningKey())
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE userid = \$1 LIMIT \$2`).
		WithArgs(claims.UserID, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"userid", "username", "email", "password_hash", "first_name", "last_name", "is_admin", "is_active", "force_password_change", "created_at", "updated_at", "last_login",
		}).AddRow(
			claims.UserID, "alice", "alice@example.com", "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", "Alice", "User", false, true, false, now, now, nil,
		))
	mock.ExpectQuery(`SELECT .* FROM "roles" JOIN user_roles ON user_roles.roleid = roles.roleid WHERE user_roles.userid = \$1`).
		WithArgs(claims.UserID).
		WillReturnRows(sqlmock.NewRows([]string{
			"roleid", "name", "display_name", "description", "is_system_role", "is_active", "permissions", "created_at", "updated_at",
		}).AddRow(
			1, "admin", "Admin", "Administrator", true, true, []byte(`["*"]`), now, now,
		))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sso_token", Value: s})
	rr := httptest.NewRecorder()

	handler := SSOMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r)
		if !ok || user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if user.PasswordHash == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(user.Roles) != 1 || user.Roles[0].Name != "admin" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestSSOMiddleware_DeniesInactiveLocalUser(t *testing.T) {
	os.Setenv("SSO_JWT_SECRET", "test-secret-sso")
	defer os.Unsetenv("SSO_JWT_SECRET")

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DriverName:           "sqlmock",
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		sqlDB.Close()
		t.Fatalf("failed to create gorm db: %v", err)
	}
	restore := repository.WithTestDatabases(nil, gormDB)
	defer func() {
		restore()
		sqlDB.Close()
	}()

	claims := ssoClaims{
		UserID:   101,
		Username: "inactive",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}
	s, err := signHS256(claims, ssoSigningKey())
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE userid = \$1 LIMIT \$2`).
		WithArgs(claims.UserID, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"userid", "username", "email", "password_hash", "first_name", "last_name", "is_admin", "is_active", "force_password_change", "created_at", "updated_at", "last_login",
		}).AddRow(
			claims.UserID, "inactive", "inactive@example.com", "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", "Ina", "Ctive", false, false, false, now, now, nil,
		))

	nextCalled := false
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sso_token", Value: s})
	rr := httptest.NewRecorder()

	handler := SSOMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Fatalf("expected middleware to block request for inactive local user")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestSSOMiddleware_DeniesInactiveRentalCoreUser(t *testing.T) {
	os.Setenv("SSO_JWT_SECRET", "test-secret-sso")
	defer os.Unsetenv("SSO_JWT_SECRET")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userid":202,"username":"remote-inactive","is_active":false}`))
	}))
	defer server.Close()

	os.Setenv("RENTALCORE_BASE_URL", server.URL)
	defer os.Unsetenv("RENTALCORE_BASE_URL")

	claims := ssoClaims{
		UserID:   202,
		Username: "remote-inactive",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}
	s, err := signHS256(claims, ssoSigningKey())
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	nextCalled := false
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sso_token", Value: s})
	rr := httptest.NewRecorder()

	handler := SSOMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Fatalf("expected middleware to block request for inactive RentalCore user")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestSSOMiddleware_DeniesWhenRentalCoreLookupFails(t *testing.T) {
	os.Setenv("SSO_JWT_SECRET", "test-secret-sso")
	defer os.Unsetenv("SSO_JWT_SECRET")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	os.Setenv("RENTALCORE_BASE_URL", server.URL)
	defer os.Unsetenv("RENTALCORE_BASE_URL")

	claims := ssoClaims{
		UserID:   303,
		Username: "remote-missing",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}
	s, err := signHS256(claims, ssoSigningKey())
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	nextCalled := false
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sso_token", Value: s})
	rr := httptest.NewRecorder()

	handler := SSOMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Fatalf("expected middleware to block request when RentalCore lookup fails")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
}
