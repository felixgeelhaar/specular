package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Tier represents the subscription level
type Tier string

const (
	// TierFree represents the free tier with basic features
	TierFree Tier = "free"
	// TierPro represents the pro tier with advanced features
	TierPro Tier = "pro"
	// TierEnterprise represents the enterprise tier with all features
	TierEnterprise Tier = "enterprise"
)

// License represents a Specular license
type License struct {
	Tier      Tier      `json:"tier"`
	Key       string    `json:"key,omitempty"`
	Email     string    `json:"email,omitempty"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Features  []string  `json:"features,omitempty"`
}

// FeatureGateError is returned when a feature requires a higher tier
type FeatureGateError struct {
	Feature      string
	RequiredTier Tier
	CurrentTier  Tier
}

func (e *FeatureGateError) Error() string {
	return fmt.Sprintf("feature '%s' requires %s tier (current: %s)",
		e.Feature, e.RequiredTier, e.CurrentTier)
}

// TierFeatures maps each tier to its available features
var TierFeatures = map[Tier][]string{
	TierFree: {
		"spec.generate",
		"spec.lock",
		"spec.validate",
		"plan.create",
		"plan.review",
		"plan.explain",
		"bundle.create",
		"drift.check",
		"provider.local",
		"generate.basic",
		"auto.basic",
	},
	TierPro: {
		// All free features plus:
		"governance.init",
		"governance.doctor",
		"governance.status",
		"policy.init",
		"policy.validate",
		"policy.approve",
		"policy.list",
		"policy.diff",
		"approvals.create",
		"approvals.list",
		"bundle.gate",
		"bundle.inspect",
		"bundle.list",
		"attestations.ecdsa",
		"provider.cloud",
		"provider.routing",
		"hooks.script",
		"hooks.webhook",
		"hooks.slack",
		"security.encryption",
		"security.audit",
		"auto.advanced",
		"auto.checkpoint",
	},
	TierEnterprise: {
		// All Pro features plus:
		"governance.rbac",
		"governance.multi-tenancy",
		"policy.advanced",
		"policy.custom-rules",
		"compliance.soc2",
		"compliance.export",
		"audit.detailed",
		"audit.export",
		"sso.saml",
		"sso.oidc",
		"security.vault-integration",
		"monitoring.prometheus",
		"monitoring.grafana",
		"support.priority",
		"support.dedicated",
	},
}

// DefaultLicense returns a free tier license
func DefaultLicense() *License {
	return &License{
		Tier:     TierFree,
		IssuedAt: time.Now(),
	}
}

// Load reads the license from .specular/license.json
func Load() (*License, error) {
	licensePath := filepath.Join(".specular", "license.json")

	// If no license file exists, return free tier
	if _, err := os.Stat(licensePath); os.IsNotExist(err) {
		return DefaultLicense(), nil
	}

	data, err := os.ReadFile(licensePath)
	if err != nil {
		return nil, fmt.Errorf("reading license file: %w", err)
	}

	var lic License
	if err := json.Unmarshal(data, &lic); err != nil {
		return nil, fmt.Errorf("parsing license file: %w", err)
	}

	// Validate expiration
	if !lic.ExpiresAt.IsZero() && time.Now().After(lic.ExpiresAt) {
		return nil, errors.New("license expired")
	}

	return &lic, nil
}

// Save writes the license to .specular/license.json
func (l *License) Save() error {
	specularDir := ".specular"
	if err := os.MkdirAll(specularDir, 0755); err != nil {
		return fmt.Errorf("creating .specular directory: %w", err)
	}

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling license: %w", err)
	}

	licensePath := filepath.Join(specularDir, "license.json")
	if err := os.WriteFile(licensePath, data, 0644); err != nil {
		return fmt.Errorf("writing license file: %w", err)
	}

	return nil
}

// HasFeature checks if the license includes access to a specific feature
func (l *License) HasFeature(feature string) bool {
	features := TierFeatures[l.Tier]

	// Check if feature is in current tier
	for _, f := range features {
		if f == feature {
			return true
		}
	}

	// Check inherited features from lower tiers
	if l.Tier == TierPro || l.Tier == TierEnterprise {
		for _, f := range TierFeatures[TierFree] {
			if f == feature {
				return true
			}
		}
	}

	if l.Tier == TierEnterprise {
		for _, f := range TierFeatures[TierPro] {
			if f == feature {
				return true
			}
		}
	}

	return false
}

// RequireFeature checks if a feature is available and returns an error if not
func RequireFeature(feature string, requiredTier Tier) error {
	lic, err := Load()
	if err != nil {
		// If we can't load license, assume free tier
		lic = DefaultLicense()
	}

	if !lic.HasFeature(feature) {
		return &FeatureGateError{
			Feature:      feature,
			RequiredTier: requiredTier,
			CurrentTier:  lic.Tier,
		}
	}

	return nil
}

// GetTier returns the current license tier
func GetTier() (Tier, error) {
	lic, err := Load()
	if err != nil {
		return TierFree, err
	}
	return lic.Tier, nil
}

// IsPro returns true if the license is Pro or Enterprise
func IsPro() bool {
	tier, _ := GetTier()
	return tier == TierPro || tier == TierEnterprise
}

// IsEnterprise returns true if the license is Enterprise
func IsEnterprise() bool {
	tier, _ := GetTier()
	return tier == TierEnterprise
}
