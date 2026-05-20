package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"
)

// RBACService handles role-based access control operations
type RBACService struct {
	db *gorm.DB
}

// NewRBACService creates a new RBAC service
func NewRBACService() *RBACService {
	return &RBACService{
		db: repository.GetDB(),
	}
}

// GetUserRoles returns all roles for a specific user
func (s *RBACService) GetUserRoles(userID uint) ([]models.Role, error) {
	var roles []models.Role
	err := s.db.Table("roles").
		Joins("JOIN user_roles ON user_roles.roleid = roles.roleid").
		Where("user_roles.userid = ?", userID).
		Find(&roles).Error

	return roles, err
}

// HasRole checks if a user has a specific role
func (s *RBACService) HasRole(userID uint, roleName string) (bool, error) {
	var count int64
	err := s.db.Table("user_roles").
		Joins("JOIN roles ON roles.roleid = user_roles.roleid").
		Where("user_roles.userid = ? AND roles.name = ?", userID, roleName).
		Count(&count).Error

	return count > 0, err
}

// HasAnyRole checks if a user has any of the specified roles
func (s *RBACService) HasAnyRole(userID uint, roleNames []string) (bool, error) {
	var count int64
	err := s.db.Table("user_roles").
		Joins("JOIN roles ON roles.roleid = user_roles.roleid").
		Where("user_roles.userid = ? AND roles.name IN ?", userID, roleNames).
		Count(&count).Error

	return count > 0, err
}

// AssignRole assigns a role to a user
func (s *RBACService) AssignRole(userID uint, roleID int) error {
	userRole := models.UserRole{UserID: userID, RoleID: roleID, AssignedAt: time.Now().UTC()}
	return s.db.Create(&userRole).Error
}

// RemoveRole removes a role from a user
func (s *RBACService) RemoveRole(userID uint, roleID int) error {
	return s.db.Where("userid = ? AND roleid = ?", userID, roleID).
		Delete(&models.UserRole{}).Error
}

