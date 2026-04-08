package services

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetAPILimit retrieves the configured API limit from settings
// Returns the limit value or the defaultLimit if not found or on error
func GetAPILimit(settingKey string, defaultLimit int) int {
	db := repository.GetDB()
	if db == nil {
		log.Printf("[SETTINGS] Database not available, using default limit: %d", defaultLimit)
		return defaultLimit
	}

	// Read raw JSON to tolerate legacy rows where value is a JSON number,
	// e.g. 50000, instead of an object like {"limit": 50000}.
	var row struct {
		Value json.RawMessage `gorm:"column:value"`
	}
	if err := db.Table("app_settings").
		Select("value").
		Where("scope = ? AND key = ?", "warehousecore", settingKey).
		Limit(1).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[SETTINGS] Setting %s not found, using default limit: %d", settingKey, defaultLimit)
			return defaultLimit
		}
		log.Printf("[SETTINGS] Failed to query %s: %v; using default limit: %d", settingKey, err, defaultLimit)
		return defaultLimit
	}

	var parsed interface{}
	if err := json.Unmarshal(row.Value, &parsed); err != nil {
		log.Printf("[SETTINGS] Failed to parse setting %s, using default limit: %d", settingKey, defaultLimit)
		return defaultLimit
	}

	switch v := parsed.(type) {
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	case map[string]interface{}:
		if limitVal, ok := v["limit"]; ok {
			switch lv := limitVal.(type) {
			case float64:
				return int(lv)
			case string:
				if n, err := strconv.Atoi(lv); err == nil {
					return n
				}
			}
		}
	}

	log.Printf("[SETTINGS] Invalid limit value in %s, using default limit: %d", settingKey, defaultLimit)
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
		log.Printf("[SETTINGS] Database not available, using default currency symbol")
		return "€"
	}

	// Try global scope first (shared with RentalCore), then warehousecore scope as fallback.
	for _, scope := range []string{"global", "warehousecore"} {
		var setting models.AppSetting
		if err := db.Where("scope = ? AND key = ?", scope, "app.currency").First(&setting).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("[SETTINGS] Failed to query currency symbol (scope=%s): %v", scope, err)
			}
			continue
		}

		if symbol, ok := setting.Value["symbol"].(string); ok && symbol != "" {
			return symbol
		}
	}

	log.Printf("[SETTINGS] No symbol field in currency setting, using default")
	return "€"
}

// UpdateCurrencySymbol updates the currency symbol in the database.
// Writes to scope='global' so RentalCore can also read the value.
func UpdateCurrencySymbol(symbol string) error {
	db := repository.GetDB()
	if db == nil {
		return ErrDatabaseNotAvailable
	}

	currencyValue := models.JSONMap{"symbol": symbol}

	// Write to global scope (shared with RentalCore).
	var setting models.AppSetting
	err := db.Where("scope = ? AND key = ?", "global", "app.currency").First(&setting).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = models.AppSetting{
			Scope: "global",
			Key:   "app.currency",
			Value: currencyValue,
		}
		return db.Create(&setting).Error
	} else if err != nil {
		return err
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

	limitValue := models.JSONMap{"limit": limit}
	setting := models.AppSetting{
		Scope: "warehousecore",
		Key:   settingKey,
		Value: limitValue,
	}

	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "scope"}, {Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"value":      limitValue,
			"updated_at": gorm.Expr("NOW()"),
		}),
	}).Create(&setting).Error
}

var ErrDatabaseNotAvailable = &SettingsError{Message: "Database not available"}

type SettingsError struct {
	Message string
}

func (e *SettingsError) Error() string {
	return e.Message
}
