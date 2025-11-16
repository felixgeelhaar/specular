package license

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultLicense(t *testing.T) {
	lic := DefaultLicense()
	if lic.Tier != TierFree {
		t.Errorf("expected free tier, got %s", lic.Tier)
	}
	if lic.IssuedAt.IsZero() {
		t.Error("expected IssuedAt to be set")
	}
}

func TestHasFeature(t *testing.T) {
	tests := []struct {
		name     string
		tier     Tier
		feature  string
		expected bool
	}{
		// Free tier features
		{"free has spec.generate", TierFree, "spec.generate", true},
		{"free has plan.create", TierFree, "plan.create", true},
		{"free does not have governance.init", TierFree, "governance.init", false},
		{"free does not have policy.approve", TierFree, "policy.approve", false},

		// Pro tier features
		{"pro has spec.generate", TierPro, "spec.generate", true}, // inherited from free
		{"pro has governance.init", TierPro, "governance.init", true},
		{"pro has policy.approve", TierPro, "policy.approve", true},
		{"pro does not have governance.rbac", TierPro, "governance.rbac", false},
		{"pro does not have sso.saml", TierPro, "sso.saml", false},

		// Enterprise tier features
		{"enterprise has spec.generate", TierEnterprise, "spec.generate", true}, // inherited from free
		{"enterprise has governance.init", TierEnterprise, "governance.init", true}, // inherited from pro
		{"enterprise has governance.rbac", TierEnterprise, "governance.rbac", true},
		{"enterprise has sso.saml", TierEnterprise, "sso.saml", true},
		{"enterprise has compliance.soc2", TierEnterprise, "compliance.soc2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lic := &License{Tier: tt.tier}
			result := lic.HasFeature(tt.feature)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLicenseSaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create and save license
	lic := &License{
		Tier:      TierPro,
		Key:       "test-key-123",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	}

	if err := lic.Save(); err != nil {
		t.Fatalf("failed to save license: %v", err)
	}

	// Verify file exists
	licensePath := filepath.Join(".specular", "license.json")
	if _, err := os.Stat(licensePath); os.IsNotExist(err) {
		t.Fatal("license file was not created")
	}

	// Load license
	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load license: %v", err)
	}

	// Verify contents
	if loaded.Tier != TierPro {
		t.Errorf("expected tier %s, got %s", TierPro, loaded.Tier)
	}
	if loaded.Key != "test-key-123" {
		t.Errorf("expected key 'test-key-123', got '%s'", loaded.Key)
	}
	if loaded.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", loaded.Email)
	}
}

func TestLoadDefaultsToFree(t *testing.T) {
	// Create temp directory with no license file
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	lic, err := Load()
	if err != nil {
		t.Fatalf("failed to load default license: %v", err)
	}

	if lic.Tier != TierFree {
		t.Errorf("expected free tier, got %s", lic.Tier)
	}
}

func TestExpiredLicense(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create expired license
	lic := &License{
		Tier:      TierPro,
		Key:       "test-key-expired",
		IssuedAt:  time.Now().Add(-365 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	if err := lic.Save(); err != nil {
		t.Fatalf("failed to save license: %v", err)
	}

	// Try to load expired license
	_, err := Load()
	if err == nil {
		t.Error("expected error for expired license, got nil")
	}
	if err.Error() != "license expired" {
		t.Errorf("expected 'license expired' error, got '%s'", err.Error())
	}
}

func TestRequireFeature(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create free tier license
	lic := DefaultLicense()
	if err := lic.Save(); err != nil {
		t.Fatalf("failed to save license: %v", err)
	}

	// Test free tier feature (should succeed)
	err := RequireFeature("spec.generate", TierFree)
	if err != nil {
		t.Errorf("expected no error for free feature, got %v", err)
	}

	// Test pro tier feature (should fail)
	err = RequireFeature("governance.init", TierPro)
	if err == nil {
		t.Error("expected error for pro feature on free tier, got nil")
	}

	gateErr, ok := err.(*FeatureGateError)
	if !ok {
		t.Errorf("expected FeatureGateError, got %T", err)
	}
	if gateErr.Feature != "governance.init" {
		t.Errorf("expected feature 'governance.init', got '%s'", gateErr.Feature)
	}
	if gateErr.RequiredTier != TierPro {
		t.Errorf("expected required tier %s, got %s", TierPro, gateErr.RequiredTier)
	}
	if gateErr.CurrentTier != TierFree {
		t.Errorf("expected current tier %s, got %s", TierFree, gateErr.CurrentTier)
	}
}

func TestIsPro(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Test with free tier
	lic := DefaultLicense()
	lic.Save()
	if IsPro() {
		t.Error("expected IsPro() to return false for free tier")
	}

	// Test with pro tier
	lic.Tier = TierPro
	lic.Save()
	if !IsPro() {
		t.Error("expected IsPro() to return true for pro tier")
	}

	// Test with enterprise tier
	lic.Tier = TierEnterprise
	lic.Save()
	if !IsPro() {
		t.Error("expected IsPro() to return true for enterprise tier")
	}
}

func TestIsEnterprise(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Test with free tier
	lic := DefaultLicense()
	lic.Save()
	if IsEnterprise() {
		t.Error("expected IsEnterprise() to return false for free tier")
	}

	// Test with pro tier
	lic.Tier = TierPro
	lic.Save()
	if IsEnterprise() {
		t.Error("expected IsEnterprise() to return false for pro tier")
	}

	// Test with enterprise tier
	lic.Tier = TierEnterprise
	lic.Save()
	if !IsEnterprise() {
		t.Error("expected IsEnterprise() to return true for enterprise tier")
	}
}

func TestLicenseJSONFormat(t *testing.T) {
	lic := &License{
		Tier:      TierPro,
		Key:       "test-key",
		Email:     "test@example.com",
		IssuedAt:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.MarshalIndent(lic, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal license: %v", err)
	}

	var loaded License
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal license: %v", err)
	}

	if loaded.Tier != TierPro {
		t.Errorf("expected tier %s, got %s", TierPro, loaded.Tier)
	}
	if loaded.Key != "test-key" {
		t.Errorf("expected key 'test-key', got '%s'", loaded.Key)
	}
}
