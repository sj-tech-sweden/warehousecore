package models

import "time"

// User represents a user in the system (shared with RentalCore)
type User struct {
	UserID       uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"column:username;unique;not null" json:"username"`
	Email        string     `gorm:"column:email;unique;not null" json:"email"`
	PasswordHash string     `gorm:"column:password_hash;not null" json:"-"`
	FirstName    string     `gorm:"column:first_name" json:"first_name"`
	LastName     string     `gorm:"column:last_name" json:"last_name"`
	IsAdmin      bool       `gorm:"column:is_admin;default:false" json:"is_admin"`
	IsActive     bool       `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updated_at"`
	LastLogin    *time.Time `gorm:"column:last_login" json:"last_login"`
	Roles        []Role     `json:"roles,omitempty" gorm:"-"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}

// Session represents a user session (shared with RentalCore)
type Session struct {
	SessionID string    `gorm:"column:id;primaryKey" json:"session_id"`
	UserID    uint      `gorm:"column:user_id;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null" json:"expires_at"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	User      User      `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
}

// TableName specifies the table name for Session model
func (Session) TableName() string {
	return "sessions"
}
