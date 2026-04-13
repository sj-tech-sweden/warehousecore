package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"warehousecore/internal/repository"
)

// APIKeyMiddleware protects public endpoints using an API key (header X-API-Key or query param api_key).
func APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if key == "" {
			key = strings.TrimSpace(r.URL.Query().Get("api_key"))
		}

		if key == "" {
			http.Error(w, "missing API key", http.StatusUnauthorized)
			return
		}

		if !isAPIKeyValid(key) {
			http.Error(w, "invalid API key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAPIKeyValid(raw string) bool {
	db := repository.GetSQLDB()
	if db == nil {
		return false
	}
	hash := hashAPIKey(raw)

	var id int
	err := db.QueryRow(`SELECT id FROM api_keys WHERE api_key_hash = $1 AND is_active = TRUE LIMIT 1`, hash).Scan(&id)
	if err != nil {
		return false
	}

	// Update last_used_at synchronously (single indexed UPDATE).
	if _, err := db.Exec("UPDATE api_keys SET last_used_at = $1 WHERE id = $2", time.Now(), id); err != nil {
		log.Printf("WARN [WarehouseCore]: failed to update last_used_at for API key (id=%d): %v", id, err)
	}

	return true
}

func hashAPIKey(key string) string {
	return repository.HashAPIKey(key)
}
