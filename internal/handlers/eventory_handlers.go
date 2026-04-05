package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"warehousecore/internal/services"
)

// ===========================
// EVENTORY INTEGRATION HANDLERS
// ===========================

// eventoryConfigLoadErrorMsg returns an actionable admin-facing message for
// Eventory config load failures. Decryption / key-size errors are mapped to a
// hint about EVENTORY_CREDENTIAL_KEY so operators can recover without digging
// through server logs; everything else falls back to a generic message.
func eventoryConfigLoadErrorMsg(err error) string {
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "eventory_credential_key") ||
			strings.Contains(msg, "invalid key") ||
			strings.Contains(msg, "crypto/aes") ||
			strings.Contains(msg, "cipher") ||
			strings.Contains(msg, "decrypt") {
			return "Failed to load Eventory settings: encrypted credentials could not be decrypted — verify that EVENTORY_CREDENTIAL_KEY is set correctly"
		}
	}
	return "Failed to load Eventory settings"
}

// GetEventorySettings returns the current Eventory integration configuration.
// Secrets (API key, password) are masked so they are never exposed to the browser.
func GetEventorySettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := services.GetEventoryConfig()
	if err != nil {
		log.Printf("[EVENTORY] Failed to get config: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": eventoryConfigLoadErrorMsg(err)})
		return
	}

	maskedKey := ""
	if cfg.APIKey != "" {
		maskedKey = maskAPIKey(cfg.APIKey)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_url":                  cfg.APIURL,
		"api_key_configured":       cfg.APIKey != "",
		"api_key_masked":           maskedKey,
		"username":                 cfg.Username,
		"username_configured":      cfg.Username != "",
		"password_configured":      cfg.Password != "",
		"token_endpoint":           cfg.TokenEndpoint,
		"supplier_name":            cfg.SupplierName,
		"supplier_name_configured": strings.TrimSpace(cfg.SupplierName) != "",
		"supplier_name_effective":  cfg.EffectiveSupplierName(),
		"sync_interval_minutes":    cfg.SyncIntervalMinutes,
	})
}

