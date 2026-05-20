package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"warehousecore/internal/services"
)

// GetTwentySettings returns the persisted Twenty integration config.
func GetTwentySettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := services.GetTwentyConfig()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to load twenty settings: %v", err)})
		return
	}

	baseURL, baseURLSource, baseURLLocked := effectiveTwentyBaseURL(cfg)
	apiKey, apiKeySource, apiKeyLocked := effectiveTwentyAPIKey(cfg)

	masked := ""
	if apiKey != "" {
		masked = maskAPIKey(apiKey)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"base_url":              baseURL,
		"base_url_source":       baseURLSource,
		"base_url_locked":       baseURLLocked,
		"api_key_configured":    apiKey != "",
		"api_key_source":        apiKeySource,
		"api_key_locked":        apiKeyLocked,
		"api_key_masked":        masked,
		"sync_interval_minutes": cfg.SyncIntervalMinutes,
		"enable_job_bootstrap":  cfg.EnableJobBootstrap,
	})
}

// UpdateTwentySettings updates the persisted Twenty integration config.
// Empty api_key keeps the existing key unless clear_api_key=true.
func UpdateTwentySettings(w http.ResponseWriter, r *http.Request) {
	var raw struct {
		BaseURL             *string `json:"base_url"`
		APIKey              string  `json:"api_key"`
		ClearAPIKey         bool    `json:"clear_api_key"`
		SyncIntervalMinutes *int    `json:"sync_interval_minutes"`
		EnableJobBootstrap  *bool   `json:"enable_job_bootstrap"`
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	raw.APIKey = strings.TrimSpace(raw.APIKey)
	if raw.BaseURL != nil {
		trimmed := strings.TrimSpace(*raw.BaseURL)
		raw.BaseURL = &trimmed
	}

	existing, err := services.GetTwentyConfig()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to load existing settings: %v", err)})
		return
	}

	_, _, baseURLLocked := effectiveTwentyBaseURL(existing)
	_, _, apiKeyLocked := effectiveTwentyAPIKey(existing)

	if raw.BaseURL != nil && *raw.BaseURL != "" && !baseURLLocked {
		if err := validateTwentyBaseURL(*raw.BaseURL); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	if raw.SyncIntervalMinutes != nil && *raw.SyncIntervalMinutes != 0 && !services.IsAllowedTwentySyncInterval(*raw.SyncIntervalMinutes) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "sync_interval_minutes must be 0 or one of: 15, 30, 60, 120, 240, 480, 1440"})
		return
	}

	baseURL := existing.BaseURL
	if !baseURLLocked && raw.BaseURL != nil {
		baseURL = *raw.BaseURL
	}

	apiKey := existing.APIKey
	if apiKeyLocked {
		apiKey = existing.APIKey
	} else if raw.ClearAPIKey {
		apiKey = ""
	} else if raw.APIKey != "" {
		apiKey = raw.APIKey
	}

	enableBootstrap := existing.EnableJobBootstrap
	if raw.EnableJobBootstrap != nil {
		enableBootstrap = *raw.EnableJobBootstrap
	}

	syncInterval := existing.SyncIntervalMinutes
	if raw.SyncIntervalMinutes != nil {
		syncInterval = *raw.SyncIntervalMinutes
	}

	cfg := &services.TwentyConfig{
		BaseURL:             baseURL,
		APIKey:              apiKey,
		SyncIntervalMinutes: syncInterval,
		EnableJobBootstrap:  enableBootstrap,
	}
	if err := services.SaveTwentyConfig(cfg); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to save settings: %v", err)})
		return
	}

	// Re-apply scheduler immediately (includes initial sync run).
	services.GetTwentyScheduler().Reset()

	effectiveBaseURL, effectiveBaseURLSource, effectiveBaseURLLocked := effectiveTwentyBaseURL(cfg)
	effectiveAPIKey, effectiveAPIKeySource, effectiveAPIKeyLocked := effectiveTwentyAPIKey(cfg)

	masked := ""
	if effectiveAPIKey != "" {
		masked = maskAPIKey(effectiveAPIKey)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":               "Twenty settings saved",
		"base_url":              effectiveBaseURL,
		"base_url_source":       effectiveBaseURLSource,
		"base_url_locked":       effectiveBaseURLLocked,
		"api_key_configured":    effectiveAPIKey != "",
		"api_key_source":        effectiveAPIKeySource,
		"api_key_locked":        effectiveAPIKeyLocked,
		"api_key_masked":        masked,
		"sync_interval_minutes": cfg.SyncIntervalMinutes,
		"enable_job_bootstrap":  cfg.EnableJobBootstrap,
	})
}

// SyncTwentyProductsNow triggers an immediate sync from WarehouseCore products
// to Twenty and returns counters.
func SyncTwentyProductsNow(w http.ResponseWriter, r *http.Request) {
	counts, err := syncProductsToTwenty(r.Context())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("sync failed: %v", err)})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":             true,
		"created":        counts.TotalCreated(),
		"updated":        counts.TotalUpdated(),
		"productCreated": counts.ProductCreated,
		"productUpdated": counts.ProductUpdated,
		"packageCreated": counts.PackageCreated,
		"packageUpdated": counts.PackageUpdated,
	})
}

func validateTwentyBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid base_url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("base_url must use http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("base_url must include host")
	}
	if u.User != nil {
		return fmt.Errorf("base_url must not include credentials")
	}
	return nil
}

func effectiveTwentyBaseURL(cfg *services.TwentyConfig) (value string, source string, locked bool) {
	if env := twentyBaseURLFromEnv(); env != "" {
		return env, "env", true
	}
	if cfg != nil {
		if dbVal := strings.TrimSpace(cfg.BaseURL); dbVal != "" {
			return dbVal, "database", false
		}
	}
	return "", "none", false
}

func effectiveTwentyAPIKey(cfg *services.TwentyConfig) (value string, source string, locked bool) {
	if env := strings.TrimSpace(os.Getenv("TWENTY_API_KEY")); env != "" {
		return env, "env", true
	}
	if env := strings.TrimSpace(os.Getenv("TWENTY_ACCESS_TOKEN")); env != "" {
		return env, "env", true
	}
	if env := strings.TrimSpace(os.Getenv("TWENTY_TOKEN")); env != "" {
		return env, "env", true
	}
	if cfg != nil {
		if dbVal := strings.TrimSpace(cfg.APIKey); dbVal != "" {
			return dbVal, "database", false
		}
	}
	return "", "none", false
}

func twentyBaseURLFromEnv() string {
	for _, key := range []string{"TWENTY_BASE_URL", "TWENTY_URL", "TWENTY_SERVER_URL", "TWENTY_GRAPHQL_URL"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return normalizeTwentyBaseURL(raw)
		}
	}
	return ""
}

func normalizeTwentyBaseURL(raw string) string {
	v := strings.TrimRight(strings.TrimSpace(raw), "/")
	if strings.HasSuffix(strings.ToLower(v), "/graphql") {
		v = strings.TrimSuffix(v, "/graphql")
	}
	return v
}
