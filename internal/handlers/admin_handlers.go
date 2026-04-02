package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"warehousecore/internal/led"
	"warehousecore/internal/middleware"
	"warehousecore/internal/models"
	"warehousecore/internal/services"
	"warehousecore/internal/validation"

	"github.com/gorilla/mux"
)

// ===========================
// ZONE TYPES HANDLERS
// ===========================

// GetZoneTypes returns all zone types
func GetZoneTypes(w http.ResponseWriter, r *http.Request) {
	adminService := services.NewAdminService()
	zoneTypes, err := adminService.GetAllZoneTypes()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, zoneTypes)
}

// GetZoneType returns a single zone type by ID
func GetZoneType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
		return
	}

	adminService := services.NewAdminService()
	zoneType, err := adminService.GetZoneTypeByID(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Zone type not found"})
		return
	}

	respondJSON(w, http.StatusOK, zoneType)
}

// CreateZoneType creates a new zone type
func CreateZoneType(w http.ResponseWriter, r *http.Request) {
	var zoneType models.ZoneType
	if err := json.NewDecoder(r.Body).Decode(&zoneType); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if zoneType.Key == "" || zoneType.Label == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Key and label are required"})
		return
	}

	adminService := services.NewAdminService()

	// Apply safe LED defaults if none were provided in the payload
	ledDefaults, err := adminService.GetLEDSingleBinDefault()
	if err != nil || ledDefaults == nil {
		ledDefaults = &models.LEDSingleBinDefault{
			Color:     "#FF7A00",
			Pattern:   "breathe",
			Intensity: 180,
		}
	}

	if zoneType.DefaultLEDPattern == "" {
		zoneType.DefaultLEDPattern = ledDefaults.Pattern
	} else if !validation.ValidatePattern(zoneType.DefaultLEDPattern) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid LED pattern. Must be solid, breathe, or blink"})
		return
	}

	if zoneType.DefaultLEDColor == "" {
		zoneType.DefaultLEDColor = ledDefaults.Color
	} else if !validation.ValidateColorHex(zoneType.DefaultLEDColor) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid color. Use #RRGGBB or #AARRGGBB"})
		return
	}

	if zoneType.DefaultIntensity == 0 {
		zoneType.DefaultIntensity = ledDefaults.Intensity
	} else if zoneType.DefaultIntensity > 255 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid intensity. Must be 0-255"})
		return
	}

	if err := adminService.CreateZoneType(&zoneType); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, zoneType)
}

// UpdateZoneType updates an existing zone type
func UpdateZoneType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
		return
	}

	var updates models.ZoneType
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate LED defaults if provided
	if updates.DefaultLEDPattern != "" && !validation.ValidatePattern(updates.DefaultLEDPattern) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid LED pattern"})
		return
	}
	if updates.DefaultLEDColor != "" && !validation.ValidateColorHex(updates.DefaultLEDColor) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid color. Use #RRGGBB or #AARRGGBB"})
		return
	}
	if updates.DefaultIntensity > 255 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid intensity. Must be 0-255"})
		return
	}

	adminService := services.NewAdminService()
	if err := adminService.UpdateZoneType(id, &updates); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Fetch and return updated zone type
	zoneType, _ := adminService.GetZoneTypeByID(id)
	respondJSON(w, http.StatusOK, zoneType)
}

// DeleteZoneType deletes a zone type
func DeleteZoneType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
		return
	}

	adminService := services.NewAdminService()
	if err := adminService.DeleteZoneType(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Zone type deleted successfully"})
}

// ===========================
// LED DEFAULTS HANDLERS
// ===========================

// GetLEDSingleBinDefault returns LED defaults for single bin highlight
func GetLEDSingleBinDefault(w http.ResponseWriter, r *http.Request) {
	adminService := services.NewAdminService()
	defaults, err := adminService.GetLEDSingleBinDefault()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, defaults)
}

