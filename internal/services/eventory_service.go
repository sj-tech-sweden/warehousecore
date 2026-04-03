package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"

	"gorm.io/gorm"
)

const (
	eventorySettingScope = "warehousecore"
	eventorySettingKey   = "eventory.config"
)

// EventoryConfig holds Eventory API configuration
type EventoryConfig struct {
	APIURL              string `json:"api_url"`
	// APIKey is used when the Eventory instance uses a static bearer token.
	APIKey              string `json:"api_key"`
	// Username and Password are used for the OAuth2 Resource Owner Password
	// Credentials (ROPC) grant when the Eventory instance requires login.
	Username            string `json:"username"`
	Password            string `json:"password"`
	// TokenEndpoint is the OAuth2 token URL. Defaults to /oauth/token relative
	// to APIURL when empty.
	TokenEndpoint       string `json:"token_endpoint"`
	// SupplierName is the value stored in rental_equipment.supplier_name for
	// products imported from this Eventory account. Defaults to "Eventory".
	SupplierName        string `json:"supplier_name"`
	// SyncIntervalMinutes controls automatic background syncing.
	// 0 means disabled; positive values trigger a sync every N minutes.
	SyncIntervalMinutes int    `json:"sync_interval_minutes"`
}

// EffectiveSupplierName returns SupplierName or the default "Eventory".
func (c *EventoryConfig) EffectiveSupplierName() string {
	if s := strings.TrimSpace(c.SupplierName); s != "" {
		return s
	}
	return "Eventory"
}

