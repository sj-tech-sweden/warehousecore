package models

import (
	"database/sql"
	"time"
)

// MovementAction represents the type of movement
type MovementAction string

const (
	ActionIntake   MovementAction = "intake"
	ActionOuttake  MovementAction = "outtake"
	ActionTransfer MovementAction = "transfer"
	ActionReturn   MovementAction = "return"
	ActionMove     MovementAction = "move"
)

// DeviceMovement represents a physical movement of a device
type DeviceMovement struct {
	MovementID  int64          `json:"movement_id" db:"movement_id"`
	DeviceID    string         `json:"device_id" db:"device_id"`
	Action      string         `json:"action" db:"action"`
	FromZoneID  sql.NullInt64  `json:"from_zone_id" db:"from_zone_id"`
	ToZoneID    sql.NullInt64  `json:"to_zone_id" db:"to_zone_id"`
	FromJobID   sql.NullInt64  `json:"from_job_id" db:"from_job_id"`
	ToJobID     sql.NullInt64  `json:"to_job_id" db:"to_job_id"`
	Barcode     sql.NullString `json:"barcode" db:"barcode"`
	UserID      sql.NullInt64  `json:"user_id" db:"user_id"`
	Notes       sql.NullString `json:"notes" db:"notes"`
	Metadata    sql.NullString `json:"metadata" db:"metadata"` // JSON field
	Timestamp   time.Time      `json:"timestamp" db:"timestamp"`
}

// MovementWithDetails includes device and location details
type MovementWithDetails struct {
	DeviceMovement
	DeviceName    string `json:"device_name,omitempty" db:"device_name"`
	ProductName   string `json:"product_name,omitempty" db:"product_name"`
	FromZoneName  string `json:"from_zone_name,omitempty" db:"from_zone_name"`
	ToZoneName    string `json:"to_zone_name,omitempty" db:"to_zone_name"`
	FromJobNumber string `json:"from_job_number,omitempty" db:"from_job_number"`
	ToJobNumber   string `json:"to_job_number,omitempty" db:"to_job_number"`
	Username      string `json:"username,omitempty" db:"username"`
}

// MovementFilter represents query filters for movements
type MovementFilter struct {
	DeviceID   string
	Action     string
	ZoneID     *int64
	JobID      *int64
	DateFrom   *time.Time
	DateTo     *time.Time
	Limit      int
	Offset     int
}
