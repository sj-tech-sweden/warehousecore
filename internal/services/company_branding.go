package services

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"warehousecore/internal/models"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

const defaultBrandName = "WarehouseCore"

type companyRecord struct {
	ID          uint   `gorm:"column:id"`
	CompanyName string `gorm:"column:company_name"`
}

func (companyRecord) TableName() string {
	return "company_settings"
}

// CompanyBrandingService provides cached access to the shared company name.
type CompanyBrandingService struct {
	db        *gorm.DB
	mu        sync.RWMutex
	name      string
	lastFetch time.Time
	ttl       time.Duration
}

func NewCompanyBrandingService(db *gorm.DB) *CompanyBrandingService {
	return &CompanyBrandingService{
		db:  db,
		ttl: 30 * time.Second,
	}
}

func (s *CompanyBrandingService) CompanyName() string {
	s.mu.RLock()
	name := s.name
	fresh := time.Since(s.lastFetch) < s.ttl && name != ""
	s.mu.RUnlock()
	if fresh {
		return name
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Since(s.lastFetch) < s.ttl && s.name != "" {
		return s.name
	}

	// Prefer app_settings (authoritative admin-managed company.info).
	var appSetting models.AppSetting
	appSettingsQuery := s.db.Table("app_settings").Where("scope = ? AND key = ?", "warehousecore", "company.info").Limit(1).Find(&appSetting)
	if appSettingsQuery.Error != nil && !isMissingAppSettingsTableErr(appSettingsQuery.Error) {
		log.Printf("[BRANDING] app_settings lookup failed: %v", appSettingsQuery.Error)
	}
	if appSettingsQuery.Error == nil && appSettingsQuery.RowsAffected > 0 {
		if nameVal, ok := appSetting.Value["name"].(string); ok && strings.TrimSpace(nameVal) != "" {
			s.name = sanitizeBrandName(nameVal)
		}
	}

	if s.name == "" {
		var record companyRecord
		companyQuery := s.db.Order("id DESC").Limit(1).Find(&record)
		if companyQuery.Error != nil {
			log.Printf("[BRANDING] company_settings lookup failed: %v", companyQuery.Error)
		}
		if companyQuery.Error == nil && companyQuery.RowsAffected > 0 {
			s.name = sanitizeBrandName(record.CompanyName)
		}
	}

	if s.name == "" {
		s.name = defaultBrandName
	}
	s.lastFetch = time.Now()
	return s.name
}

func isMissingAppSettingsTableErr(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return true
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "42P01" {
		return true
	}
	// Fallback for wrapped/non-pq errors where SQLSTATE is not exposed.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, `relation "app_settings" does not exist`)
}

func (s *CompanyBrandingService) Update(name string) {
	s.mu.Lock()
	s.name = sanitizeBrandName(name)
	s.lastFetch = time.Now()
	s.mu.Unlock()
}

func (s *CompanyBrandingService) Invalidate() {
	s.mu.Lock()
	s.lastFetch = time.Time{}
	s.mu.Unlock()
}

func sanitizeBrandName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return defaultBrandName
	}
	return trimmed
}