// EventoryProduct represents a product from the Eventory API
type EventoryProduct struct {
	ID          interface{} `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Price       float64     `json:"price"`
}

// GetEventoryConfig retrieves the Eventory API configuration from settings.
func GetEventoryConfig() (*EventoryConfig, error) {
	adminSvc := NewAdminService()
	setting, err := adminSvc.GetSetting(eventorySettingScope, eventorySettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &EventoryConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query eventory config: %w", err)
	}

	b, err := json.Marshal(setting.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal eventory config: %w", err)
	}

	var cfg EventoryConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal eventory config: %w", err)
	}

	return &cfg, nil
}

// SaveEventoryConfig persists the Eventory API configuration using the shared
// AdminService.SetSetting helper.
func SaveEventoryConfig(cfg *EventoryConfig) error {
	if repository.GetDB() == nil {
		return ErrDatabaseNotAvailable
	}

	adminSvc := NewAdminService()
	value := models.JSONMap{
		"api_url":               cfg.APIURL,
		"api_key":               cfg.APIKey,
		"username":              cfg.Username,
		"password":              cfg.Password,
		"token_endpoint":        cfg.TokenEndpoint,
		"supplier_name":         cfg.SupplierName,
		"sync_interval_minutes": cfg.SyncIntervalMinutes,
	}

	return adminSvc.SetSetting(eventorySettingScope, eventorySettingKey, value)
}

// BootstrapEventoryFromEnv seeds the Eventory config from environment variables
// (EVENTORY_API_URL / EVENTORY_API_KEY) if the setting does not yet exist in
// the database. This runs once at startup.
func BootstrapEventoryFromEnv() {
	db := repository.GetDB()
	if db == nil {
		return
	}

	apiURL := strings.TrimSpace(envOrEmpty("EVENTORY_API_URL"))
	apiKey := strings.TrimSpace(envOrEmpty("EVENTORY_API_KEY"))
	if apiURL == "" {
		return // nothing to seed
	}

	// Only seed if no setting exists yet
	var setting models.AppSetting
	err := db.Where("scope = ? AND key = ?", eventorySettingScope, eventorySettingKey).First(&setting).Error
	if err == nil {
		return // already configured
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[EVENTORY] Bootstrap: failed to check existing config: %v", err)
		return
	}

	cfg := &EventoryConfig{APIURL: apiURL, APIKey: apiKey}
	if err := SaveEventoryConfig(cfg); err != nil {
		log.Printf("[EVENTORY] Bootstrap: failed to seed config from env: %v", err)
		return
	}
	log.Printf("[EVENTORY] Bootstrap: seeded config from environment variables")
}

// FetchEventoryProducts obtains an access token when credentials are configured,
// then fetches the product list from the Eventory API. It tries multiple common
// endpoint paths and handles various response JSON shapes.
func FetchEventoryProducts(cfg *EventoryConfig) ([]EventoryProduct, error) {
	if cfg.APIURL == "" {
		return nil, errors.New("Eventory API URL is not configured")
	}

	client := &http.Client{Timeout: 15 * time.Second}

	// Obtain access token via OAuth2 ROPC when credentials are provided
	bearerToken := cfg.APIKey
	if cfg.Username != "" && cfg.Password != "" {
		token, err := fetchOAuthToken(client, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain Eventory access token: %w", err)
		}
		bearerToken = token
	}

	// Try common product endpoint paths
	endpoints := []string{
		"/api/v1/products",
		"/api/products",
		"/products",
	}

	var lastErr error
	for _, endpoint := range endpoints {
		products, err := fetchFromEndpoint(client, cfg.APIURL, endpoint, bearerToken)
		if err == nil {
			return products, nil
		}
		lastErr = err
		log.Printf("[EVENTORY] Endpoint %s failed: %v", endpoint, err)
	}

	return nil, fmt.Errorf("all Eventory endpoints failed: %w", lastErr)
}

// fetchOAuthToken performs an OAuth2 Resource Owner Password Credentials grant
// and returns the access token string.
func fetchOAuthToken(client *http.Client, cfg *EventoryConfig) (string, error) {
	tokenEndpoint := strings.TrimSpace(cfg.TokenEndpoint)
	if tokenEndpoint == "" {
		u, err := joinPath(cfg.APIURL, "/oauth/token")
		if err != nil {
			return "", fmt.Errorf("failed to construct token endpoint URL: %w", err)
		}
		tokenEndpoint = u
	}

	form := neturl.Values{
		"grant_type": {"password"},
		"username":   {cfg.Username},
		"password":   {cfg.Password},
	}

	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", errors.New("token response contained no access_token")
	}

	return tokenResp.AccessToken, nil
}

func fetchFromEndpoint(client *http.Client, baseURL, endpoint, bearerToken string) ([]EventoryProduct, error) {
	fullURL, err := joinPath(baseURL, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
		req.Header.Set("X-API-Key", bearerToken)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("endpoint not found (404)")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized – check your credentials")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return parseEventoryProductsResponse(body)
}

// parseEventoryProductsResponse tries multiple common JSON shapes for a product list.
func parseEventoryProductsResponse(body []byte) ([]EventoryProduct, error) {
	// Shape 1: top-level array
	var directList []EventoryProduct
	if err := json.Unmarshal(body, &directList); err == nil {
		return directList, nil
	}

	// Shape 2: {"data": [...]} or {"products": [...]} or {"items": [...]}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(body, &wrapper); err == nil {
		for _, key := range []string{"data", "products", "items", "results"} {
			if raw, ok := wrapper[key]; ok {
				var list []EventoryProduct
				if err := json.Unmarshal(raw, &list); err == nil {
					return list, nil
				}
			}
		}
	}

	return nil, errors.New("unrecognised response format from Eventory API")
}

// ValidateEventoryURL checks that the given URL is safe to use for outbound
// HTTP requests (prevents SSRF). It requires an http or https scheme, rejects
// URLs with embedded credentials, and blocks loopback / private IP ranges.
func ValidateEventoryURL(rawURL string) error {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("URL must use http or https scheme")
	}
	if u.User != nil {
		return errors.New("URL must not contain embedded credentials")
	}

	host := u.Hostname()
	if host == "" {
		return errors.New("URL must contain a valid hostname")
	}

	// Reject obviously private / loopback hostnames
	lc := strings.ToLower(host)
	if lc == "localhost" || lc == "127.0.0.1" || lc == "::1" {
		return errors.New("URL must not target localhost")
	}

	// Attempt IP parse to block private ranges
	ip := net.ParseIP(host)
	if ip != nil {
		if isPrivateIP(ip) {
			return errors.New("URL must not target a private IP address")
		}
	}

	return nil
}

// isPrivateIP reports whether ip is in a private / link-local / loopback range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // link-local
		"::1/128",
		"fc00::/7", // ULA
		"fe80::/10", // link-local IPv6
	}
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

// joinPath safely joins a base URL and a path segment using url.JoinPath.
func joinPath(base, elem string) (string, error) {
	u, err := neturl.Parse(base)
	if err != nil {
		return "", err
	}
	result, err := u.Parse(elem)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

// envOrEmpty returns os.Getenv(key).
func envOrEmpty(key string) string {
	return os.Getenv(key)
}

