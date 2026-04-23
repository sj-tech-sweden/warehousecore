package middleware

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

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
			json.NewEncoder(w).Encode(map[string]string{"error": "database unavailable"}) //nolint:errcheck
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
	err := db.QueryRow(
		`SELECT id FROM api_keys WHERE api_key_hash = $1 AND is_active = TRUE LIMIT 1`,
		hash,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // key not found – credential mismatch
		}
		return false, fmt.Errorf("api_key query: %w", err)
	}

	// Throttle last_used_at updates using a conditional UPDATE at the DB level.
	// Because the WHERE clause checks the current value atomically, concurrent
	// requests for the same key only trigger one write per 5-minute window even
	// under burst load – avoiding the read-then-update race in application code.
	if _, err := db.Exec(
		`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < NOW() - INTERVAL '5 minutes')`,
		id,
	); err != nil {
		log.Printf("WARN [WarehouseCore]: failed to update last_used_at for API key (id=%d): %v", id, err)
	}

	return true, nil
}

func hashAPIKey(key string) string {
	return repository.HashAPIKey(key)
}
