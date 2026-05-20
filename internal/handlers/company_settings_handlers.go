package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"warehousecore/internal/models"
	"warehousecore/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CompanySettings struct {
	// Basic
	Name         string `json:"name"`
	AddressLine1 string `json:"address_line1,omitempty"`
	AddressLine2 string `json:"address_line2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
	Country      string `json:"country,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Email        string `json:"email,omitempty"`
	Website      string `json:"website,omitempty"`

	// Invoicing / Legal
	TaxNumber      string  `json:"tax_number,omitempty"`
	VATNumber      string  `json:"vat_number,omitempty"`
	InvoicePrefix  string  `json:"invoice_prefix,omitempty"`
	InvoiceFooter  string  `json:"invoice_footer,omitempty"`
	DefaultTaxRate float64 `json:"default_tax_rate,omitempty"`
	Currency       string  `json:"currency,omitempty"`

	// Banking
	BankName      string `json:"bank_name,omitempty"`
	IBAN          string `json:"iban,omitempty"`
	BIC           string `json:"bic,omitempty"`
	AccountHolder string `json:"account_holder,omitempty"`

	// Company details
	CEOName        string `json:"ceo_name,omitempty"`
	RegisterCourt  string `json:"register_court,omitempty"`
	RegisterNumber string `json:"register_number,omitempty"`

	// Branding
	BrandPrimaryColor string `json:"brand_primary_color,omitempty"`
	BrandAccentColor  string `json:"brand_accent_color,omitempty"`
	BrandDarkMode     *bool  `json:"brand_dark_mode,omitempty"`
	BrandLogoURL      string `json:"brand_logo_url,omitempty"`

	// SMTP
	SMTPHost      string `json:"smtp_host,omitempty"`
	SMTPPort      *int   `json:"smtp_port,omitempty"`
	SMTPUsername  string `json:"smtp_username,omitempty"`
	SMTPPassword  string `json:"smtp_password,omitempty"`
	SMTPFromEmail string `json:"smtp_from_email,omitempty"`
	SMTPFromName  string `json:"smtp_from_name,omitempty"`
	SMTPUseTLS    *bool  `json:"smtp_use_tls,omitempty"`
}

const companySMTPEncryptedPrefix = "enc:"

func companySMTPCredentialKey() ([]byte, error) {
	if raw := strings.TrimSpace(os.Getenv("COMPANY_SMTP_CREDENTIAL_KEY")); raw != "" {
		return parseCompanySMTPCredentialKey(raw, "COMPANY_SMTP_CREDENTIAL_KEY")
	}
	if raw := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY")); raw != "" {
		return parseCompanySMTPCredentialKey(raw, "ENCRYPTION_KEY")
	}
	return nil, nil
}

func parseCompanySMTPCredentialKey(raw string, source string) ([]byte, error) {
	if len(raw) == 32 {
		return []byte(raw), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	return nil, fmt.Errorf("%s must be exactly 32 bytes or base64-encoded 32 bytes", source)
}

func encryptCompanySMTPPassword(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key, err := companySMTPCredentialKey()
	if err != nil {
		return "", err
	}
	if len(key) == 0 {
		return "", errors.New("encryption key not configured: set COMPANY_SMTP_CREDENTIAL_KEY or ENCRYPTION_KEY to a 32-byte value or base64-encoded 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return companySMTPEncryptedPrefix + base64.URLEncoding.EncodeToString(ciphertext), nil
}

// GetCompanySettings returns company settings stored in app_settings (scope=warehousecore, key=company.info)
func GetCompanySettings(w http.ResponseWriter, r *http.Request) {
	db := repository.GetDB()
	if db == nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "database not available"})
		return
	}

	var setting models.AppSetting
	if err := db.Where("scope = ? AND key = ?", "warehousecore", "company.info").First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondJSON(w, http.StatusOK, CompanySettings{})
			return
		}
		log.Printf("[COMPANY] Failed to read company settings: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var cs CompanySettings
	b, _ := json.Marshal(setting.Value)
	_ = json.Unmarshal(b, &cs)
	cs.SMTPPassword = ""

	respondJSON(w, http.StatusOK, cs)
}

// UpdateCompanySettings creates or updates the company.info setting (scope=warehousecore)
func UpdateCompanySettings(w http.ResponseWriter, r *http.Request) {
	db := repository.GetDB()
	if db == nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "database not available"})
		return
	}

	var cs CompanySettings
	if err := json.NewDecoder(r.Body).Decode(&cs); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	var existingSetting models.AppSetting
	existingSMTPPassword := ""
	if err := db.Where("scope = ? AND key = ?", "warehousecore", "company.info").First(&existingSetting).Error; err == nil {
		if smtpPassword, ok := existingSetting.Value["smtp_password"].(string); ok {
			existingSMTPPassword = smtpPassword
		}
	}

	smtpPasswordForStorage := existingSMTPPassword
	if strings.TrimSpace(cs.SMTPPassword) != "" {
		trimmedSMTPPassword := strings.TrimSpace(cs.SMTPPassword)
		if strings.HasPrefix(trimmedSMTPPassword, companySMTPEncryptedPrefix) {
			smtpPasswordForStorage = trimmedSMTPPassword
		} else {
			encrypted, err := encryptCompanySMTPPassword(trimmedSMTPPassword)
			if err != nil {
				log.Printf("[COMPANY] Failed to encrypt SMTP password: %v", err)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encrypt SMTP password"})
				return
			}
			smtpPasswordForStorage = encrypted
		}
	}

	// Map struct to JSONMap for storage
	value := models.JSONMap{
		"name":          cs.Name,
		"address_line1": cs.AddressLine1,
		"address_line2": cs.AddressLine2,
		"city":          cs.City,
		"state":         cs.State,
		"postal_code":   cs.PostalCode,
		"country":       cs.Country,
		"phone":         cs.Phone,
		"email":         cs.Email,
		"website":       cs.Website,

		"tax_number":       cs.TaxNumber,
		"vat_number":       cs.VATNumber,
		"invoice_prefix":   cs.InvoicePrefix,
		"invoice_footer":   cs.InvoiceFooter,
		"default_tax_rate": cs.DefaultTaxRate,
		"currency":         cs.Currency,

		"bank_name":      cs.BankName,
		"iban":           cs.IBAN,
		"bic":            cs.BIC,
		"account_holder": cs.AccountHolder,

		"ceo_name":        cs.CEOName,
		"register_court":  cs.RegisterCourt,
		"register_number": cs.RegisterNumber,

		"brand_primary_color": cs.BrandPrimaryColor,
		"brand_accent_color":  cs.BrandAccentColor,
		"brand_dark_mode":     cs.BrandDarkMode,
		"brand_logo_url":      cs.BrandLogoURL,

		"smtp_host":       cs.SMTPHost,
		"smtp_port":       cs.SMTPPort,
		"smtp_username":   cs.SMTPUsername,
		"smtp_password":   smtpPasswordForStorage,
		"smtp_from_email": cs.SMTPFromEmail,
		"smtp_from_name":  cs.SMTPFromName,
		"smtp_use_tls":    cs.SMTPUseTLS,
	}

	setting := models.AppSetting{
		Scope: "warehousecore",
		Key:   "company.info",
		Value: value,
	}

	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "scope"}, {Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"value": value, "updated_at": gorm.Expr("NOW()")}),
	}).Create(&setting).Error; err != nil {
		log.Printf("[COMPANY] Failed to save company settings: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	cs.SMTPPassword = ""
	respondJSON(w, http.StatusOK, cs)
}
