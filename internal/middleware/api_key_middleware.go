package middleware

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

		w.Header().Set("Content-Type", "application/json")
		if key == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing API key"}) //nolint:errcheck
			return
		}

		valid, err := isAPIKeyValid(key)
		if err != nil {
			log.Printf("[APIKEY] database error during key validation: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database unavailable"}) //nolint:errcheck
			return
		}
		if !valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid API key"}) //nolint:errcheck
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAPIKeyValid(raw string) (bool, error) {
	db := repository.GetSQLDB()
	if db == nil {
		return false, errors.New("SQL DB handle is nil")
	}
	hash := hashAPIKey(raw)

	var id int
	err := db.QueryRow(`SELECT id FROM api_keys WHERE api_key_hash = $1 AND is_active = TRUE LIMIT 1`, hash).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // key not found – credential mismatch
		}
		return false, fmt.Errorf("api_key query: %w", err)
	}

	// Update last_used_at synchronously (single indexed UPDATE).
	if _, err := db.Exec("UPDATE api_keys SET last_used_at = $1 WHERE id = $2", time.Now(), id); err != nil {
		log.Printf("WARN [WarehouseCore]: failed to update last_used_at for API key (id=%d): %v", id, err)
	}

	return true, nil
}

func hashAPIKey(key string) string {
	return repository.HashAPIKey(key)
}