// UpdateLEDSingleBinDefault updates LED defaults for single bin highlight
func UpdateLEDSingleBinDefault(w http.ResponseWriter, r *http.Request) {
	var payload models.LEDSingleBinDefault
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate pattern
	if !validation.ValidatePattern(payload.Pattern) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid pattern. Must be solid, breathe, or blink"})
		return
	}

	// Validate color
	if !validation.ValidateColorHex(payload.Color) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid color. Use #RRGGBB or #AARRGGBB"})
		return
	}

	// Validate intensity
	if payload.Intensity > 255 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid intensity. Must be 0-255"})
		return
	}

	adminService := services.NewAdminService()
	if err := adminService.SetLEDSingleBinDefault(payload.Color, payload.Pattern, payload.Intensity); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, payload)
}

// PreviewLEDSettings publishes a temporary highlight using the provided appearances
func PreviewLEDSettings(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Appearances []models.LEDAppearance `json:"appearances"`
		ClearBefore bool                   `json:"clear_before"`
		TargetBinID string                 `json:"target_bin_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if len(payload.Appearances) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "At least one appearance is required"})
		return
	}

	for _, appearance := range payload.Appearances {
		if !validation.ValidateColorHex(appearance.Color) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid color in preview request"})
			return
		}
		if !validation.ValidatePattern(appearance.Pattern) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid pattern in preview request"})
			return
		}
		if appearance.Intensity > 255 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid intensity in preview request"})
			return
		}
		if appearance.Speed < 0 || appearance.Speed > 10000 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid speed in preview request"})
			return
		}
	}

	service := led.GetService()
	if err := service.PreviewAppearances(payload.Appearances, payload.ClearBefore, payload.TargetBinID); err != nil {
		log.Printf("[LED] Failed to run preview: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Preview command sent"})
}

// GetLEDJobHighlightSettings returns the current highlight configuration for job packing
func GetLEDJobHighlightSettings(w http.ResponseWriter, r *http.Request) {
	adminService := services.NewAdminService()
	settings, err := adminService.GetLEDJobHighlightSettings()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// UpdateLEDJobHighlightSettings updates how bins are highlighted when preparing jobs
func UpdateLEDJobHighlightSettings(w http.ResponseWriter, r *http.Request) {
	var payload models.LEDJobHighlightSettings
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Normalize with defaults and validate
	defaults := models.DefaultLEDJobHighlightSettings()
	payload.Normalize(defaults)

	if payload.Mode != "all_bins" && payload.Mode != "required_only" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid mode. Must be all_bins or required_only"})
		return
	}

	for _, appearance := range []models.LEDAppearance{payload.Required, payload.NonRequired} {
		if !validation.ValidateColorHex(appearance.Color) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid color. Use #RRGGBB format"})
			return
		}
		if !validation.ValidatePattern(appearance.Pattern) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid pattern. Must be solid, breathe, or blink"})
			return
		}
		if appearance.Intensity > 255 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid intensity. Must be 0-255"})
			return
		}
		if appearance.Speed < 0 || appearance.Speed > 10000 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid speed. Must be between 0 and 10000 milliseconds"})
			return
		}
	}

	adminService := services.NewAdminService()
	if err := adminService.SetLEDJobHighlightSettings(&payload); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, payload)
}

// ===========================
// ROLES HANDLERS
// ===========================

// GetRoles returns all available roles
func GetRoles(w http.ResponseWriter, r *http.Request) {
	rbacService := services.NewRBACService()
	roles, err := rbacService.GetAllRoles()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, roles)
}

// GetUsersWithRoles returns all users with their assigned roles
func GetUsersWithRoles(w http.ResponseWriter, r *http.Request) {
	rbacService := services.NewRBACService()
	users, err := rbacService.GetUsersWithRoles()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Shape response for frontend: keys match expected casing
	type RoleDTO struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	type UserDTO struct {
		UserID    uint      `json:"userID"`
		Username  string    `json:"username"`
		Email     string    `json:"email"`
		FirstName string    `json:"first_name"`
		LastName  string    `json:"last_name"`
		Roles     []RoleDTO `json:"roles"`
	}

	out := make([]UserDTO, 0, len(users))
	for _, u := range users {
		roles := make([]RoleDTO, 0, len(u.Roles))
		for _, rle := range u.Roles {
			roles = append(roles, RoleDTO{ID: rle.ID, Name: rle.Name, Description: rle.Description})
		}
		out = append(out, UserDTO{
			UserID:    u.User.UserID,
			Username:  u.User.Username,
			Email:     u.User.Email,
			FirstName: u.User.FirstName,
			LastName:  u.User.LastName,
			Roles:     roles,
		})
	}

	respondJSON(w, http.StatusOK, out)
}

// GetUserRoles returns roles for a specific user
func GetUserRoles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
		return
	}

	rbacService := services.NewRBACService()
	roles, err := rbacService.GetUserRoles(uint(userID))
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, roles)
}

// UpdateUserRoles updates roles for a specific user
func UpdateUserRoles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
		return
	}

	var payload struct {
		RoleIDs []int `json:"role_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	currentUser, ok := middleware.GetUserFromContext(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	rbacService := services.NewRBACService()

	currentRoles := currentUser.Roles
	if len(currentRoles) == 0 {
		var loadErr error
		currentRoles, loadErr = rbacService.GetUserRoles(currentUser.UserID)
		if loadErr != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": loadErr.Error()})
			return
		}
	}

	allRoles, err := rbacService.GetAllRoles()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	existingRoles, err := rbacService.GetUserRoles(uint(userID))
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Build helper lookups
	restrictedNames := map[string]struct{}{
		"admin":       {},
		"manager":     {},
		"super_admin": {},
	}
	restrictedIDs := make(map[int]struct{})
	for _, role := range allRoles {
		if _, restricted := restrictedNames[strings.ToLower(role.Name)]; restricted {
			restrictedIDs[role.ID] = struct{}{}
		}
	}

	existingSet := make(map[int]struct{}, len(existingRoles))
	for _, role := range existingRoles {
		existingSet[role.ID] = struct{}{}
	}

	payloadSet := make(map[int]struct{}, len(payload.RoleIDs))
	for _, id := range payload.RoleIDs {
		payloadSet[id] = struct{}{}
	}

	hasAdminPrivilege := false
	for _, role := range currentRoles {
		if strings.EqualFold(role.Name, "admin") || strings.EqualFold(role.Name, "super_admin") {
			hasAdminPrivilege = true
			break
		}
	}

	if !hasAdminPrivilege {
		for restrictedID := range restrictedIDs {
			_, hadRole := existingSet[restrictedID]
			_, wantsRole := payloadSet[restrictedID]
			if hadRole != wantsRole {
				respondJSON(w, http.StatusForbidden, map[string]string{
					"error": "Insufficient permissions to modify elevated roles",
				})
				return
			}
		}
	}

	if err := rbacService.SetUserRoles(uint(userID), payload.RoleIDs); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Return updated roles
	roles, _ := rbacService.GetUserRoles(uint(userID))
	respondJSON(w, http.StatusOK, roles)
}

