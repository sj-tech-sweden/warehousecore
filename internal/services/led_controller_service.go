package services

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"warehousecore/internal/led"
	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

// validFirmwareTypes is the whitelist of accepted firmware_type values.
var validFirmwareTypes = map[string]bool{
	"arduino": true,
	"esphome": true,
}

// normalizeFirmwareType trims, lowercases, and validates a firmware type
// string. It returns the normalised value and true when valid, or an empty
// string and false when the input is unknown/empty.
func normalizeFirmwareType(raw string) (string, bool) {
	ft := strings.ToLower(strings.TrimSpace(raw))
	if validFirmwareTypes[ft] {
		return ft, true
	}
	return "", false
}

// LEDControllerService manages LED controller records
type LEDControllerService struct {
	db *gorm.DB
}

// NewLEDControllerService creates a new service
func NewLEDControllerService() *LEDControllerService {
	return &LEDControllerService{db: repository.GetDB()}
}

// ListControllers returns all controllers with associated zone types
func (s *LEDControllerService) ListControllers() ([]models.LEDController, error) {
	if s.db == nil {
		return nil, errors.New("database not initialised")
	}

	var controllers []models.LEDController
	if err := s.db.Preload("ZoneTypes").Order("display_name ASC").Find(&controllers).Error; err != nil {
		return nil, err
	}
	return controllers, nil
}

// GetControllerByID fetches controller by numeric ID
func (s *LEDControllerService) GetControllerByID(id int) (*models.LEDController, error) {
	if s.db == nil {
		return nil, errors.New("database not initialised")
	}

	var controller models.LEDController
	if err := s.db.Preload("ZoneTypes").First(&controller, id).Error; err != nil {
		return nil, err
	}
	return &controller, nil
}

// GetControllerByIdentifier fetches controller by controller_id
func (s *LEDControllerService) GetControllerByIdentifier(identifier string) (*models.LEDController, error) {
	if s.db == nil {
		return nil, errors.New("database not initialised")
	}

	var controller models.LEDController
	if err := s.db.Preload("ZoneTypes").Where("controller_id = ?", identifier).First(&controller).Error; err != nil {
		return nil, err
	}
	return &controller, nil
}

