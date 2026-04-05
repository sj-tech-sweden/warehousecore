package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
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

// encryptedPrefix is prepended to credential values that have been encrypted
// at rest so that plain-text values stored before encryption was introduced can
// still be read transparently.
const encryptedPrefix = "enc:"

// eventoryCredentialKey returns the 32-byte AES-256 key read from the
// EVENTORY_CREDENTIAL_KEY environment variable. It returns (nil, nil) when the
// variable is not set (credentials are stored as plain text, with a warning).
// The value may be either a base64-encoded 32-byte key (recommended, produced
// by e.g. `openssl rand -base64 32`) or exactly 32 raw printable ASCII bytes.
// Accepting base64 allows callers to use full 256-bit entropy rather than being
// limited to printable characters.
func eventoryCredentialKey() ([]byte, error) {
	raw := os.Getenv("EVENTORY_CREDENTIAL_KEY")
	if raw == "" {
		return nil, nil
	}
	// Attempt base64 decoding first so operators can store a high-entropy key.
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	// Fall back to treating the raw string as bytes (legacy / simple 32-char ASCII key).
	key := []byte(raw)
	if len(key) != 32 {
		return nil, fmt.Errorf("EVENTORY_CREDENTIAL_KEY must decode to exactly 32 bytes (use `openssl rand -base64 32` to generate a suitable value); got %d bytes", len(key))
	}
	return key, nil
}

// encryptCredential encrypts plaintext using AES-256-GCM and returns a
// base64url-encoded string prefixed with encryptedPrefix. If key is nil the
// original plaintext is returned unchanged.
func encryptCredential(plaintext string, key []byte) (string, error) {
	if len(key) == 0 || plaintext == "" {
		return plaintext, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("rand nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decryptCredential reverses encryptCredential. Values that do not carry the
// encryptedPrefix (i.e., plain text stored before encryption was enabled) are
// returned as-is so that existing configurations continue to work.
func decryptCredential(stored string, key []byte) (string, error) {
	if !strings.HasPrefix(stored, encryptedPrefix) {
		return stored, nil
	}
	if len(key) == 0 {
		return "", fmt.Errorf("EVENTORY_CREDENTIAL_KEY is required to decrypt stored credentials")
	}
	data, err := base64.URLEncoding.DecodeString(stored[len(encryptedPrefix):])
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("gcm.Open: %w", err)
	}
	return string(plaintext), nil
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

	// Decrypt credentials at rest if they were stored encrypted.
	key, err := eventoryCredentialKey()
	if err != nil {
		return nil, err
	}
	if cfg.APIKey, err = decryptCredential(cfg.APIKey, key); err != nil {
		return nil, fmt.Errorf("failed to decrypt api_key: %w", err)
	}
	if cfg.Password, err = decryptCredential(cfg.Password, key); err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	return &cfg, nil
}

// SaveEventoryConfig persists the Eventory API configuration using the shared
// AdminService.SetSetting helper.
func SaveEventoryConfig(cfg *EventoryConfig) error {
	if repository.GetDB() == nil {
		return ErrDatabaseNotAvailable
	}

	key, err := eventoryCredentialKey()
	if err != nil {
		return err
	}
	if key == nil {
		log.Printf("[EVENTORY] WARNING: EVENTORY_CREDENTIAL_KEY is not set; API key and password will be stored as plain text in app_settings")
	}

	encAPIKey, err := encryptCredential(cfg.APIKey, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt api_key: %w", err)
	}
	encPassword, err := encryptCredential(cfg.Password, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	adminSvc := NewAdminService()
	value := models.JSONMap{
		"api_url":               cfg.APIURL,
		"api_key":               encAPIKey,
		"username":              cfg.Username,
		"password":              encPassword,
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
		"2002::/16",       // RFC 3056 6to4 (embeds IPv4; can reach private ranges)
		"2001::/32",       // RFC 4380 Teredo (tunnels IPv4 addresses)
		"64:ff9b::/96",    // RFC 6052 well-known NAT64 prefix (maps IPv4 addresses)
		"64:ff9b:1::/48",  // RFC 8215 local-use NAT64 prefix
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
			// Private/reserved resolved addresses are skipped; we only block
			// when no safe address is available, preventing the caller from
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
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
		// Block cross-host redirects to prevent sensitive request headers
		// (e.g. X-API-Key) from being forwarded to a different origin.
		// Go's default redirect policy strips Authorization on cross-host
		// redirects, but does not strip custom headers. Also block scheme
		// downgrades (e.g. HTTPS→HTTP on the same host) which would expose
		// credentials over an unencrypted connection.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) == 0 {
				return nil
			}
			prev := via[len(via)-1]
			// Block cross-origin redirects: compare effective host (Hostname +
			// default-port-normalised port) so that a redirect from
			// https://example.com/a to https://example.com:443/b (explicit
			// default port) is not incorrectly treated as cross-host.
			if !sameEffectiveHost(prev.URL, req.URL) {
				return http.ErrUseLastResponse
			}
			if strings.ToLower(prev.URL.Scheme) == "https" && strings.ToLower(req.URL.Scheme) != "https" {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
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

// sameEffectiveHost returns true when a and b refer to the same host after
// normalising default ports (80 for http, 443 for https). This avoids blocking
// same-origin redirects that explicitly include the scheme's default port, e.g.
// https://example.com/a → https://example.com:443/b.
func sameEffectiveHost(a, b *neturl.URL) bool {
	effectivePort := func(u *neturl.URL) string {
		if p := u.Port(); p != "" {
			return p
		}
		switch strings.ToLower(u.Scheme) {
		case "https":
			return "443"
		case "http":
			return "80"
		}
		return ""
	}
	return strings.EqualFold(a.Hostname(), b.Hostname()) &&
		effectivePort(a) == effectivePort(b)
}

// envOrEmpty returns os.Getenv(key).
func envOrEmpty(key string) string {
	return os.Getenv(key)
}
