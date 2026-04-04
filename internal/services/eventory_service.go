package services

import (
	"context"
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
	APIURL string `json:"api_url"`
	// APIKey is used when the Eventory instance uses a static bearer token.
	APIKey string `json:"api_key"`
	// Username and Password are used for the OAuth2 Resource Owner Password
	// Credentials (ROPC) grant when the Eventory instance requires login.
	Username string `json:"username"`
	Password string `json:"password"`
	// TokenEndpoint is the OAuth2 token URL. Defaults to /oauth/token relative
	// to APIURL when empty.
	TokenEndpoint string `json:"token_endpoint"`
	// SupplierName is the value stored in rental_equipment.supplier_name for
	// products imported from this Eventory account. Defaults to "Eventory".
	SupplierName string `json:"supplier_name"`
	// SyncIntervalMinutes controls automatic background syncing.
	// 0 means disabled; positive values trigger a sync every N minutes.
	SyncIntervalMinutes int `json:"sync_interval_minutes"`
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

	// Validate the URL before saving to prevent seeding an unsafe address.
	if err := ValidateEventoryURL(apiURL); err != nil {
		log.Printf("[EVENTORY] Bootstrap: EVENTORY_API_URL is invalid/unsafe (%v), skipping seed", err)
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
// endpoint paths and handles various response JSON shapes. Outbound requests use
// a custom transport that rejects connections to private/reserved IPs at dial
// time, preventing DNS rebinding attacks.
func FetchEventoryProducts(cfg *EventoryConfig) ([]EventoryProduct, error) {
	return fetchEventoryProductsWith(cfg, newSSRFSafeClient())
}

// fetchEventoryProductsWith is the testable core of FetchEventoryProducts.
// The caller supplies the HTTP client, allowing tests to inject a plain client
// targeting a local httptest.Server without triggering SSRF guards.
func fetchEventoryProductsWith(cfg *EventoryConfig, client *http.Client) ([]EventoryProduct, error) {
	if cfg.APIURL == "" {
		return nil, errors.New("Eventory API URL is not configured")
	}

	// Obtain access token via OAuth2 ROPC when credentials are provided.
	// The OAuth token is used for Authorization: Bearer; the API key (if set)
	// is always sent as X-API-Key regardless of which auth method is used.
	oauthToken := ""
	if cfg.Username != "" && cfg.Password != "" {
		token, err := fetchOAuthToken(client, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain Eventory access token: %w", err)
		}
		oauthToken = token
	}

	// Try common product endpoint paths. Paths are relative (no leading slash)
	// so that any configured path prefix in cfg.APIURL is preserved when the
	// final request URL is built by joinPath.
	endpoints := []string{
		"api/v1/products",
		"api/products",
		"products",
	}

	var lastErr error
	for _, endpoint := range endpoints {
		products, err := fetchFromEndpoint(client, cfg.APIURL, endpoint, oauthToken, cfg.APIKey)
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
		u, err := joinPath(cfg.APIURL, "oauth/token")
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

// fetchFromEndpoint fetches products from a single endpoint path.
// oauthToken is used for Authorization: Bearer (OAuth2 access token).
// apiKey is sent as X-API-Key (static API key). Either may be empty.
func fetchFromEndpoint(client *http.Client, baseURL, endpoint, oauthToken, apiKey string) ([]EventoryProduct, error) {
	fullURL, err := joinPath(baseURL, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Authorization: Bearer from the OAuth access token when available.
	// Fall back to the API key as the bearer value when no OAuth token is present.
	if oauthToken != "" {
		req.Header.Set("Authorization", "Bearer "+oauthToken)
	} else if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	// Always send X-API-Key from the configured API key (never from the OAuth token).
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
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
// URLs with embedded credentials, blocks loopback / private IP literals, and
// resolves the hostname to catch DNS-based SSRF.
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

	// If the host is a bare IP, check it directly.
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return errors.New("URL must not target a private IP address")
		}
		return nil
	}

	// For hostnames, resolve all A/AAAA records and reject any that are private
	// or loopback. This prevents DNS-based SSRF even when the literal hostname
	// looks safe. Fail closed: if DNS resolution fails we cannot verify that the
	// host is safe, so we reject the URL to prevent SSRF via unresolvable names
	// that could later resolve to private ranges. A short timeout prevents slow
	// DNS from hanging settings saves or server bootstrap.
	dnsCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(dnsCtx, host)
	if err != nil {
		return fmt.Errorf("URL hostname could not be resolved during validation: %w", err)
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && isPrivateIP(ip) {
			return fmt.Errorf("hostname resolves to a private/reserved IP address (%s)", addr)
		}
	}

	return nil
}

// isPrivateIP reports whether ip should be blocked from outbound SSRF checks.
// It rejects loopback, link-local, private RFC-1918/RFC-4193, unspecified
// (0.0.0.0 / ::), multicast, and all other RFC 6890 special-use ranges —
// in other words, any address that is not publicly routable.
func isPrivateIP(ip net.IP) bool {
	// Reject anything that is not a global unicast address first.
	// This covers 0.0.0.0/::, multicast (224.0.0.0/4, ff00::/8),
	// and loopback / link-local in one check.
	if !ip.IsGlobalUnicast() {
		return true
	}

	// IsGlobalUnicast() still returns true for many RFC 6890 special-use ranges
	// (RFC-1918 private, ULA, shared address space, benchmarking, TEST-NETs,
	// reserved, IETF protocol assignments, etc.) so we must check those explicitly.
	privateRanges := []string{
		"10.0.0.0/8",      // RFC 1918 private
		"172.16.0.0/12",   // RFC 1918 private
		"192.168.0.0/16",  // RFC 1918 private
		"169.254.0.0/16",  // link-local (belt-and-suspenders)
		"100.64.0.0/10",   // RFC 6598 shared address space (CGNAT)
		"198.18.0.0/15",   // RFC 2544 benchmarking
		"192.0.0.0/24",    // RFC 6890 IETF protocol assignments
		"192.0.2.0/24",    // RFC 5737 TEST-NET-1
		"198.51.100.0/24", // RFC 5737 TEST-NET-2
		"203.0.113.0/24",  // RFC 5737 TEST-NET-3
		"240.0.0.0/4",     // RFC 1112 reserved (class E)
		"fc00::/7",        // ULA
		"fe80::/10",       // link-local IPv6 (belt-and-suspenders)
		"2001:db8::/32",   // RFC 3849 documentation
		"100::/64",        // RFC 6666 discard-only (IPv6)
	}
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

// newSSRFSafeClient returns an *http.Client whose DialContext rejects connections
// to private, loopback, and other non-global-unicast addresses at connect time.
// This defends against DNS rebinding: even if ValidateEventoryURL allowed a host
// at configuration time, a rebind that resolves to a private IP at request time
// will be blocked here before any data is sent.
func newSSRFSafeClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// addr is "host:port"; split off the port
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address %q: %w", addr, err)
			}
			// Resolve the host to IPs. Fail if resolution fails.
			addrs, err := net.DefaultResolver.LookupHost(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %q: %w", host, err)
			}
			// Find the first safe (non-private) address and dial it.
			// If ANY resolved address is private we log it, but we only block
			// when no safe address is available to prevent the caller from
			// falling back to a private IP via round-robin.
			var safeAddr string
			for _, a := range addrs {
				ip := net.ParseIP(a)
				if ip != nil && isPrivateIP(ip) {
					continue // skip private addresses
				}
				safeAddr = a
				break
			}
			if safeAddr == "" {
				return nil, fmt.Errorf("blocked: all resolved addresses for %s are private/reserved", host)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(safeAddr, port))
		},
	}
	return &http.Client{Timeout: 15 * time.Second, Transport: transport}
}

// joinPath appends the relative path elem to the base URL, preserving any path
// prefix already present in base. elem must be a relative path without a leading
// slash. A trailing slash is added to the base path before resolving so that the
// relative reference is appended rather than replacing the last path segment.
func joinPath(base, elem string) (string, error) {
	u, err := neturl.Parse(base)
	if err != nil {
		return "", err
	}
	// Ensure base path ends with "/" so relative resolution appends correctly.
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	ref, err := neturl.Parse(elem)
	if err != nil {
		return "", err
	}
	return u.ResolveReference(ref).String(), nil
}

// envOrEmpty returns os.Getenv(key).
func envOrEmpty(key string) string {
	return os.Getenv(key)
}
