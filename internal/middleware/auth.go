package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

type contextKey string

const UserContextKey = contextKey("user")

// AuthMiddleware validates session and loads user
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Debug: Log all cookies
		fmt.Printf("DEBUG [WarehouseCore]: AuthMiddleware - Path: %s, Cookies: %+v\n", r.URL.Path, r.Cookies())

		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			// No session cookie - return 401
			fmt.Printf("DEBUG [WarehouseCore]: No session_id cookie found for %s (error: %v)\n", r.URL.Path, err)
			http.Error(w, `{"error":"Unauthorized - No session"}`, http.StatusUnauthorized)
			return
		}

		// URL-decode the cookie value (browsers may URL-encode it)
		sessionID, err := url.QueryUnescape(cookie.Value)
		if err != nil {
			fmt.Printf("DEBUG [WarehouseCore]: Failed to decode cookie for %s: %v\n", r.URL.Path, err)
			http.Error(w, `{"error":"Unauthorized - Invalid cookie"}`, http.StatusUnauthorized)
			return
		}

		fmt.Printf("DEBUG [WarehouseCore]: Found session_id cookie: %s (decoded: %s) for path: %s\n", cookie.Value, sessionID, r.URL.Path)

		// Validate session in database
		db := repository.GetDB()
		if db == nil {
			http.Error(w, `{"error":"Database unavailable"}`, http.StatusInternalServerError)
			return
		}

		var session models.Session
		err = db.Preload("User").
			Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).
			First(&session).Error

		if err != nil {
			// Invalid or expired session
			fmt.Printf("DEBUG [WarehouseCore]: Session validation failed for %s: %v\n", sessionID, err)
			http.Error(w, `{"error":"Unauthorized - Invalid session"}`, http.StatusUnauthorized)
			return
		}

		fmt.Printf("DEBUG [WarehouseCore]: Session valid for user: %s (ID: %d)\n", session.User.Username, session.User.UserID)

		// Check if user is active
		if !session.User.IsActive {
			http.Error(w, `{"error":"Unauthorized - User inactive"}`, http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, &session.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
