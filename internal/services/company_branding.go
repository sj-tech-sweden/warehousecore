package services

import (
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const defaultBrandName = "RentalCore"

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
		ttl: 5 * time.Minute,
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

	var record companyRecord
	if err := s.db.Order("id DESC").First(&record).Error; err == nil {
		s.name = sanitizeBrandName(record.CompanyName)
	} else if s.name == "" {
		s.name = defaultBrandName
	}
	s.lastFetch = time.Now()
	return s.name
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
