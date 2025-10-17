package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"storagecore/internal/led"
)

// HighlightJobBins highlights LED bins for a specific job
func HighlightJobBins(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "job_id parameter required"})
		return
	}

	service := led.GetService()
	if err := service.HighlightJobBins(jobID); err != nil {
		log.Printf("[LED] Failed to highlight bins for job %s: %v", jobID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "LED highlight command sent",
		"job_id":  jobID,
	})
}

// ClearLEDs turns off all LEDs
func ClearLEDs(w http.ResponseWriter, r *http.Request) {
	service := led.GetService()
	if err := service.ClearAllLEDs(); err != nil {
		log.Printf("[LED] Failed to clear LEDs: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "LED clear command sent"})
}

// IdentifyLEDs sends identify command (blink all for testing)
func IdentifyLEDs(w http.ResponseWriter, r *http.Request) {
	service := led.GetService()
	if err := service.IdentifyController(); err != nil {
		log.Printf("[LED] Failed to send identify command: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "LED identify command sent"})
}

// TestBin tests a specific bin by blinking its LEDs
func TestBin(w http.ResponseWriter, r *http.Request) {
	shelfID := r.URL.Query().Get("shelf_id")
	binID := r.URL.Query().Get("bin_id")

	if shelfID == "" || binID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "shelf_id and bin_id parameters required"})
		return
	}

	service := led.GetService()
	if err := service.TestBin(shelfID, binID); err != nil {
		log.Printf("[LED] Failed to test bin %s/%s: %v", shelfID, binID, err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Bin test command sent",
		"shelf_id": shelfID,
		"bin_id":   binID,
	})
}

// GetLEDStatus returns current LED system status
func GetLEDStatus(w http.ResponseWriter, r *http.Request) {
	service := led.GetService()
	status := service.GetStatus()
	respondJSON(w, http.StatusOK, status)
}

// GetLEDMapping returns the current LED mapping configuration
func GetLEDMapping(w http.ResponseWriter, r *http.Request) {
	service := led.GetService()
	mapping, err := service.GetMapping()
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Mapping not loaded"})
		return
	}

	respondJSON(w, http.StatusOK, mapping)
}

// UpdateLEDMapping updates the LED mapping configuration
func UpdateLEDMapping(w http.ResponseWriter, r *http.Request) {
	var mapping led.LEDMapping
	if err := json.NewDecoder(r.Body).Decode(&mapping); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid mapping JSON"})
		return
	}

	// Basic validation
	if mapping.WarehouseID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id is required"})
		return
	}
	if len(mapping.Shelves) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "At least one shelf is required"})
		return
	}

	service := led.GetService()
	if err := service.UpdateMapping(&mapping); err != nil {
		log.Printf("[LED] Failed to update mapping: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Save to file
	if err := service.SaveMapping("internal/led/config/led_mapping.json"); err != nil {
		log.Printf("[LED] Failed to save mapping to file: %v", err)
		// Don't fail the request - mapping is already updated in memory
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Mapping updated successfully"})
}

// ValidateLEDMapping validates a mapping configuration without saving it
func ValidateLEDMapping(w http.ResponseWriter, r *http.Request) {
	var mapping led.LEDMapping
	if err := json.NewDecoder(r.Body).Decode(&mapping); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"valid": false,
			"error": "Invalid JSON format",
		})
		return
	}

	// Validation checks
	errors := []string{}

	if mapping.WarehouseID == "" {
		errors = append(errors, "warehouse_id is required")
	}
	if len(mapping.Shelves) == 0 {
		errors = append(errors, "At least one shelf is required")
	}
	if mapping.LEDStrip.Length <= 0 {
		errors = append(errors, "led_strip.length must be greater than 0")
	}
	if mapping.LEDStrip.DataPin < 0 {
		errors = append(errors, "led_strip.data_pin must be non-negative")
	}

	// Check for duplicate bin IDs
	binIDs := make(map[string]bool)
	for _, shelf := range mapping.Shelves {
		for _, bin := range shelf.Bins {
			if binIDs[bin.BinID] {
				errors = append(errors, "Duplicate bin_id: "+bin.BinID)
			}
			binIDs[bin.BinID] = true

			// Validate pixel indices
			for _, pixel := range bin.Pixels {
				if pixel < 0 || pixel >= mapping.LEDStrip.Length {
					errors = append(errors,
						"Invalid pixel index in bin "+bin.BinID+": out of range")
					break
				}
			}
		}
	}

	if len(errors) > 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"errors": errors,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":        true,
		"total_bins":   len(binIDs),
		"total_shelves": len(mapping.Shelves),
	})
}
