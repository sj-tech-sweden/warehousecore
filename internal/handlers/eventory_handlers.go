package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"warehousecore/internal/repository"
	"warehousecore/internal/services"
)

// ===========================
// EVENTORY INTEGRATION HANDLERS
// ===========================

// GetEventorySettings returns the current Eventory integration configuration
// The API key is masked in the response so it is never exposed to the browser.
func GetEventorySettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := services.GetEventoryConfig()
	if err != nil {
		log.Printf("[EVENTORY] Failed to get config: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load Eventory settings"})
		return
	}

	maskedKey := ""
	if cfg.APIKey != "" {
		maskedKey = maskAPIKey(cfg.APIKey)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_url":            cfg.APIURL,
		"api_key_configured": cfg.APIKey != "",
		"api_key_masked":     maskedKey,
	})
}

// UpdateEventorySettings saves the Eventory API URL and (optionally) the API key.
// Sending an empty api_key in the payload leaves the existing stored key unchanged.
func UpdateEventorySettings(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		APIURL string `json:"api_url"`
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	payload.APIURL = strings.TrimSpace(payload.APIURL)
	if payload.APIURL == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "API URL is required"})
		return
	}

	// If no new key is provided, keep the existing one
	apiKey := strings.TrimSpace(payload.APIKey)
	if apiKey == "" {
		existing, err := services.GetEventoryConfig()
		if err == nil {
			apiKey = existing.APIKey
		}
	}

	cfg := &services.EventoryConfig{
		APIURL: payload.APIURL,
		APIKey: apiKey,
	}

	if err := services.SaveEventoryConfig(cfg); err != nil {
		log.Printf("[EVENTORY] Failed to save config: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save Eventory settings"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_url":            cfg.APIURL,
		"api_key_configured": cfg.APIKey != "",
		"message":            "Eventory settings saved successfully",
	})
}

// GetEventoryProducts fetches the product list directly from the Eventory API
// and returns it to the browser (useful for previewing before syncing).
func GetEventoryProducts(w http.ResponseWriter, r *http.Request) {
	cfg, err := services.GetEventoryConfig()
	if err != nil {
		log.Printf("[EVENTORY] Failed to get config: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load Eventory settings"})
		return
	}

	products, err := services.FetchEventoryProducts(cfg)
	if err != nil {
		log.Printf("[EVENTORY] Failed to fetch products: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("Failed to fetch from Eventory: %v", err)})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"products": products,
		"count":    len(products),
	})
}

// SyncEventoryProducts fetches products from Eventory and upserts them into
// the local rental_equipment table using the supplier name "Eventory".
func SyncEventoryProducts(w http.ResponseWriter, r *http.Request) {
	cfg, err := services.GetEventoryConfig()
	if err != nil {
		log.Printf("[EVENTORY] Failed to get config: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load Eventory settings"})
		return
	}

	products, err := services.FetchEventoryProducts(cfg)
	if err != nil {
		log.Printf("[EVENTORY] Failed to fetch products for sync: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("Failed to fetch from Eventory: %v", err)})
		return
	}

	db := repository.GetSQLDB()
	if db == nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Database not available"})
		return
	}

	imported := 0
	updated := 0
	skipped := 0

	for _, p := range products {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			skipped++
			continue
		}

		category := strings.TrimSpace(p.Category)
		description := strings.TrimSpace(p.Description)

		// Check whether a row with this product name and supplier already exists
		var existingID int
		err := db.QueryRow(
			`SELECT equipment_id FROM rental_equipment WHERE product_name = $1 AND supplier_name = 'Eventory'`,
			name,
		).Scan(&existingID)

		if err != nil {
			// Not found – insert
			_, insertErr := db.Exec(`
				INSERT INTO rental_equipment
					(product_name, supplier_name, rental_price, customer_price, category, description, is_active, created_at, updated_at)
				VALUES ($1, 'Eventory', $2, 0, $3, $4, TRUE, $5, $5)
			`, name, p.Price, nullableStr(category), nullableStr(description), time.Now())
			if insertErr != nil {
				log.Printf("[EVENTORY] Failed to insert product %q: %v", name, insertErr)
				skipped++
			} else {
				imported++
			}
		} else {
			// Exists – update
			_, updateErr := db.Exec(`
				UPDATE rental_equipment
				SET rental_price = $1, category = $2, description = $3, updated_at = $4
				WHERE equipment_id = $5
			`, p.Price, nullableStr(category), nullableStr(description), time.Now(), existingID)
			if updateErr != nil {
				log.Printf("[EVENTORY] Failed to update product %q (id=%d): %v", name, existingID, updateErr)
				skipped++
			} else {
				updated++
			}
		}
	}

	log.Printf("[EVENTORY] Sync complete: %d imported, %d updated, %d skipped", imported, updated, skipped)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"imported": imported,
		"updated":  updated,
		"skipped":  skipped,
		"total":    len(products),
		"message":  fmt.Sprintf("Sync complete: %d imported, %d updated, %d skipped", imported, updated, skipped),
	})
}

// maskAPIKey returns a masked version of an API key, showing only the first 4 and last 4 chars.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

// nullableStr returns nil for empty strings (for nullable DB columns).
func nullableStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
