package services

import (
	"encoding/json"
	"log"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

// GetAPILimit retrieves the configured API limit from settings
// Returns the limit value or the defaultLimit if not found or on error
func GetAPILimit(settingKey string, defaultLimit int) int {
	db := repository.GetDB()
	if db == nil {
		log.Printf("[SETTINGS] Database not available, using default limit: %d", defaultLimit)
		return defaultLimit
	}

	var setting models.AppSetting
	if err := db.Where("scope = ? AND key = ?", "warehousecore", settingKey).First(&setting).Error; err != nil {
		log.Printf("[SETTINGS] Setting %s not found, using default limit: %d", settingKey, defaultLimit)
		return defaultLimit
	}

	// Parse JSON to get limit value
	bytes, err := json.Marshal(setting.Value)
	if err != nil {
		log.Printf("[SETTINGS] Failed to marshal setting %s, using default limit: %d", settingKey, defaultLimit)
		return defaultLimit
	}

	var limitConfig map[string]interface{}
	if err := json.Unmarshal(bytes, &limitConfig); err != nil {
		log.Printf("[SETTINGS] Failed to unmarshal setting %s, using default limit: %d", settingKey, defaultLimit)
		return defaultLimit
	}

	// Extract limit value
	if limitVal, ok := limitConfig["limit"]; ok {
		switch v := limitVal.(type) {
		case float64:
			return int(v)
		case int:
			return v
		default:
			log.Printf("[SETTINGS] Invalid limit type in %s, using default limit: %d", settingKey, defaultLimit)
			return defaultLimit
		}
	}

	log.Printf("[SETTINGS] No limit field in %s, using default limit: %d", settingKey, defaultLimit)
	return defaultLimit
}

// GetDeviceLimit retrieves the configured device API limit
func GetDeviceLimit() int {
	return GetAPILimit("api.device_limit", 50000)
}

// GetCaseLimit retrieves the configured case API limit
func GetCaseLimit() int {
	return GetAPILimit("api.case_limit", 50000)
}

// GetCurrencySymbol retrieves the configured currency symbol from settings
// Returns "€" as the default if not configured
func GetCurrencySymbol() string {
	db := repository.GetDB()
	if db == nil {
		return "€"
	}

	var setting models.AppSetting
	if err := db.Where("scope = ? AND key = ?", "warehousecore", "app.currency").First(&setting).Error; err != nil {
		return "€"
	}

	bytes, err := json.Marshal(setting.Value)
	if err != nil {
		return "€"
	}

	var currencyConfig map[string]interface{}
	if err := json.Unmarshal(bytes, &currencyConfig); err != nil {
		return "€"
	}

	if symbol, ok := currencyConfig["symbol"].(string); ok && symbol != "" {
		return symbol
	}

	return "€"
}

// UpdateCurrencySymbol updates the currency symbol in the database
func UpdateCurrencySymbol(symbol string) error {
	db := repository.GetDB()
	if db == nil {
		return ErrDatabaseNotAvailable
	}

	var setting models.AppSetting
	err := db.Where("scope = ? AND key = ?", "warehousecore", "app.currency").First(&setting).Error

	currencyValue := models.JSONMap{"symbol": symbol}

	if err != nil {
		setting = models.AppSetting{
			Scope: "warehousecore",
			Key:   "app.currency",
			Value: currencyValue,
		}
		return db.Create(&setting).Error
	}

	setting.Value = currencyValue
	return db.Save(&setting).Error
}

// UpdateAPILimit updates the API limit setting in the database
func UpdateAPILimit(settingKey string, limit int) error {
	db := repository.GetDB()
	if db == nil {
		return ErrDatabaseNotAvailable
	}

	// Check if setting exists
	var setting models.AppSetting
	err := db.Where("scope = ? AND key = ?", "warehousecore", settingKey).First(&setting).Error

	limitValue := models.JSONMap{"limit": limit}

	if err != nil {
		// Create new setting
		setting = models.AppSetting{
			Scope: "warehousecore",
			Key:   settingKey,
			Value: limitValue,
		}
		return db.Create(&setting).Error
	}

	// Update existing setting
	setting.Value = limitValue
	return db.Save(&setting).Error
}

var ErrDatabaseNotAvailable = &SettingsError{Message: "Database not available"}

type SettingsError struct {
	Message string
}

func (e *SettingsError) Error() string {
	return e.Message
}
