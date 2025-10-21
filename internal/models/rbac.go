package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Role represents a system role for RBAC (uses RentalCore schema)
type Role struct {
	ID          int       `json:"id" gorm:"column:roleID;primaryKey"`
	Name        string    `json:"name" gorm:"column:name;not null"`
	DisplayName string    `json:"display_name" gorm:"column:display_name"`
	Description string    `json:"description" gorm:"column:description"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName specifies the table name for Role model
func (Role) TableName() string {
	return "roles"
}

// UserRole represents a user-to-role assignment (RentalCore schema)
type UserRole struct {
	UserID     uint      `json:"user_id" gorm:"column:userID;primaryKey"`
	RoleID     int       `json:"role_id" gorm:"column:roleID;primaryKey"`
	AssignedAt time.Time `json:"assigned_at" gorm:"column:assigned_at;autoCreateTime"`
	IsActive   bool      `json:"is_active" gorm:"column:is_active"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:UserID"`
	Role Role `json:"role,omitempty" gorm:"foreignKey:RoleID;references:ID"`
}

// TableName specifies the table name for UserRole model
func (UserRole) TableName() string {
	return "user_roles"
}

// ZoneType represents a configurable zone type with LED defaults
type ZoneType struct {
	ID                int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Key               string    `json:"key" gorm:"column:key;unique;not null"`
	Label             string    `json:"label" gorm:"column:label;not null"`
	Description       string    `json:"description" gorm:"column:description"`
	DefaultLEDPattern string    `json:"default_led_pattern" gorm:"column:default_led_pattern;type:enum('solid','breathe','blink')"`
	DefaultLEDColor   string    `json:"default_led_color" gorm:"column:default_led_color"`
	DefaultIntensity  uint8     `json:"default_intensity" gorm:"column:default_intensity"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName specifies the table name for ZoneType model
func (ZoneType) TableName() string {
	return "zone_types"
}

// JSONMap is a custom type for JSON storage
type JSONMap map[string]interface{}

// Value implements driver.Valuer interface for GORM
func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface for GORM
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// AppSetting represents a global application setting
type AppSetting struct {
	ID        int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Scope     string    `json:"scope" gorm:"column:scope;type:enum('global','warehousecore');not null"`
	Key       string    `json:"key" gorm:"column:k;not null"`
	Value     JSONMap   `json:"value" gorm:"column:v;type:json;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName specifies the table name for AppSetting model
func (AppSetting) TableName() string {
	return "app_settings"
}

// UserProfile represents WarehouseCore-specific user profile data
type UserProfile struct {
	ID          int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID      uint      `json:"user_id" gorm:"column:user_id;unique;not null"`
	DisplayName string    `json:"display_name" gorm:"column:display_name"`
	AvatarURL   string    `json:"avatar_url" gorm:"column:avatar_url"`
	Prefs       JSONMap   `json:"prefs" gorm:"column:prefs;type:json"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`

	// Relationship
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:UserID"`
}

// TableName specifies the table name for UserProfile model
func (UserProfile) TableName() string {
	return "user_profiles"
}

// LEDSingleBinDefault represents LED default settings for single bin highlight
type LEDSingleBinDefault struct {
	Color     string `json:"color"`
	Pattern   string `json:"pattern"`
	Intensity uint8  `json:"intensity"`
}

// UserWithRoles represents a user with their assigned roles
type UserWithRoles struct {
	User
	Roles []Role `json:"roles"`
}