// UpdateEventorySettings saves the Eventory connection settings.
// Empty api_key / password fields leave the existing stored values unchanged.
// Missing sync_interval_minutes (null in JSON) preserves the existing value.
func UpdateEventorySettings(w http.ResponseWriter, r *http.Request) {
	var rawPayload struct {
		APIURL              string `json:"api_url"`
		APIKey              string `json:"api_key"`
		Username            string `json:"username"`
		Password            string `json:"password"`
		TokenEndpoint       string `json:"token_endpoint"`
		SupplierName        string `json:"supplier_name"`
		SyncIntervalMinutes *int   `json:"sync_interval_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&rawPayload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	rawPayload.APIURL = strings.TrimSpace(rawPayload.APIURL)
	if rawPayload.APIURL == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "API URL is required"})
		return
	}

	// SSRF protection: validate the API URL
	if err := services.ValidateEventoryURL(rawPayload.APIURL); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid API URL: %v", err)})
		return
	}

	// Validate token_endpoint when provided — it is also used for outbound requests.
	tokenEndpoint := strings.TrimSpace(rawPayload.TokenEndpoint)
	if tokenEndpoint != "" {
		if err := services.ValidateEventoryURL(tokenEndpoint); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid token endpoint URL: %v", err)})
			return
		}
	}

	// Load existing config to preserve secrets and interval when new values are blank/absent
	existing, err := services.GetEventoryConfig()
	if err != nil {
		log.Printf("[EVENTORY] Failed to load existing config while preserving secrets: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load existing Eventory settings"})
		return
	}

	apiKey := strings.TrimSpace(rawPayload.APIKey)
	if apiKey == "" {
		apiKey = existing.APIKey
	}

	password := strings.TrimSpace(rawPayload.Password)
	if password == "" {
		password = existing.Password
	}

	// Require HTTPS for the effective OAuth token destination when username or
	// password credentials are configured. If token_endpoint is explicitly set,
	// it must be HTTPS. If it is blank, fetchOAuthToken will derive the token
	// endpoint from api_url (appending /oauth/token), so api_url must also be
	// HTTPS in that case to avoid sending credentials over cleartext HTTP.
	effectiveUsername := strings.TrimSpace(rawPayload.Username)
	if effectiveUsername != "" || password != "" {
		if tokenEndpoint != "" {
			if !strings.HasPrefix(strings.ToLower(tokenEndpoint), "https://") {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Token endpoint must use HTTPS when username or password credentials are configured"})
				return
			}
		} else if !strings.HasPrefix(strings.ToLower(rawPayload.APIURL), "https://") {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "API URL must use HTTPS when username or password credentials are configured and no token endpoint is provided"})
			return
		}
	}

	// Preserve existing sync interval if the field was omitted from the payload.
	syncIntervalMinutes := existing.SyncIntervalMinutes
	if rawPayload.SyncIntervalMinutes != nil {
		syncIntervalMinutes = *rawPayload.SyncIntervalMinutes
	}
	// Validate against the allowed set (0 = disabled; other values must match the
	// supported intervals).
	if syncIntervalMinutes != 0 && !services.IsAllowedSyncInterval(syncIntervalMinutes) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid sync_interval_minutes; allowed values: 0, 15, 30, 60, 120, 240, 480, 1440"})
		return
	}

	cfg := &services.EventoryConfig{
		APIURL:              rawPayload.APIURL,
		APIKey:              apiKey,
		Username:            effectiveUsername,
		Password:            password,
		TokenEndpoint:       tokenEndpoint,
		SupplierName:        strings.TrimSpace(rawPayload.SupplierName),
		SyncIntervalMinutes: syncIntervalMinutes,
	}

	if err := services.SaveEventoryConfig(cfg); err != nil {
		log.Printf("[EVENTORY] Failed to save config: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save Eventory settings"})
		return
	}

	// Restart the background scheduler with the new interval
	services.GetEventoryScheduler().Reset()

	maskedKey := ""
	if cfg.APIKey != "" {
		maskedKey = maskAPIKey(cfg.APIKey)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_url":                  cfg.APIURL,
		"api_key_configured":       cfg.APIKey != "",
		"api_key_masked":           maskedKey,
		"username":                 cfg.Username,
		"username_configured":      cfg.Username != "",
		"password_configured":      cfg.Password != "",
		"token_endpoint":           cfg.TokenEndpoint,
		"supplier_name":            cfg.SupplierName,
		"supplier_name_configured": strings.TrimSpace(cfg.SupplierName) != "",
		"supplier_name_effective":  cfg.EffectiveSupplierName(),
		"sync_interval_minutes":    cfg.SyncIntervalMinutes,
		"message":                  "Eventory settings saved successfully",
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
// the local rental_equipment table. Returns 409 if a sync is already running.
func SyncEventoryProducts(w http.ResponseWriter, r *http.Request) {
	sched := services.GetEventoryScheduler()
	if !sched.TryAcquireSync() {
		respondJSON(w, http.StatusConflict, map[string]string{"error": "A sync is already in progress"})
		return
	}
	defer sched.ReleaseSync()

	cfg, err := services.GetEventoryConfig()
	if err != nil {
		log.Printf("[EVENTORY] Failed to get config: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load Eventory settings"})
		return
	}

	imported, updated, skipped, total, err := services.RunEventorySync(cfg)
	if err != nil {
		log.Printf("[EVENTORY] Sync failed: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("Sync failed: %v", err)})
		return
	}

	log.Printf("[EVENTORY] Manual sync complete: %d imported, %d updated, %d skipped", imported, updated, skipped)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"imported": imported,
		"updated":  updated,
		"skipped":  skipped,
		"total":    total,
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
