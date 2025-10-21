package services

import (
	"fmt"
	"gorm.io/gorm"
	"log"
	"os"
	"time"
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
		Joins("JOIN user_roles ON user_roles.roleID = roles.roleID").
		Where("user_roles.userID = ?", userID).
		Find(&roles).Error

	return roles, err
}

// HasRole checks if a user has a specific role
func (s *RBACService) HasRole(userID uint, roleName string) (bool, error) {
	var count int64
	err := s.db.Table("user_roles").
		Joins("JOIN roles ON roles.roleID = user_roles.roleID").
		Where("user_roles.userID = ? AND roles.name = ?", userID, roleName).
		Count(&count).Error

	return count > 0, err
}

// HasAnyRole checks if a user has any of the specified roles
func (s *RBACService) HasAnyRole(userID uint, roleNames []string) (bool, error) {
	var count int64
	err := s.db.Table("user_roles").
		Joins("JOIN roles ON roles.roleID = user_roles.roleID").
		Where("user_roles.userID = ? AND roles.name IN ?", userID, roleNames).
		Count(&count).Error

	return count > 0, err
}

// AssignRole assigns a role to a user
func (s *RBACService) AssignRole(userID uint, roleID int) error {
	userRole := models.UserRole{UserID: userID, RoleID: roleID, AssignedAt: time.Now().UTC(), IsActive: true}
	return s.db.Create(&userRole).Error
}

// RemoveRole removes a role from a user
func (s *RBACService) RemoveRole(userID uint, roleID int) error {
	return s.db.Where("userID = ? AND roleID = ?", userID, roleID).
		Delete(&models.UserRole{}).Error
}

// SetUserRoles replaces all user roles with the provided list
func (s *RBACService) SetUserRoles(userID uint, roleIDs []int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing roles
		if err := tx.Where("userID = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}

		// Insert new roles
		for _, roleID := range roleIDs {
			userRole := models.UserRole{UserID: userID, RoleID: roleID, IsActive: true}
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
func (s *RBACService) CreateRole(name, description string) (*models.Role, error) {
	role := models.Role{
		Name:        name,
		Description: description,
	}

	if err := s.db.Create(&role).Error; err != nil {
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
