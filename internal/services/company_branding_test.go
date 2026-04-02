package services

import (
	"testing"
	"time"
)

func TestSanitizeBrandNameEmpty(t *testing.T) {
	result := sanitizeBrandName("")
	if result != defaultBrandName {
		t.Fatalf("expected %q for empty string, got %q", defaultBrandName, result)
	}
}

func TestSanitizeBrandNameWhitespace(t *testing.T) {
	result := sanitizeBrandName("   ")
	if result != defaultBrandName {
		t.Fatalf("expected %q for whitespace-only string, got %q", defaultBrandName, result)
	}
}

func TestSanitizeBrandNameTrimmed(t *testing.T) {
	result := sanitizeBrandName("  Tsunami Events  ")
	if result != "Tsunami Events" {
		t.Fatalf("expected \"Tsunami Events\", got %q", result)
	}
}

func TestSanitizeBrandNameNormal(t *testing.T) {
	result := sanitizeBrandName("WarehouseCore")
	if result != "WarehouseCore" {
		t.Fatalf("expected \"WarehouseCore\", got %q", result)
	}
}

func TestCompanyBrandingServiceUpdate(t *testing.T) {
	svc := &CompanyBrandingService{
		ttl: 30 * time.Second,
	}

	svc.Update("Tsunami Events")
	if svc.name != "Tsunami Events" {
		t.Fatalf("expected name=\"Tsunami Events\", got %q", svc.name)
	}
	if svc.lastFetch.IsZero() {
		t.Fatal("expected lastFetch to be set after Update")
	}
}

func TestCompanyBrandingServiceUpdateEmptyFallsToDefault(t *testing.T) {
	svc := &CompanyBrandingService{
		ttl: 30 * time.Second,
	}

	svc.Update("")
	if svc.name != defaultBrandName {
		t.Fatalf("expected %q for empty update, got %q", defaultBrandName, svc.name)
	}
}

func TestCompanyBrandingServiceInvalidate(t *testing.T) {
	svc := &CompanyBrandingService{
		name:      "SomeName",
		lastFetch: time.Now(),
		ttl:       30 * time.Second,
	}

	svc.Invalidate()
	if !svc.lastFetch.IsZero() {
		t.Fatal("expected lastFetch to be zeroed after Invalidate")
	}
}

func TestCompanyBrandingServiceCachedName(t *testing.T) {
	svc := &CompanyBrandingService{
		ttl: 30 * time.Second,
	}
	svc.Update("CachedCo")

	// CompanyName should return cached value without hitting DB (cache is fresh)
	name := svc.CompanyName()
	if name != "CachedCo" {
		t.Fatalf("expected \"CachedCo\" from cache, got %q", name)
	}
}
