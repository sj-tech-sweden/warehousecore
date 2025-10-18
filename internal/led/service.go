package led

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"warehousecore/internal/repository"
)

// Service handles LED-related business logic
type Service struct {
	mapping   *LEDMapping
	publisher *Publisher
	mu        sync.RWMutex
}

var (
	serviceInstance *Service
	serviceOnce     sync.Once
)

// GetService returns the singleton LED service instance
func GetService() *Service {
	serviceOnce.Do(func() {
		serviceInstance = NewService()
	})
	return serviceInstance
}

// NewService creates a new LED service
func NewService() *Service {
	s := &Service{
		publisher: GetPublisher(),
	}

	// Load mapping configuration
	if err := s.LoadMapping("internal/led/config/led_mapping.json"); err != nil {
		log.Printf("[LED] Failed to load mapping: %v", err)
	}

	return s
}

// LoadMapping loads the LED mapping configuration from file
func (s *Service) LoadMapping(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read mapping file: %w", err)
	}

	var mapping LEDMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return fmt.Errorf("failed to parse mapping JSON: %w", err)
	}

	s.mapping = &mapping
	log.Printf("[LED] Loaded mapping: %d shelves, %d total bins", len(mapping.Shelves), s.countTotalBins())
	return nil
}

// SaveMapping saves the LED mapping configuration to file
func (s *Service) SaveMapping(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.mapping == nil {
		return fmt.Errorf("no mapping loaded")
	}

	data, err := json.MarshalIndent(s.mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write mapping file: %w", err)
	}

	return nil
}

// GetMapping returns a copy of the current mapping
func (s *Service) GetMapping() (*LEDMapping, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.mapping == nil {
		return nil, fmt.Errorf("no mapping loaded")
	}

	// Return a copy to prevent external modifications
	mappingCopy := *s.mapping
	return &mappingCopy, nil
}

// UpdateMapping updates the mapping configuration
func (s *Service) UpdateMapping(mapping *LEDMapping) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mapping = mapping
	log.Printf("[LED] Mapping updated: %d shelves, %d bins", len(mapping.Shelves), s.countTotalBins())
	return nil
}

// HighlightJobBins highlights bins for devices in a specific job
func (s *Service) HighlightJobBins(jobID string) error {
	// Get devices for this job from database
	deviceZones, err := s.getJobDeviceZones(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job devices: %w", err)
	}

	if len(deviceZones) == 0 {
		return fmt.Errorf("no devices found for job %s", jobID)
	}

	s.mu.RLock()
	mapping := s.mapping
	s.mu.RUnlock()

	return s.publisher.PublishHighlight(jobID, mapping, deviceZones)
}

// ClearAllLEDs turns off all LEDs
func (s *Service) ClearAllLEDs() error {
	return s.publisher.PublishClear()
}

// IdentifyController sends identify command to test LEDs
func (s *Service) IdentifyController() error {
	return s.publisher.PublishIdentify()
}

// TestBin sends a test command for a specific bin
func (s *Service) TestBin(shelfID, binID string) error {
	s.mu.RLock()
	mapping := s.mapping
	s.mu.RUnlock()

	if mapping == nil {
		return fmt.Errorf("no mapping loaded")
	}

	// Find bin in mapping
	var pixels []int
	found := false
	for _, shelf := range mapping.Shelves {
		if shelf.ShelfID == shelfID {
			for _, bin := range shelf.Bins {
				if bin.BinID == binID {
					pixels = bin.Pixels
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("bin %s not found in shelf %s", binID, shelfID)
	}

	// Create test command (blink pattern)
	cmd := LEDCommand{
		Op:          "highlight",
		WarehouseID: mapping.WarehouseID,
		Shelves: []Shelf{
			{
				ShelfID: shelfID,
				Bins: []Bin{
					{
						BinID:     binID,
						Pixels:    pixels,
						Color:     "#00FF00", // Green for test
						Pattern:   "blink",
						Intensity: 255,
					},
				},
			},
		},
	}

	return s.publisher.PublishCommand(cmd)
}

// GetStatus returns the current LED system status
func (s *Service) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"mqtt_connected":  s.publisher.IsConnected(),
		"mqtt_dry_run":    s.publisher.IsDryRun(),
		"mapping_loaded":  s.mapping != nil,
		"warehouse_id":    "",
		"total_shelves":   0,
		"total_bins":      0,
	}

	if s.mapping != nil {
		status["warehouse_id"] = s.mapping.WarehouseID
		status["total_shelves"] = len(s.mapping.Shelves)
		status["total_bins"] = s.countTotalBins()
	}

	return status
}

// getJobDeviceZones retrieves device zone codes for a job from database
// Returns a map of deviceID -> zone code (e.g., "WDL-06-RG-01-F-01")
func (s *Service) getJobDeviceZones(jobID string) (map[string]string, error) {
	db := repository.GetSQLDB()

	// Query job devices with their zone CODE (not name!)
	// The zone code matches the bin_id in the LED mapping configuration
	query := `
		SELECT jd.deviceID, COALESCE(z.code, '') as zone_code
		FROM jobdevices jd
		LEFT JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN storage_zones z ON d.zone_id = z.zone_id
		WHERE jd.jobID = ? AND d.status = 'in_storage' AND z.code IS NOT NULL
	`

	rows, err := db.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deviceZones := make(map[string]string)
	for rows.Next() {
		var deviceID, zoneCode string
		if err := rows.Scan(&deviceID, &zoneCode); err != nil {
			log.Printf("[LED] Error scanning device row: %v", err)
			continue
		}
		deviceZones[deviceID] = zoneCode
		log.Printf("[LED] Device %s is in zone %s", deviceID, zoneCode)
	}

	return deviceZones, nil
}

// countTotalBins counts total bins in current mapping (must be called with lock held)
func (s *Service) countTotalBins() int {
	if s.mapping == nil {
		return 0
	}
	count := 0
	for _, shelf := range s.mapping.Shelves {
		count += len(shelf.Bins)
	}
	return count
}

// Close cleanup resources
func (s *Service) Close() {
	if s.publisher != nil {
		s.publisher.Close()
	}
}