// CreateController creates a new controller and assigns zone types
func (s *LEDControllerService) CreateController(controller *models.LEDController, zoneTypeIDs []int) (*models.LEDController, error) {
	if s.db == nil {
		return nil, errors.New("database not initialised")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(controller).Error; err != nil {
			return err
		}

		if len(zoneTypeIDs) > 0 {
			var zoneTypes []models.ZoneType
			if err := tx.Where("id IN ?", zoneTypeIDs).Find(&zoneTypes).Error; err != nil {
				return err
			}
			if err := tx.Model(controller).Association("ZoneTypes").Replace(zoneTypes); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetControllerByID(controller.ID)
}

// UpdateController updates base properties and optionally zone types
func (s *LEDControllerService) UpdateController(id int, updates map[string]interface{}, zoneTypeIDs []int) (*models.LEDController, error) {
	if s.db == nil {
		return nil, errors.New("database not initialised")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&models.LEDController{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		if zoneTypeIDs != nil {
			var controller models.LEDController
			if err := tx.First(&controller, id).Error; err != nil {
				return err
			}
			var zoneTypes []models.ZoneType
			if len(zoneTypeIDs) > 0 {
				if err := tx.Where("id IN ?", zoneTypeIDs).Find(&zoneTypes).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&controller).Association("ZoneTypes").Replace(zoneTypes); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetControllerByID(id)
}

// DeleteController removes a controller
func (s *LEDControllerService) DeleteController(id int) error {
	if s.db == nil {
		return errors.New("database not initialised")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.LEDController{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

// RecordHeartbeat updates last_seen timestamp for controller ID and stores telemetry data
func (s *LEDControllerService) RecordHeartbeat(identifier string, payload *models.LEDControllerHeartbeat) (*models.LEDController, error) {
	if s.db == nil {
		return nil, errors.New("database not initialised")
	}

	identifier = strings.TrimSpace(identifier)
	now := time.Now()
	updates := map[string]interface{}{
		"last_seen": now,
		"is_active": true,
	}

	var status models.JSONMap
	var normalizedMac string
	var macCandidates []string
	if payload != nil {
		if payload.TopicSuffix != "" {
			updates["topic_suffix"] = payload.TopicSuffix
		}
		if payload.IPAddress != "" {
			updates["ip_address"] = payload.IPAddress
		}
		if payload.Hostname != "" {
			updates["hostname"] = payload.Hostname
		}
		if payload.FirmwareVersion != "" {
			updates["firmware_version"] = payload.FirmwareVersion
		}
		if payload.FirmwareType != "" {
			if ft, ok := normalizeFirmwareType(payload.FirmwareType); ok {
				updates["firmware_type"] = ft
			}
		}
		if payload.MacAddress != "" {
			originalMac := strings.TrimSpace(payload.MacAddress)
			normalizedMac = normalizeMACAddress(payload.MacAddress)
			if normalizedMac != "" {
				updates["mac_address"] = normalizedMac
				payload.MacAddress = normalizedMac
				macCandidates = append(macCandidates, normalizedMac)
			} else {
				updates["mac_address"] = originalMac
			}
			if originalMac != "" && (normalizedMac == "" || !strings.EqualFold(normalizedMac, originalMac)) {
				macCandidates = append(macCandidates, originalMac)
			}
		}

		status = make(models.JSONMap)
		if payload.WifiRSSI != nil {
			status["wifi_rssi"] = *payload.WifiRSSI
		}
		if payload.UptimeSeconds != nil {
			status["uptime_seconds"] = *payload.UptimeSeconds
		}
		if payload.LedCount != nil {
			status["led_count"] = *payload.LedCount
		}
		if payload.ActiveLEDs != nil {
			status["active_leds"] = *payload.ActiveLEDs
		}
		if payload.WarehouseID != "" {
			status["warehouse_id"] = payload.WarehouseID
		}
		if payload.Status != "" {
			status["status"] = payload.Status
		}
		if len(status) > 0 {
			status["heartbeat_received_at"] = now.UTC().Format(time.RFC3339)
			updates["status_data"] = status
		} else {
			status = nil
		}
	}

	if identifier != "" {
		updates["controller_id"] = identifier
	}

	result := s.db.Model(&models.LEDController{}).
		Where("controller_id = ?", identifier).
		Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 && len(macCandidates) > 0 {
		for _, candidate := range macCandidates {
			result = s.db.Model(&models.LEDController{}).
				Where("mac_address = ?", candidate).
				Updates(updates)

			if result.Error != nil {
				return nil, result.Error
			}
			if result.RowsAffected > 0 {
				break
			}
		}
	}

	if result.RowsAffected == 0 {
		controller := models.LEDController{
			ControllerID: identifier,
			DisplayName:  identifier,
			TopicSuffix:  identifier,
			IsActive:     true,
			LastSeen:     &now,
			FirmwareType: "arduino",
		}

		if payload != nil {
			if payload.TopicSuffix != "" {
				controller.TopicSuffix = payload.TopicSuffix
			}
			if payload.IPAddress != "" {
				value := payload.IPAddress
				controller.IPAddress = &value
			}
			if payload.Hostname != "" {
				value := payload.Hostname
				controller.Hostname = &value
			}
			if payload.FirmwareVersion != "" {
				value := payload.FirmwareVersion
				controller.FirmwareVersion = &value
			}
			if payload.FirmwareType != "" {
				if ft, ok := normalizeFirmwareType(payload.FirmwareType); ok {
					controller.FirmwareType = ft
				}
			}
			if payload.MacAddress != "" {
				value := payload.MacAddress
				controller.MacAddress = &value
			}
			if status != nil && len(status) > 0 {
				// Create copy to avoid shared reference
				statusCopy := make(models.JSONMap, len(status))
				for k, v := range status {
					statusCopy[k] = v
				}
				controller.StatusData = statusCopy
			}
		}

		if controller.TopicSuffix == "" {
			controller.TopicSuffix = identifier
		}

		if err := s.db.Create(&controller).Error; err != nil {
			return nil, err
		}
		return &controller, nil
	}

	return s.GetControllerByIdentifier(identifier)
}

// GetPrimaryControllerForZoneType returns the first controller assigned to the given zone type ID
func (s *LEDControllerService) GetPrimaryControllerForZoneType(zoneTypeID int) (*models.LEDController, error) {
	if s.db == nil {
		return nil, errors.New("database not initialised")
	}

	if zoneTypeID <= 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var controller models.LEDController
	err := s.db.Preload("ZoneTypes").
		Joins("JOIN led_controller_zone_types lcz ON lcz.controller_id = led_controllers.id").
		Where("lcz.zone_type_id = ?", zoneTypeID).
		Order("led_controllers.id ASC").
		First(&controller).Error

	if err != nil {
		return nil, err
	}
	return &controller, nil
}

func normalizeMACAddress(mac string) string {
	clean := strings.TrimSpace(strings.ToLower(mac))
	if clean == "" {
		return ""
	}

	replacer := strings.NewReplacer(":", "", "-", "", ".", "", " ", "")
	clean = replacer.Replace(clean)
	if len(clean) < 12 {
		return ""
	}
	if len(clean) > 12 {
		clean = clean[len(clean)-12:]
	}

	for _, r := range clean {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return ""
		}
	}

	var sb strings.Builder
	for i := 0; i < 12; i += 2 {
		if sb.Len() > 0 {
			sb.WriteByte(':')
		}
		sb.WriteString(clean[i : i+2])
	}
	return sb.String()
}

// ConfigureController sends configuration to an LED controller via MQTT
func (s *LEDControllerService) ConfigureController(id int, ledCount *int, dataPin *int, chipset *string) error {
	if s.db == nil {
		return errors.New("database not initialised")
	}

	// Get controller details
	var controller models.LEDController
	if err := s.db.First(&controller, id).Error; err != nil {
		return err
	}

	// Import LED package for MQTT publishing
	publisher := led.GetPublisher()
	if publisher == nil {
		return errors.New("MQTT publisher not available")
	}

	// Create config command
	cmd := led.LEDCommand{
		Op:          "config",
		WarehouseID: controller.TopicSuffix,
		LedCount:    ledCount,
		DataPin:     dataPin,
		Chipset:     chipset,
	}

	// Publish to controller's topic
	if err := publisher.PublishCommandToController(&controller, cmd); err != nil {
		return err
	}

	return nil
}

// RestartController sends a restart command to an LED controller via MQTT
func (s *LEDControllerService) RestartController(id int) error {
	if s.db == nil {
		return errors.New("database not initialised")
	}

	// Get controller details
	var controller models.LEDController
	if err := s.db.First(&controller, id).Error; err != nil {
		return err
	}

	// Import LED package for MQTT publishing
	publisher := led.GetPublisher()
	if publisher == nil {
		return errors.New("MQTT publisher not available")
	}

	// Create restart command
	cmd := led.LEDCommand{
		Op:          "restart",
		WarehouseID: controller.TopicSuffix,
	}

	// Publish to controller's topic
	if err := publisher.PublishCommandToController(&controller, cmd); err != nil {
		return err
	}

	return nil
}