// ===========================
// PROFILE HANDLERS
// ===========================

// GetMyProfile returns the current user's profile
func GetMyProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	adminService := services.NewAdminService()
	rbacService := services.NewRBACService()

	profile, err := adminService.GetProfileWithUser(user.UserID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Add user roles to response
	roles, err := rbacService.GetUserRoles(user.UserID)
	if err != nil {
		log.Printf("GetMyProfile: Error getting user roles for user %d: %v", user.UserID, err)
	}

	// Build DTO matching frontend expectations
	type RoleDTO struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	type UserDTO struct {
		UserID    uint   `json:"userID"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	userDTO := UserDTO{
		UserID:    profile.User.UserID,
		Username:  profile.User.Username,
		Email:     profile.User.Email,
		FirstName: profile.User.FirstName,
		LastName:  profile.User.LastName,
	}
	rolesDTO := make([]RoleDTO, 0, len(roles))
	for _, rle := range roles {
		rolesDTO = append(rolesDTO, RoleDTO{ID: rle.ID, Name: rle.Name, Description: rle.Description})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"profile": map[string]interface{}{
			"id":           profile.ID,
			"user_id":      profile.UserID,
			"display_name": profile.DisplayName,
			"avatar_url":   profile.AvatarURL,
			"prefs":        profile.Prefs,
			"user":         userDTO,
		},
		"roles": rolesDTO,
	})
}

// UpdateMyProfile updates the current user's profile
func UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	var payload struct {
		DisplayName string         `json:"display_name"`
		AvatarURL   string         `json:"avatar_url"`
		Prefs       models.JSONMap `json:"prefs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	adminService := services.NewAdminService()
	if err := adminService.UpdateUserProfile(user.UserID, payload.DisplayName, payload.AvatarURL, payload.Prefs); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Return updated profile
	profile, _ := adminService.GetProfileWithUser(user.UserID)
	respondJSON(w, http.StatusOK, profile)
}

// ===========================
// API LIMITS HANDLERS
// ===========================

// GetAPILimits returns the configured API limits for devices and cases
func GetAPILimits(w http.ResponseWriter, r *http.Request) {
	deviceLimit := services.GetDeviceLimit()
	caseLimit := services.GetCaseLimit()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"device_limit": deviceLimit,
		"case_limit":   caseLimit,
	})
}

