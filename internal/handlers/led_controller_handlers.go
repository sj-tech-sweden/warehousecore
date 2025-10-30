package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"warehousecore/internal/models"
	"warehousecore/internal/services"
)

type ledControllerRequest struct {
	ControllerID string         `json:"controller_id"`
	DisplayName  string         `json:"display_name"`
	TopicSuffix  string         `json:"topic_suffix"`
	IsActive     *bool          `json:"is_active"`
	Metadata     models.JSONMap `json:"metadata"`
	ZoneTypeIDs  []int          `json:"zone_type_ids"`
}

// ListLEDControllers returns all registered controllers
func ListLEDControllers(w http.ResponseWriter, r *http.Request) {
	service := services.NewLEDControllerService()
	controllers, err := service.ListControllers()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, controllers)
}

// CreateLEDController registers a new controller
func CreateLEDController(w http.ResponseWriter, r *http.Request) {
	var payload ledControllerRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if payload.ControllerID == "" || payload.DisplayName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "controller_id and display_name are required"})
		return
	}

	topic := payload.TopicSuffix
	if topic == "" {
		topic = payload.ControllerID
	}

	controller := &models.LEDController{
		ControllerID: payload.ControllerID,
		DisplayName:  payload.DisplayName,
		TopicSuffix:  topic,
		Metadata:     payload.Metadata,
		IsActive:     true,
	}

	if payload.IsActive != nil {
		controller.IsActive = *payload.IsActive
	}

	service := services.NewLEDControllerService()
	created, err := service.CreateController(controller, payload.ZoneTypeIDs)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusCreated, created)
}

// UpdateLEDController updates controller properties
func UpdateLEDController(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid controller id"})
		return
	}

	var payload ledControllerRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	updates := map[string]interface{}{}

	if payload.DisplayName != "" {
		updates["display_name"] = payload.DisplayName
	}
	if payload.TopicSuffix != "" {
		updates["topic_suffix"] = payload.TopicSuffix
	}
	if payload.Metadata != nil {
		updates["metadata"] = payload.Metadata
	}
	if payload.IsActive != nil {
		updates["is_active"] = *payload.IsActive
	}

	service := services.NewLEDControllerService()
	updated, err := service.UpdateController(id, updates, payload.ZoneTypeIDs)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// DeleteLEDController removes a controller
func DeleteLEDController(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid controller id"})
		return
	}

	service := services.NewLEDControllerService()
	if err := service.DeleteController(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ConfigureLEDController sends configuration to an LED controller via MQTT
func ConfigureLEDController(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid controller id"})
		return
	}

	var config struct {
		LedCount *int    `json:"led_count"`
		DataPin  *int    `json:"data_pin"`
		Chipset  *string `json:"chipset"`
	}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Validate led_count if provided
	if config.LedCount != nil && (*config.LedCount < 1 || *config.LedCount > 1200) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "led_count must be between 1 and 1200"})
		return
	}

	// Validate data_pin if provided
	if config.DataPin != nil && (*config.DataPin < 0 || *config.DataPin > 50) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "data_pin must be between 0 and 50"})
		return
	}

	// At least one parameter must be provided
	if config.LedCount == nil && config.DataPin == nil && config.Chipset == nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one configuration parameter required"})
		return
	}

	service := services.NewLEDControllerService()
	if err := service.ConfigureController(id, config.LedCount, config.DataPin, config.Chipset); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response := map[string]interface{}{"status": "configuration sent"}
	if config.LedCount != nil {
		response["led_count"] = *config.LedCount
	}
	if config.DataPin != nil {
		response["data_pin"] = *config.DataPin
	}
	if config.Chipset != nil {
		response["chipset"] = *config.Chipset
	}

	respondJSON(w, http.StatusOK, response)
}

// RestartLEDController sends a restart command to an LED controller via MQTT
func RestartLEDController(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid controller id"})
		return
	}

	service := services.NewLEDControllerService()
	if err := service.RestartController(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "restart command sent",
		"message": "ESP32 will restart in 2 seconds",
	})
}
