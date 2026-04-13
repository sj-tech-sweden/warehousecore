package middleware

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
	"warehousecore/internal/services"

	"gorm.io/gorm"
)

type contextKey string

const UserContextKey = contextKey("user")

const authDebugLogsEnabled = false

func authDebugLog(format string, args ...interface{}) {
	if !authDebugLogsEnabled {
		return
	}
	log.Printf(format, args...)
}

// errDBUnavailable is returned by authenticate helpers when the database
// connection is nil or unreachable so the middleware can map it to HTTP 500.
var errDBUnavailable = errors.New("database unavailable")

// AuthMiddleware validates session cookie or admin API key and loads user.
// It first checks for a session_id cookie. If none is found, it falls back
// to X-API-Key header or Authorization: Bearer <key> header. Only API keys
// with is_admin=true are accepted; non-admin keys are rejected so that
// programmatic access via AuthMiddleware is limited to explicitly granted
// admin keys.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Debug: Log all cookies
		authDebugLog("DEBUG [WarehouseCore]: AuthMiddleware - Path: %s, Cookies: %+v", r.URL.Path, r.Cookies())

		// --- Try session cookie first ---
		hadSessionCookie := false
		cookie, err := r.Cookie("session_id")
		if err == nil && cookie.Value != "" {
			hadSessionCookie = true
			user, authErr := authenticateSession(cookie.Value)
			if authErr != nil {
				if errors.Is(authErr, errDBUnavailable) {
					http.Error(w, `{"error":"Database unavailable"}`, http.StatusInternalServerError)
					return
				}
				// DB query error (connection lost mid-request, etc.) → 500
				log.Printf("[AUTH] session auth DB error: %v", authErr)
				http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
				return
			}
			if user != nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// --- Fallback: admin API key via X-API-Key or Authorization: Bearer ---
		apiKey := extractAPIKey(r)
		if apiKey != "" {
			user, authErr := authenticateAdminAPIKey(apiKey)
			if authErr != nil {
				if errors.Is(authErr, errDBUnavailable) {
					http.Error(w, `{"error":"Database unavailable"}`, http.StatusInternalServerError)
					return
				}
				log.Printf("[AUTH] API key auth DB error: %v", authErr)
				http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
				return
			}
			if user != nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			http.Error(w, `{"error":"Unauthorized - Invalid API key"}`, http.StatusUnauthorized)
			return
		}

		// No valid credentials
		if hadSessionCookie {
			http.Error(w, `{"error":"Unauthorized - Invalid session"}`, http.StatusUnauthorized)
			return
		}
		authDebugLog("DEBUG [WarehouseCore]: No session_id cookie or API key found for %s", r.URL.Path)
		http.Error(w, `{"error":"Unauthorized - No session"}`, http.StatusUnauthorized)
	})
}

// extractAPIKey reads the raw API key from the request.
// It checks the X-API-Key header first, then the Authorization: Bearer header.
// The Bearer scheme check is case-insensitive per RFC 6750.
func extractAPIKey(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
		parts := strings.Fields(auth)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}
	return ""
}

// authenticateSession validates a session cookie value and returns the user,
// or (nil, nil) if the session is invalid/expired, or (nil, error) on DB
// failures so the caller can distinguish outages from credential mismatches.
func authenticateSession(cookieValue string) (*models.User, error) {
	sessionID, err := url.QueryUnescape(cookieValue)
	if err != nil {
		authDebugLog("DEBUG [WarehouseCore]: Failed to decode cookie: %v", err)
		return nil, nil // bad cookie value – not a DB error
	}

	authDebugLog("DEBUG [WarehouseCore]: Found session_id cookie (decoded: %s)", sessionID)

	db := repository.GetDB()
	if db == nil {
		return nil, errDBUnavailable
	}

	var session models.Session
	err = db.Preload("User").
		Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			authDebugLog("DEBUG [WarehouseCore]: Session not found for %s", sessionID)
			return nil, nil // credential mismatch
		}
		// Actual DB error (connection lost, etc.)
		authDebugLog("DEBUG [WarehouseCore]: Session query error for %s: %v", sessionID, err)
		return nil, fmt.Errorf("session query: %w", err)
	}

	authDebugLog("DEBUG [WarehouseCore]: Session valid for user: %s (ID: %d)", session.User.Username, session.User.UserID)

	if !session.User.IsActive {
		return nil, nil
	}

	rbacService := services.NewRBACService()
	if roles, err := rbacService.GetUserRoles(session.User.UserID); err == nil {
		session.User.Roles = roles
	} else {
		authDebugLog("DEBUG [WarehouseCore]: Failed to load roles for user %d: %v", session.User.UserID, err)
	}

	return &session.User, nil
}

// authenticateAdminAPIKey validates a raw API key against the api_keys table.
// Only keys with is_admin=true are accepted. Returns (nil, nil) when the key
// is not found or is not admin, and (nil, error) on DB failures.
func authenticateAdminAPIKey(raw string) (*models.User, error) {
	db := repository.GetSQLDB()
	if db == nil {
		return nil, errDBUnavailable
	}

	hash := repository.HashAPIKey(raw)

	var id int
	var name string
	var isAdmin bool
	err := db.QueryRow(
		`SELECT id, name, is_admin FROM api_keys WHERE api_key_hash = $1 AND is_active = TRUE LIMIT 1`,
		hash,
	).Scan(&id, &name, &isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // key not found – credential mismatch
		}
		return nil, fmt.Errorf("api_key query: %w", err)
	}

	// Only admin keys may authenticate via AuthMiddleware
	if !isAdmin {
		authDebugLog("DEBUG [WarehouseCore]: API key %q (id=%d) is not an admin key – rejecting", name, id)
		return nil, nil
	}

	// Update last_used_at synchronously (single indexed UPDATE).
	if _, err := db.Exec(`UPDATE api_keys SET last_used_at = $1 WHERE id = $2`, time.Now(), id); err != nil {
		log.Printf("WARN [WarehouseCore]: failed to update last_used_at for admin API key %q (id=%d): %v", name, id, err)
	}

	authDebugLog("DEBUG [WarehouseCore]: Admin API key authenticated: %q (id=%d)", name, id)

	// UserID 0 is a sentinel indicating API-key authentication (no real user row).
	return &models.User{
		UserID:   0,
		Username: "api-key:" + name,
		IsActive: true,
		IsAdmin:  true,
		Roles: []models.Role{
			{Name: "admin"},
			{Name: "warehouse_admin"},
		},
	}, nil
}

// OptionalAuthMiddleware loads user if session exists, but doesn't require it
func OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			// No session - continue without user
			next.ServeHTTP(w, r)
			return
		}

		// URL-decode the cookie value
		sessionID, err := url.QueryUnescape(cookie.Value)
		if err != nil {
			// Failed to decode - continue without user
			next.ServeHTTP(w, r)
			return
		}

		// Try to validate session
		db := repository.GetDB()
		if db != nil {
			var session models.Session
			err = db.Preload("User").
				Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).
				First(&session).Error

			if err == nil && session.User.IsActive {
				// Load roles for optional contexts too
				rbacService := services.NewRBACService()
				if roles, roleErr := rbacService.GetUserRoles(session.User.UserID); roleErr == nil {
					session.User.Roles = roles
				}
				// Valid session - add user to context
				ctx := context.WithValue(r.Context(), UserContextKey, &session.User)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the user from request context
func GetUserFromContext(r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	return user, ok
}
