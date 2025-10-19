package services

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

// AdminService handles admin-related operations
type AdminService struct {
	db *gorm.DB
}

// NewAdminService creates a new admin service
func NewAdminService() *AdminService {
	return &AdminService{
		db: repository.GetDB(),
	}
}

// ===========================
// ZONE TYPES
// ===========================

// GetAllZoneTypes returns all zone types
func (s *AdminService) GetAllZoneTypes() ([]models.ZoneType, error) {
	var zoneTypes []models.ZoneType
	err := s.db.Find(&zoneTypes).Error
	return zoneTypes, err
}

// GetZoneTypeByID returns a zone type by ID
func (s *AdminService) GetZoneTypeByID(id int) (*models.ZoneType, error) {
	var zoneType models.ZoneType
	err := s.db.First(&zoneType, id).Error
	if err != nil {
		return nil, err
	}
	return &zoneType, nil
}

// GetZoneTypeByKey returns a zone type by its key
func (s *AdminService) GetZoneTypeByKey(key string) (*models.ZoneType, error) {
	var zoneType models.ZoneType
	err := s.db.Where("key = ?", key).First(&zoneType).Error
	if err != nil {
		return nil, err
	}
	return &zoneType, nil
}

// CreateZoneType creates a new zone type
func (s *AdminService) CreateZoneType(zoneType *models.ZoneType) error {
	return s.db.Create(zoneType).Error
}

// UpdateZoneType updates an existing zone type
func (s *AdminService) UpdateZoneType(id int, updates *models.ZoneType) error {
	return s.db.Model(&models.ZoneType{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteZoneType deletes a zone type
func (s *AdminService) DeleteZoneType(id int) error {
	return s.db.Delete(&models.ZoneType{}, id).Error
}

// ===========================
// APP SETTINGS
// ===========================

// GetSetting retrieves a setting by scope and key
func (s *AdminService) GetSetting(scope, key string) (*models.AppSetting, error) {
	var setting models.AppSetting
	err := s.db.Where("scope = ? AND k = ?", scope, key).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// SetSetting creates or updates a setting
func (s *AdminService) SetSetting(scope, key string, value interface{}) error {
	// Convert value to JSON
	var jsonValue models.JSONMap
	switch v := value.(type) {
	case models.JSONMap:
		jsonValue = v
	case map[string]interface{}:
		jsonValue = models.JSONMap(v)
	default:
		// Try to marshal and unmarshal
		bytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		if err := json.Unmarshal(bytes, &jsonValue); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}
	}

	// Upsert setting
	var setting models.AppSetting
	err := s.db.Where("scope = ? AND k = ?", scope, key).First(&setting).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		setting = models.AppSetting{
			Scope: scope,
			Key:   key,
			Value: jsonValue,
		}
		return s.db.Create(&setting).Error
	} else if err != nil {
		return err
	}

	// Update existing
	setting.Value = jsonValue
	return s.db.Save(&setting).Error
}

// GetLEDSingleBinDefault retrieves LED defaults for single bin highlight
func (s *AdminService) GetLEDSingleBinDefault() (*models.LEDSingleBinDefault, error) {
	setting, err := s.GetSetting("warehousecore", "led.single_bin.default")
	if err != nil {
		// Return safe defaults if not found
		if err == gorm.ErrRecordNotFound {
			return &models.LEDSingleBinDefault{
				Color:     "#FF7A00",
				Pattern:   "breathe",
				Intensity: 180,
			}, nil
		}
		return nil, err
	}

	// Parse JSON to LEDSingleBinDefault
	var ledDefault models.LEDSingleBinDefault
	bytes, err := json.Marshal(setting.Value)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &ledDefault); err != nil {
		return nil, err
	}

	return &ledDefault, nil
}

// SetLEDSingleBinDefault updates LED defaults for single bin highlight
func (s *AdminService) SetLEDSingleBinDefault(color, pattern string, intensity uint8) error {
	value := map[string]interface{}{
		"color":     color,
		"pattern":   pattern,
		"intensity": intensity,
	}

	return s.SetSetting("warehousecore", "led.single_bin.default", value)
}

// GetLEDJobHighlightSettings retrieves highlight configuration for job packing bins
func (s *AdminService) GetLEDJobHighlightSettings() (*models.LEDJobHighlightSettings, error) {
	defaults := models.DefaultLEDJobHighlightSettings()

	setting, err := s.GetSetting("warehousecore", "led.job.highlight")
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return defaults, nil
		}
		return nil, err
	}

	var highlight models.LEDJobHighlightSettings
	bytes, err := json.Marshal(setting.Value)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &highlight); err != nil {
		return nil, err
	}

	highlight.Normalize(defaults)
	return &highlight, nil
}

// SetLEDJobHighlightSettings persists the highlight configuration for job packing bins
func (s *AdminService) SetLEDJobHighlightSettings(settings *models.LEDJobHighlightSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	settings.Normalize(models.DefaultLEDJobHighlightSettings())
	return s.SetSetting("warehousecore", "led.job.highlight", settings)
}

// ===========================
// USER PROFILES
// ===========================

// GetUserProfile retrieves a user's profile
func (s *AdminService) GetUserProfile(userID uint) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := s.db.Where("user_id = ?", userID).First(&profile).Error

	if err == gorm.ErrRecordNotFound {
		// Create default profile if not exists
		profile = models.UserProfile{
			UserID: userID,
			Prefs:  make(models.JSONMap),
		}
		if err := s.db.Create(&profile).Error; err != nil {
			return nil, err
		}
		return &profile, nil
	}

	return &profile, err
}

// UpdateUserProfile updates a user's profile
func (s *AdminService) UpdateUserProfile(userID uint, displayName, avatarURL string, prefs models.JSONMap) error {
	profile, err := s.GetUserProfile(userID)
	if err != nil {
		return err
	}

	if displayName != "" {
		profile.DisplayName = displayName
	}
	if avatarURL != "" {
		profile.AvatarURL = avatarURL
	}
	if prefs != nil {
		profile.Prefs = prefs
	}

	return s.db.Save(profile).Error
}

// GetProfileWithUser retrieves a profile with user data
func (s *AdminService) GetProfileWithUser(userID uint) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := s.db.Preload("User").Where("user_id = ?", userID).First(&profile).Error

	if err == gorm.ErrRecordNotFound {
		// Create default profile if not exists
		profile = models.UserProfile{
			UserID: userID,
			Prefs:  make(models.JSONMap),
		}
		if err := s.db.Create(&profile).Error; err != nil {
			return nil, err
		}
		// Load user relation
		if err := s.db.Preload("User").First(&profile, profile.ID).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &profile, nil
}
