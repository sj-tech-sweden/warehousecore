package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
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
	APIKey string `json:"api_key"`
}

// EventoryProduct represents a product from the Eventory API
type EventoryProduct struct {
	ID          interface{} `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Price       float64     `json:"price"`
}

// GetEventoryConfig retrieves the Eventory API configuration from settings
func GetEventoryConfig() (*EventoryConfig, error) {
	db := repository.GetDB()
	if db == nil {
		return nil, ErrDatabaseNotAvailable
	}

	var setting models.AppSetting
	err := db.Where("scope = ? AND key = ?", eventorySettingScope, eventorySettingKey).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &EventoryConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query eventory config: %w", err)
	}

	bytes, err := json.Marshal(setting.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal eventory config: %w", err)
	}

	var cfg EventoryConfig
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal eventory config: %w", err)
	}

	return &cfg, nil
}

// SaveEventoryConfig persists the Eventory API configuration
func SaveEventoryConfig(cfg *EventoryConfig) error {
	db := repository.GetDB()
	if db == nil {
		return ErrDatabaseNotAvailable
	}

	value := models.JSONMap{
		"api_url": cfg.APIURL,
		"api_key": cfg.APIKey,
	}

	var setting models.AppSetting
	err := db.Where("scope = ? AND key = ?", eventorySettingScope, eventorySettingKey).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = models.AppSetting{
			Scope: eventorySettingScope,
			Key:   eventorySettingKey,
			Value: value,
		}
		return db.Create(&setting).Error
	} else if err != nil {
		return fmt.Errorf("failed to query eventory config: %w", err)
	}

	setting.Value = value
	return db.Save(&setting).Error
}

// FetchEventoryProducts calls the Eventory API and returns the raw product list
// It tries to parse common response shapes that a rental product API might return.
func FetchEventoryProducts(cfg *EventoryConfig) ([]EventoryProduct, error) {
	if cfg.APIURL == "" {
		return nil, errors.New("Eventory API URL is not configured")
	}

	client := &http.Client{Timeout: 15 * time.Second}

	// Try common product endpoint paths
	endpoints := []string{
		"/api/v1/products",
		"/api/products",
		"/products",
	}

	var lastErr error
	for _, endpoint := range endpoints {
		products, err := fetchFromEndpoint(client, cfg, endpoint)
		if err == nil {
			return products, nil
		}
		lastErr = err
		log.Printf("[EVENTORY] Endpoint %s failed: %v", endpoint, err)
	}

	return nil, fmt.Errorf("all Eventory endpoints failed: %w", lastErr)
}

func fetchFromEndpoint(client *http.Client, cfg *EventoryConfig, endpoint string) ([]EventoryProduct, error) {
	url := cfg.APIURL + endpoint
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
		req.Header.Set("X-API-Key", cfg.APIKey)
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
		return nil, fmt.Errorf("unauthorized – check your API key")
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

// parseEventoryProductsResponse tries multiple common JSON shapes for a product list
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