// SetUserRoles replaces all user roles with the provided list
func (s *RBACService) SetUserRoles(userID uint, roleIDs []int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing roles
		if err := tx.Where("userid = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}

		// Insert new roles
		for _, roleID := range roleIDs {
			userRole := models.UserRole{UserID: userID, RoleID: roleID}
			if err := tx.Create(&userRole).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetAllRoles returns all available roles
func (s *RBACService) GetAllRoles() ([]models.Role, error) {
	var roles []models.Role
	err := s.db.Find(&roles).Error
	return roles, err
}

// GetRoleByName returns a role by its name
func (s *RBACService) GetRoleByName(name string) (*models.Role, error) {
	var role models.Role
	err := s.db.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetUsersWithRoles returns all users with their roles
func (s *RBACService) GetUsersWithRoles() ([]models.UserWithRoles, error) {
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}

	result := make([]models.UserWithRoles, len(users))
	for i, user := range users {
		roles, err := s.GetUserRoles(user.UserID)
		if err != nil {
			log.Printf("Error getting roles for user %d: %v", user.UserID, err)
			roles = []models.Role{}
		}
		result[i] = models.UserWithRoles{
			User:  user,
			Roles: roles,
		}
	}

	return result, nil
}

// CreateRole creates a new role
func (s *RBACService) CreateRole(name, description string, permissions []string) (*models.Role, error) {
	var perms json.RawMessage
	if permissions != nil {
		b, err := json.Marshal(permissions)
		if err != nil {
			return nil, err
		}
		perms = b
	}

	role := models.Role{
		Name:        name,
		Description: description,
		Permissions: perms,
	}

	if err := s.db.Create(&role).Error; err != nil {
		return nil, err
	}

	return &role, nil
}

// UpdateRole updates an existing role.
func (s *RBACService) UpdateRole(roleID int, name, description string, permissions []string) (*models.Role, error) {
	var role models.Role
	if err := s.db.Where("roleid = ?", roleID).First(&role).Error; err != nil {
		return nil, err
	}

	if strings.TrimSpace(name) != "" {
		role.Name = strings.TrimSpace(name)
	}
	role.Description = strings.TrimSpace(description)

	if permissions != nil {
		b, err := json.Marshal(permissions)
		if err != nil {
			return nil, err
		}
		role.Permissions = b
	}

	if err := s.db.Save(&role).Error; err != nil {
		return nil, err
	}

	return &role, nil
}

// EnsureAdminForUser ensures a specific user has admin rights
// This is used for auto-admin assignment (e.g., for "N. Thielmann")
func (s *RBACService) EnsureAdminForUser(userID uint) error {
	adminRole, err := s.GetRoleByName("admin")
	if err != nil {
		return fmt.Errorf("admin role not found: %w", err)
	}

	hasAdmin, err := s.HasRole(userID, "admin")
	if err != nil {
		return err
	}

	if !hasAdmin {
		log.Printf("[RBAC] Assigning admin role to user ID %d", userID)
		return s.AssignRole(userID, adminRole.ID)
	}

	log.Printf("[RBAC] User ID %d already has admin role", userID)
	return nil
}

// FindUserByNamePattern finds a user by name pattern (for auto-admin)
func (s *RBACService) FindUserByNamePattern(pattern string) (*models.User, error) {
	var user models.User

	// Try full name (first_name + last_name)
	err := s.db.Raw(`
		SELECT * FROM users
		WHERE CONCAT(first_name, ' ', last_name) LIKE ?
		   OR username LIKE ?
		   OR email LIKE ?
		LIMIT 1
	`, "%"+pattern+"%", "%"+pattern+"%", "%"+pattern+"%").Scan(&user).Error

	if err != nil {
		return nil, err
	}

	if user.UserID == 0 {
		return nil, fmt.Errorf("user not found with pattern: %s", pattern)
	}

	return &user, nil
}

// EnsureAutoAdminFromEnv assigns admin role to the user matching ADMIN_NAME_MATCH (default: "N. Thielmann")
func (s *RBACService) EnsureAutoAdminFromEnv() error {
	match := os.Getenv("ADMIN_NAME_MATCH")
	if match == "" {
		match = "N. Thielmann"
	}
	user, err := s.FindUserByNamePattern(match)
	if err != nil {
		log.Printf("[RBAC] Auto-admin: no user matching %q found: %v", match, err)
		return nil
	}
	return s.EnsureAdminForUser(user.UserID)
}

// EnsureDefaultAdminFromEnv will create a default admin user if no users exist.
// It reads ADMIN_USERNAME, ADMIN_EMAIL, ADMIN_PASSWORD, ADMIN_FIRSTNAME, ADMIN_LASTNAME
// from the environment. ADMIN_PASSWORD must be explicitly provided. The created user
// is assigned the `super_admin` role if present, otherwise `admin` role.
func (s *RBACService) EnsureDefaultAdminFromEnv() error {
	// Only seed if no users exist
	var count int64
	if err := s.db.Table("users").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@example.local"
	}
	pw := os.Getenv("ADMIN_PASSWORD")
	if pw == "" {
		return fmt.Errorf("ADMIN_PASSWORD is required to seed default admin user")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Build an INSERT with only the columns that actually exist in the `users` table
	var cols []string
	var placeholders []string
	var args []interface{}

	// required columns
	cols = append(cols, "username", "email", "password_hash", "is_active")
	placeholders = append(placeholders, "?", "?", "?", "?")
	args = append(args, username, email, string(hash), true)

	// optional columns
	// first_name
	exists := s.userColumnExists("first_name")
	if exists == 1 {
		cols = append(cols, "first_name")
		placeholders = append(placeholders, "?")
		args = append(args, os.Getenv("ADMIN_FIRSTNAME"))
	}

	// last_name
	exists = s.userColumnExists("last_name")
	if exists == 1 {
		cols = append(cols, "last_name")
		placeholders = append(placeholders, "?")
		args = append(args, os.Getenv("ADMIN_LASTNAME"))
	}

	// force_password_change
	exists = s.userColumnExists("force_password_change")
	if exists == 1 {
		cols = append(cols, "force_password_change")
		placeholders = append(placeholders, "?")
		args = append(args, false)
	}

	// created_at / updated_at may exist, but DB has defaults; skip unless needed

	// Build SQL
	colList := ""
	phList := ""
	for i, c := range cols {
		if i > 0 {
			colList += ", "
			phList += ", "
		}
		colList += c
		phList += placeholders[i]
	}

	// Use RETURNING userid to obtain created id
	query := fmt.Sprintf("INSERT INTO users (%s) VALUES (%s) RETURNING userid", colList, phList)

	var r struct {
		UserID int `gorm:"column:userid"`
	}
	if err := s.db.Raw(query, args...).Scan(&r).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	newUserID := uint(r.UserID)

	// Assign role: prefer super_admin then admin
	role, err := s.GetRoleByName("super_admin")
	if err != nil || role == nil {
		role, err = s.GetRoleByName("admin")
		if err != nil {
			return fmt.Errorf("failed to lookup admin role: %w", err)
		}
		if role == nil {
			return fmt.Errorf("failed to assign role to seeded admin user: neither super_admin nor admin role exists in the database; ensure migrations have been run and core roles are seeded")
		}
	}
	ur := models.UserRole{UserID: newUserID, RoleID: role.ID, AssignedAt: time.Now(), IsActive: true}
	if err := s.db.Create(&ur).Error; err != nil {
		return fmt.Errorf("failed to assign role to admin user: %w", err)
	}

	log.Printf("[RBAC] Seeded admin user '%s'", username)

	return nil
}

func (s *RBACService) userColumnExists(column string) int {
	var exists int
	if err := s.db.Raw("SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name='users' AND column_name=?", column).Scan(&exists).Error; err != nil {
		log.Printf("[RBAC] WARNING: failed to check users.%s column existence: %v", column, err)
		return 0
	}
	return exists
}
