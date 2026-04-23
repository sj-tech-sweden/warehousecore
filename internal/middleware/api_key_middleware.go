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

// APIKeyMiddleware protects service endpoints using an API key supplied via the
// X-API-Key request header. Query-parameter fallback is intentionally omitted:
// embedding credentials in URLs is a security risk (they appear in server logs,
// proxy caches, and Referer headers).
func APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("X-API-Key"))

		if key == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing API key"}) //nolint:errcheck
			return
		}

		valid, err := isAPIKeyValid(key)
		if err != nil {
			log.Printf("[APIKEY] database error during key validation: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database unavailable"}) //nolint:errcheck
			return
		}
		if !valid {
			w.Header().Set("Content-Type", "application/json")
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
	var lastUsedAt sql.NullTime
	err := db.QueryRow(
		`SELECT id, last_used_at FROM api_keys WHERE api_key_hash = $1 AND is_active = TRUE LIMIT 1`,
		hash,
	).Scan(&id, &lastUsedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // key not found – credential mismatch
		}
		return false, fmt.Errorf("api_key query: %w", err)
	}

	// Throttle last_used_at updates: only write if the recorded time is more
	// than 5 minutes old (or has never been set). This avoids unnecessary
	// write contention on high-frequency service-to-service traffic.
	const updateInterval = 5 * time.Minute
	if !lastUsedAt.Valid || time.Since(lastUsedAt.Time) > updateInterval {
		if _, err := db.Exec("UPDATE api_keys SET last_used_at = $1 WHERE id = $2", time.Now(), id); err != nil {
			log.Printf("WARN [WarehouseCore]: failed to update last_used_at for API key (id=%d): %v", id, err)
		}
	}

	return true, nil
}

func hashAPIKey(key string) string {
	return repository.HashAPIKey(key)
}
