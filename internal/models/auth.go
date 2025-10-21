package models

import "time"

// User represents a user in the system (shared with RentalCore)
type User struct {
	UserID       uint       `gorm:"column:userID;primaryKey;autoIncrement"`
	Username     string     `gorm:"column:username;unique;not null"`
	Email        string     `gorm:"column:email;unique;not null"`
	PasswordHash string     `gorm:"column:password_hash;not null"`
	FirstName    string     `gorm:"column:first_name"`
	LastName     string     `gorm:"column:last_name"`
	IsActive     bool       `gorm:"column:is_active;default:true"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
	LastLogin    *time.Time `gorm:"column:last_login"`
	Roles        []Role     `json:"roles,omitempty" gorm:"-"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}

// Session represents a user session (shared with RentalCore)
type Session struct {
	SessionID string    `gorm:"column:session_id;primaryKey"`
	UserID    uint      `gorm:"column:user_id;not null"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	User      User      `gorm:"foreignKey:UserID;references:UserID"`
}

// TableName specifies the table name for Session model
func (Session) TableName() string {
	return "sessions"
}
