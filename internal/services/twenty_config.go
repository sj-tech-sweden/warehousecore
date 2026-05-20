package services

// twenty_config.go — persistence layer for the Twenty CRM integration settings.
//
// Settings are stored in app_settings under scope="warehousecore", key="twenty.config".
// The API key is encrypted at rest using the TWENTY_CREDENTIAL_KEY env var (AES-256-GCM),
// identical to how Eventory credentials are handled.
//
// BootstrapTwentyFromEnv seeds the settings from TWENTY_BASE_URL / TWENTY_API_KEY
// environment variables on first startup if no DB record exists yet.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/gorm"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

const (
	twentySettingScope = "warehousecore"
	twentySettingKey   = "twenty.config"
)

// TwentyConfig holds all configurable options for the Twenty CRM integration.
type TwentyConfig struct {
	// BaseURL is the root URL of the Twenty server, e.g. http://localhost:2020
	BaseURL string `json:"base_url"`
	// APIKey is the bearer token used to authenticate against the Twenty GraphQL API.
	APIKey string `json:"api_key"`
	// SyncIntervalMinutes controls how often products are pushed to Twenty.
	// 0 = disabled; valid non-zero values: 15, 30, 60, 120, 240, 480, 1440.
	SyncIntervalMinutes int `json:"sync_interval_minutes"`
	// EnableJobBootstrap controls whether the adapter auto-creates local job rows
	// for Twenty Opportunities that have no warehouseCoreJobId.
	EnableJobBootstrap bool `json:"enable_job_bootstrap"`
}

// GetTwentyConfig loads the Twenty config from the database.
// Returns an empty config (not an error) when no record exists yet.
func GetTwentyConfig() (*TwentyConfig, error) {
	adminSvc := NewAdminService()
	setting, err := adminSvc.GetSetting(twentySettingScope, twentySettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &TwentyConfig{EnableJobBootstrap: true}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query twenty config: %w", err)
	}

	b, err := json.Marshal(setting.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal twenty config: %w", err)
	}

	var cfg TwentyConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twenty config: %w", err)
	}

	key, err := twentyCredentialKey()
	if err != nil {
		return nil, err
	}
	if cfg.APIKey, err = decryptCredentialWithKeyName(cfg.APIKey, key, "TWENTY_CREDENTIAL_KEY"); err != nil {
		return nil, fmt.Errorf("failed to decrypt twenty api_key: %w", err)
	}

	return &cfg, nil
}

// SaveTwentyConfig persists the Twenty config to the database.
func SaveTwentyConfig(cfg *TwentyConfig) error {
	if repository.GetDB() == nil {
		return ErrDatabaseNotAvailable
	}

	key, err := twentyCredentialKey()
	if err != nil {
		return err
	}
	if key == nil {
		log.Printf("[TWENTY] WARNING: TWENTY_CREDENTIAL_KEY is not set; API key will be stored as plain text")
	}

	encAPIKey, err := encryptCredential(cfg.APIKey, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt twenty api_key: %w", err)
	}

	adminSvc := NewAdminService()
	return adminSvc.SetSetting(twentySettingScope, twentySettingKey, models.JSONMap{
		"base_url":              cfg.BaseURL,
		"api_key":               encAPIKey,
		"sync_interval_minutes": cfg.SyncIntervalMinutes,
		"enable_job_bootstrap":  cfg.EnableJobBootstrap,
	})
}

// BootstrapTwentyFromEnv seeds the Twenty config from supported Twenty env vars
// if no DB record exists yet.
func BootstrapTwentyFromEnv() {
	if repository.GetDB() == nil {
		return
	}

	baseURL := twentyBaseURLFromEnv()
	apiKey := twentyAPIKeyFromEnv()
	if baseURL == "" && apiKey == "" {
		return
	}

	var setting models.AppSetting
	err := repository.GetDB().Where("scope = ? AND key = ?", twentySettingScope, twentySettingKey).First(&setting).Error
	if err == nil {
		return // already configured
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[TWENTY] Bootstrap: failed to check existing config: %v", err)
		return
	}

	cfg := &TwentyConfig{
		BaseURL:             baseURL,
		APIKey:              apiKey,
		SyncIntervalMinutes: 0,
		EnableJobBootstrap:  true,
	}
	if err := SaveTwentyConfig(cfg); err != nil {
		log.Printf("[TWENTY] Bootstrap: failed to seed config: %v", err)
		return
	}
	log.Printf("[TWENTY] Bootstrap: seeded config from environment variables (base_url=%s)", baseURL)
}

// twentyBaseURLFromEnv resolves the base URL from supported aliases.
// Precedence returns the first non-empty value and prefers the canonical key first:
// 1) TWENTY_BASE_URL (preferred), then legacy/compat aliases
// 2) TWENTY_URL
// 3) TWENTY_SERVER_URL
// 4) TWENTY_GRAPHQL_URL
func twentyBaseURLFromEnv() string {
	for _, key := range []string{"TWENTY_BASE_URL", "TWENTY_URL", "TWENTY_SERVER_URL", "TWENTY_GRAPHQL_URL"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return normalizeTwentyBaseURL(raw)
		}
	}
	return ""
}

// twentyAPIKeyFromEnv resolves the API key/token from supported aliases.
// Precedence returns the first non-empty value and prefers the canonical key first:
// 1) TWENTY_API_KEY (preferred), then legacy/compat aliases
// 2) TWENTY_ACCESS_TOKEN
// 3) TWENTY_TOKEN
func twentyAPIKeyFromEnv() string {
	for _, key := range []string{"TWENTY_API_KEY", "TWENTY_ACCESS_TOKEN", "TWENTY_TOKEN"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return raw
		}
	}
	return ""
}

func normalizeTwentyBaseURL(raw string) string {
	v := strings.TrimRight(strings.TrimSpace(raw), "/")
	lower := strings.ToLower(v)
	const graphqlSuffix = "/graphql"
	if strings.HasSuffix(lower, graphqlSuffix) {
		v = v[:len(v)-len(graphqlSuffix)]
	}
	return v
}

// twentyCredentialKey returns the AES-256 key for encrypting/decrypting the
// Twenty API key. Falls back to TWENTY_CREDENTIAL_KEY; nil means no encryption.
func twentyCredentialKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("TWENTY_CREDENTIAL_KEY"))
	if raw == "" {
		return nil, nil
	}
	return parseCredentialKey(raw, "TWENTY_CREDENTIAL_KEY")
}
