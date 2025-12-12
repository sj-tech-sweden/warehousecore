package services

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
)

type Revalidator struct {
	url    string
	secret string
}

func NewRevalidatorFromEnv() *Revalidator {
	url := os.Getenv("TSWEB_REVALIDATE_URL")
	secret := os.Getenv("TSWEB_REVALIDATE_SECRET")
	if url == "" || secret == "" {
		return &Revalidator{}
	}
	return &Revalidator{url: url, secret: secret}
}

// Revalidate triggers ISR revalidation for the given path (e.g., "/products").
// It is best-effort; failures are logged but not returned to the caller.
func (r *Revalidator) Revalidate(path string) {
	if r == nil || r.url == "" || r.secret == "" {
		return
	}

	payload, _ := json.Marshal(map[string]string{"path": path})
	req, err := http.NewRequest(http.MethodPost, r.url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("[REVALIDATE] build request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[REVALIDATE] call failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[REVALIDATE] non-200 response: %s", resp.Status)
	}
}

// RevalidateWithBase can be used when caller provides full URL and secret.
func RevalidateWithBase(baseURL, secret, path string) {
	if baseURL == "" || secret == "" {
		return
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	q := u.Query()
	u.RawQuery = q.Encode()

	r := &Revalidator{url: u.String(), secret: secret}
	r.Revalidate(path)
}
