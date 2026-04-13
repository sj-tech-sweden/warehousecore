package middleware

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
	"warehousecore/internal/services"
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

// AuthMiddleware validates session cookie or API key and loads user.
// It first checks for a session_id cookie. If none is found, it falls back
// to X-API-Key header or Authorization: Bearer <key> header. API keys are
// validated against the api_keys table; keys with is_admin=true receive
// admin and warehouse_admin roles in the user context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Debug: Log all cookies
		authDebugLog("DEBUG [WarehouseCore]: AuthMiddleware - Path: %s, Cookies: %+v", r.URL.Path, r.Cookies())

		// --- Try session cookie first ---
		cookie, err := r.Cookie("session_id")
		if err == nil && cookie.Value != "" {
			if user := authenticateSession(cookie.Value); user != nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// --- Fallback: API key via X-API-Key header or Authorization: Bearer ---
		apiKey := extractAPIKey(r)
		if apiKey != "" {
			if user := authenticateAPIKey(apiKey); user != nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			http.Error(w, `{"error":"Unauthorized - Invalid API key"}`, http.StatusUnauthorized)
			return
		}

		// No credentials at all
		authDebugLog("DEBUG [WarehouseCore]: No session_id cookie or API key found for %s", r.URL.Path)
		http.Error(w, `{"error":"Unauthorized - No session"}`, http.StatusUnauthorized)
	})
}

// extractAPIKey reads the raw API key from the request.
// It checks the X-API-Key header first, then the Authorization: Bearer header.
func extractAPIKey(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		if key := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")); key != "" {
			return key
		}
	}
	return ""
}

// authenticateSession validates a session cookie value and returns the user,
// or nil if the session is invalid/expired.
func authenticateSession(cookieValue string) *models.User {
	sessionID, err := url.QueryUnescape(cookieValue)
	if err != nil {
		authDebugLog("DEBUG [WarehouseCore]: Failed to decode cookie: %v", err)
		return nil
	}

	authDebugLog("DEBUG [WarehouseCore]: Found session_id cookie (decoded: %s)", sessionID)

	db := repository.GetDB()
	if db == nil {
		return nil
	}

	var session models.Session
	err = db.Preload("User").
		Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).
		First(&session).Error
	if err != nil {
		authDebugLog("DEBUG [WarehouseCore]: Session validation failed for %s: %v", sessionID, err)
		return nil
	}

	authDebugLog("DEBUG [WarehouseCore]: Session valid for user: %s (ID: %d)", session.User.Username, session.User.UserID)

	if !session.User.IsActive {
		return nil
	}

	rbacService := services.NewRBACService()
	if roles, err := rbacService.GetUserRoles(session.User.UserID); err == nil {
		session.User.Roles = roles
	} else {
		authDebugLog("DEBUG [WarehouseCore]: Failed to load roles for user %d: %v", session.User.UserID, err)
	}

	return &session.User
}

// authenticateAPIKey validates a raw API key against the api_keys table.
// If the key is active and has is_admin=true, a synthetic admin user is
// returned so that downstream RequireAdmin / RequireRole checks pass.
func authenticateAPIKey(raw string) *models.User {
	db := repository.GetSQLDB()
	if db == nil {
		return nil
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
		return nil
	}

	// Best-effort last_used_at update
	go func(id int) {
		_, _ = db.Exec(`UPDATE api_keys SET last_used_at = $1 WHERE id = $2`, time.Now(), id)
	}(id)

	authDebugLog("DEBUG [WarehouseCore]: API key authenticated: %q (id=%d, is_admin=%v)", name, id, isAdmin)

	user := &models.User{
		Username: "api-key:" + name,
		IsActive: true,
		IsAdmin:  isAdmin,
	}

	if isAdmin {
		user.Roles = []models.Role{
			{Name: "admin"},
			{Name: "warehouse_admin"},
		}
	}

	return user
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