// UpdateAPILimits updates the API limits for devices and cases
func UpdateAPILimits(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		DeviceLimit *int `json:"device_limit"`
		CaseLimit   *int `json:"case_limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate limits
	if payload.DeviceLimit != nil && *payload.DeviceLimit < 1 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Device limit must be at least 1"})
		return
	}
	if payload.CaseLimit != nil && *payload.CaseLimit < 1 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Case limit must be at least 1"})
		return
	}

	// Update device limit if provided
	if payload.DeviceLimit != nil {
		if err := services.UpdateAPILimit("api.device_limit", *payload.DeviceLimit); err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update device limit"})
			return
		}
	}

	// Update case limit if provided
	if payload.CaseLimit != nil {
		if err := services.UpdateAPILimit("api.case_limit", *payload.CaseLimit); err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update case limit"})
			return
		}
	}

	// Return updated limits
	deviceLimit := services.GetDeviceLimit()
	caseLimit := services.GetCaseLimit()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"device_limit": deviceLimit,
		"case_limit":   caseLimit,
		"message":      "API limits updated successfully",
	})
}

// ===========================
// CURRENCY SETTINGS HANDLERS
// ===========================

// GetCurrencySettings returns the configured currency symbol
func GetCurrencySettings(w http.ResponseWriter, r *http.Request) {
	symbol := services.GetCurrencySymbol()
	respondJSON(w, http.StatusOK, map[string]string{
		"currency_symbol": symbol,
	})
}

// UpdateCurrencySettings updates the currency symbol
func UpdateCurrencySettings(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CurrencySymbol string `json:"currency_symbol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if strings.TrimSpace(payload.CurrencySymbol) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Currency symbol cannot be empty"})
		return
	}

	if len([]rune(strings.TrimSpace(payload.CurrencySymbol))) > 8 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Currency symbol must be 8 characters or fewer"})
		return
	}

	symbol := strings.TrimSpace(payload.CurrencySymbol)
	if err := services.UpdateCurrencySymbol(symbol); err != nil {
		log.Printf("[CURRENCY] Failed to update currency symbol: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update currency symbol"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"currency_symbol": symbol,
		"message":         "Currency symbol updated successfully",
	})
}
