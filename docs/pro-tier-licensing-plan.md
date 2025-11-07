# Pro Tier Licensing Implementation Plan

**Status:** Planned for v1.4.0+
**Target:** Q2-Q3 2025
**Dependencies:** v1.2.0 CLI Enhancement (✅ Complete), v1.3.0 Governance Bundle

---

## 1. Overview

This document outlines the implementation plan for Specular's tiered licensing system, enabling Pro and Enterprise feature gating while maintaining a free MVP tier.

### Tier Structure

| Tier | Price | Target Audience | Key Features |
|------|-------|-----------------|--------------|
| **Free** | $0 | Individual developers | Spec, Plan, Build, Eval, Drift |
| **Pro** | $49/user/month | Teams | + Team routing, Approvals, Cloud sync |
| **Enterprise** | Custom | Organizations | + Compliance, SSO, Guardians, Attestations |

---

## 2. Architecture

### 2.1 Directory Structure

```
internal/
├── license/
│   ├── license.go          # Core license types and management
│   ├── tier.go             # Tier definitions and feature mappings
│   ├── validator.go        # License validation (JWT, file, cloud)
│   ├── checker.go          # Runtime feature gate checks
│   ├── loader.go           # License discovery and loading
│   └── license_test.go     # Comprehensive test suite
├── cmd/
│   └── license.go          # License management commands
└── config/
    └── license_config.go   # License configuration handling
```

### 2.2 Configuration Files

```
~/.specular/
├── license.yaml            # User license file
└── config.yaml             # Updated with license settings

.specular/
└── license.yaml            # Project-specific license (optional)
```

---

## 3. Implementation Phases

### Phase 1: Core License System (v1.4.0)

**Goal:** Implement basic license validation and feature gating

#### Components to Build

1. **License Data Structures** (`internal/license/license.go`)
```go
type License struct {
    Version    string    `json:"version" yaml:"version"`
    Tier       Tier      `json:"tier" yaml:"tier"`
    OrgID      string    `json:"org_id" yaml:"org_id"`
    OrgName    string    `json:"org_name" yaml:"org_name"`
    IssuedAt   time.Time `json:"issued_at" yaml:"issued_at"`
    ExpiresAt  time.Time `json:"expires_at" yaml:"expires_at"`
    MaxSeats   int       `json:"max_seats,omitempty" yaml:"max_seats,omitempty"`
    Features   []Feature `json:"features,omitempty" yaml:"features,omitempty"`
    Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type Tier int

const (
    TierFree Tier = iota
    TierPro
    TierEnterprise
)

func (t Tier) String() string {
    return [...]string{"Free", "Pro", "Enterprise"}[t]
}
```

2. **Feature Definitions** (`internal/license/tier.go`)
```go
type Feature string

const (
    // MVP Features (Free Tier)
    FeatureSpec         Feature = "spec"
    FeaturePlan         Feature = "plan"
    FeatureBuild        Feature = "build"
    FeatureEval         Feature = "eval"
    FeatureDrift        Feature = "drift"
    FeatureDoctor       Feature = "doctor"
    FeatureRoute        Feature = "route"
    FeatureInit         Feature = "init"

    // Pro Features (v1.4.0+)
    FeatureTeamRouting  Feature = "team_routing"
    FeatureApprovals    Feature = "approvals"
    FeatureCloudSync    Feature = "cloud_sync"
    FeatureDriftInbox   Feature = "drift_inbox"
    FeatureSharedSpecs  Feature = "shared_specs"

    // Enterprise Features (v1.5.0+)
    FeatureGuardians    Feature = "guardians"
    FeatureCompliance   Feature = "compliance"
    FeatureSSO          Feature = "sso"
    FeatureAttestation  Feature = "attestation"
    FeatureAuditLog     Feature = "audit_log"
    FeatureRBAC         Feature = "rbac"
    FeaturePrivateHub   Feature = "private_hub"
)

var tierFeatures = map[Tier][]Feature{
    TierFree: {
        FeatureSpec, FeaturePlan, FeatureBuild, FeatureEval,
        FeatureDrift, FeatureDoctor, FeatureRoute, FeatureInit,
    },
    TierPro: {
        // Inherits all Free features
        FeatureTeamRouting, FeatureApprovals, FeatureCloudSync,
        FeatureDriftInbox, FeatureSharedSpecs,
    },
    TierEnterprise: {
        // Inherits all Pro features
        FeatureGuardians, FeatureCompliance, FeatureSSO,
        FeatureAttestation, FeatureAuditLog, FeatureRBAC,
        FeaturePrivateHub,
    },
}

// GetTierFeatures returns all features available for a tier (including inherited)
func GetTierFeatures(tier Tier) []Feature {
    features := make([]Feature, 0)

    // Always include free features
    features = append(features, tierFeatures[TierFree]...)

    // Add Pro features if Pro or higher
    if tier >= TierPro {
        features = append(features, tierFeatures[TierPro]...)
    }

    // Add Enterprise features if Enterprise
    if tier >= TierEnterprise {
        features = append(features, tierFeatures[TierEnterprise]...)
    }

    return features
}
```

3. **License Loader** (`internal/license/loader.go`)
```go
type Loader struct {
    // Configuration
    configDir string
    projectDir string
}

func NewLoader() *Loader {
    return &Loader{
        configDir: os.ExpandEnv("$HOME/.specular"),
        projectDir: ".specular",
    }
}

// Load attempts to load license from multiple sources (priority order)
func (l *Loader) Load() (*License, error) {
    // 1. Environment variable (highest priority)
    if key := os.Getenv("SPECULAR_LICENSE_KEY"); key != "" {
        return l.loadFromKey(key)
    }

    // 2. User config file
    userLicense := filepath.Join(l.configDir, "license.yaml")
    if license, err := l.loadFromFile(userLicense); err == nil {
        return license, nil
    }

    // 3. Project-specific license
    projectLicense := filepath.Join(l.projectDir, "license.yaml")
    if license, err := l.loadFromFile(projectLicense); err == nil {
        return license, nil
    }

    // 4. Default to free tier
    return &License{
        Version: "1.0",
        Tier: TierFree,
        IssuedAt: time.Now(),
        ExpiresAt: time.Time{}, // Never expires
        OrgID: "free",
        OrgName: "Free Tier",
    }, nil
}

func (l *Loader) loadFromFile(path string) (*License, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var license License
    if err := yaml.Unmarshal(data, &license); err != nil {
        return nil, err
    }

    return &license, nil
}

func (l *Loader) loadFromKey(key string) (*License, error) {
    // Decode JWT license key
    // For MVP: simple base64 + HMAC validation
    // For Production: full JWT with RSA signatures
    return nil, fmt.Errorf("JWT license keys not yet implemented")
}
```

4. **Feature Checker** (`internal/license/checker.go`)
```go
type Checker struct {
    license *License
}

func NewChecker() (*Checker, error) {
    loader := NewLoader()
    license, err := loader.Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load license: %w", err)
    }

    return &Checker{license: license}, nil
}

// RequireFeature checks if a feature is available, returns error if not
func (c *Checker) RequireFeature(feature Feature) error {
    if !c.HasFeature(feature) {
        requiredTier := c.getRequiredTier(feature)
        return &FeatureNotAvailableError{
            Feature:      feature,
            RequiredTier: requiredTier,
            CurrentTier:  c.license.Tier,
        }
    }

    // Check if license is expired
    if !c.license.ExpiresAt.IsZero() && time.Now().After(c.license.ExpiresAt) {
        return &LicenseExpiredError{
            Tier:      c.license.Tier,
            ExpiresAt: c.license.ExpiresAt,
        }
    }

    return nil
}

// HasFeature checks if a feature is available in current license
func (c *Checker) HasFeature(feature Feature) bool {
    allowedFeatures := GetTierFeatures(c.license.Tier)
    for _, f := range allowedFeatures {
        if f == feature {
            return true
        }
    }
    return false
}

// GetTier returns the current license tier
func (c *Checker) GetTier() Tier {
    return c.license.Tier
}

// GetLicense returns the full license details
func (c *Checker) GetLicense() *License {
    return c.license
}

func (c *Checker) getRequiredTier(feature Feature) Tier {
    for tier := TierFree; tier <= TierEnterprise; tier++ {
        for _, f := range tierFeatures[tier] {
            if f == feature {
                return tier
            }
        }
    }
    return TierEnterprise // Default to highest tier if not found
}

// Custom errors
type FeatureNotAvailableError struct {
    Feature      Feature
    RequiredTier Tier
    CurrentTier  Tier
}

func (e *FeatureNotAvailableError) Error() string {
    return fmt.Sprintf(
        "feature %q requires %s tier (current: %s)",
        e.Feature,
        e.RequiredTier,
        e.CurrentTier,
    )
}

type LicenseExpiredError struct {
    Tier      Tier
    ExpiresAt time.Time
}

func (e *LicenseExpiredError) Error() string {
    return fmt.Sprintf(
        "%s license expired on %s",
        e.Tier,
        e.ExpiresAt.Format("2006-01-02"),
    )
}
```

5. **License Commands** (`internal/cmd/license.go`)
```go
var licenseCmd = &cobra.Command{
    Use:   "license",
    Short: "Manage Specular license",
    Long: `Manage and view your Specular license information.

Available subcommands:
  status   - Show current license status and features
  install  - Install a new license key
  validate - Validate current license
  upgrade  - Information about upgrading your license`,
}

var licenseStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show current license status",
    Long:  `Display detailed information about your current Specular license.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        checker, err := license.NewChecker()
        if err != nil {
            return ux.FormatError(err, "loading license")
        }

        lic := checker.GetLicense()

        fmt.Printf("Specular License Status\n")
        fmt.Printf("═══════════════════════\n\n")
        fmt.Printf("Tier:         %s\n", lic.Tier)
        fmt.Printf("Organization: %s\n", lic.OrgName)

        if !lic.ExpiresAt.IsZero() {
            daysRemaining := int(time.Until(lic.ExpiresAt).Hours() / 24)
            fmt.Printf("Expires:      %s (%d days remaining)\n",
                lic.ExpiresAt.Format("2006-01-02"), daysRemaining)
        } else {
            fmt.Printf("Expires:      Never\n")
        }

        if lic.MaxSeats > 0 {
            fmt.Printf("Max Seats:    %d\n", lic.MaxSeats)
        }

        // List available features
        fmt.Printf("\nAvailable Features:\n")
        features := license.GetTierFeatures(lic.Tier)
        for _, f := range features {
            fmt.Printf("  ✓ %s\n", f)
        }

        // Show upgrade path if not Enterprise
        if lic.Tier < license.TierEnterprise {
            fmt.Printf("\n")
            showUpgradePath(lic.Tier)
        }

        return nil
    },
}

var licenseInstallCmd = &cobra.Command{
    Use:   "install <license-key>",
    Short: "Install a new license key",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        key := args[0]

        // Validate and decode license key
        loader := license.NewLoader()
        lic, err := loader.LoadFromKey(key)
        if err != nil {
            return ux.FormatError(err, "invalid license key")
        }

        // Save to user config
        configDir := os.ExpandEnv("$HOME/.specular")
        licensePath := filepath.Join(configDir, "license.yaml")

        data, err := yaml.Marshal(lic)
        if err != nil {
            return ux.FormatError(err, "encoding license")
        }

        if err := os.MkdirAll(configDir, 0755); err != nil {
            return ux.FormatError(err, "creating config directory")
        }

        if err := os.WriteFile(licensePath, data, 0600); err != nil {
            return ux.FormatError(err, "saving license")
        }

        fmt.Printf("✓ License installed successfully\n")
        fmt.Printf("Tier: %s\n", lic.Tier)
        fmt.Printf("Organization: %s\n", lic.OrgName)

        return nil
    },
}

func showUpgradePath(currentTier license.Tier) {
    switch currentTier {
    case license.TierFree:
        fmt.Printf("Upgrade to Pro for:\n")
        for _, f := range license.tierFeatures[license.TierPro] {
            fmt.Printf("  • %s\n", f)
        }
        fmt.Printf("\nUpgrade to Enterprise for:\n")
        for _, f := range license.tierFeatures[license.TierEnterprise] {
            fmt.Printf("  • %s\n", f)
        }
    case license.TierPro:
        fmt.Printf("Upgrade to Enterprise for:\n")
        for _, f := range license.tierFeatures[license.TierEnterprise] {
            fmt.Printf("  • %s\n", f)
        }
    }
    fmt.Printf("\nVisit https://specular.dev/pricing for more information\n")
}

func init() {
    rootCmd.AddCommand(licenseCmd)
    licenseCmd.AddCommand(licenseStatusCmd)
    licenseCmd.AddCommand(licenseInstallCmd)
}
```

#### Integration with Existing Commands

**Example: Gating a Pro Feature**

```go
// internal/cmd/approve.go (Pro feature)
var approveCmd = &cobra.Command{
    Use:   "approve",
    Short: "Approval workflow management (Pro)",
    Long: `Manage approval workflows for specs and plans.

This is a Pro feature. Upgrade to Pro to enable approval workflows.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        checker, err := license.NewChecker()
        if err != nil {
            return err
        }

        // Check for Pro feature
        if err := checker.RequireFeature(license.FeatureApprovals); err != nil {
            if licErr, ok := err.(*license.FeatureNotAvailableError); ok {
                return fmt.Errorf(
                    "%w\n\n" +
                    "Approval workflows require a Pro or Enterprise license.\n" +
                    "Run 'specular license status' to see your current tier.\n" +
                    "Visit https://specular.dev/pricing to upgrade.",
                    licErr,
                )
            }
            return err
        }

        // Proceed with approval logic
        return runApprove(cmd, args)
    },
}
```

---

### Phase 2: License Validation (v1.4.1)

**Goal:** Add cryptographic license validation

#### JWT-Based License Keys

```go
// internal/license/validator.go
package license

import (
    "crypto/rsa"
    "encoding/base64"
    "fmt"
    "github.com/golang-jwt/jwt/v5"
    "time"
)

type Validator struct {
    publicKey *rsa.PublicKey
}

// Embedded public key for license validation
const publicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----`

func NewValidator() (*Validator, error) {
    pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
    if err != nil {
        return nil, fmt.Errorf("failed to parse public key: %w", err)
    }

    return &Validator{publicKey: pubKey}, nil
}

type LicenseClaims struct {
    jwt.RegisteredClaims
    Tier     string   `json:"tier"`
    OrgID    string   `json:"org_id"`
    OrgName  string   `json:"org_name"`
    MaxSeats int      `json:"max_seats,omitempty"`
    Features []string `json:"features,omitempty"`
}

func (v *Validator) Validate(licenseKey string) (*License, error) {
    // Parse JWT
    token, err := jwt.ParseWithClaims(
        licenseKey,
        &LicenseClaims{},
        func(token *jwt.Token) (interface{}, error) {
            // Verify signing method
            if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return v.publicKey, nil
        },
    )

    if err != nil {
        return nil, fmt.Errorf("invalid license key: %w", err)
    }

    claims, ok := token.Claims.(*LicenseClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid license claims")
    }

    // Convert claims to License
    license := &License{
        Version:   "1.0",
        OrgID:     claims.OrgID,
        OrgName:   claims.OrgName,
        IssuedAt:  claims.IssuedAt.Time,
        ExpiresAt: claims.ExpiresAt.Time,
        MaxSeats:  claims.MaxSeats,
    }

    // Parse tier
    switch claims.Tier {
    case "free":
        license.Tier = TierFree
    case "pro":
        license.Tier = TierPro
    case "enterprise":
        license.Tier = TierEnterprise
    default:
        return nil, fmt.Errorf("unknown tier: %s", claims.Tier)
    }

    // Parse features
    for _, f := range claims.Features {
        license.Features = append(license.Features, Feature(f))
    }

    return license, nil
}
```

---

### Phase 3: Cloud Validation (v1.5.0+)

**Goal:** Optional cloud-based license validation for Enterprise

```go
// internal/license/cloud.go
package license

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type CloudValidator struct {
    endpoint string
    apiKey   string
    cache    *LicenseCache
}

func NewCloudValidator(apiKey string) *CloudValidator {
    return &CloudValidator{
        endpoint: "https://license.specular.dev/api/v1",
        apiKey:   apiKey,
        cache:    NewLicenseCache(24 * time.Hour), // Cache for 24h
    }
}

type ValidateRequest struct {
    LicenseKey string `json:"license_key"`
    MachineID  string `json:"machine_id,omitempty"`
}

type ValidateResponse struct {
    Valid     bool      `json:"valid"`
    License   *License  `json:"license,omitempty"`
    Message   string    `json:"message,omitempty"`
    ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func (v *CloudValidator) Validate(licenseKey string) (*License, error) {
    // Check cache first
    if cached, ok := v.cache.Get(licenseKey); ok {
        return cached, nil
    }

    // Call validation API
    req := ValidateRequest{
        LicenseKey: licenseKey,
        MachineID:  getMachineID(),
    }

    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequest(
        "POST",
        v.endpoint+"/validate",
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("X-API-Key", v.apiKey)

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(httpReq)
    if err != nil {
        // Fallback to cached license if offline
        if cached, ok := v.cache.GetStale(licenseKey); ok {
            return cached, nil
        }
        return nil, fmt.Errorf("cloud validation failed: %w", err)
    }
    defer resp.Body.Close()

    var result ValidateResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    if !result.Valid {
        return nil, fmt.Errorf("license validation failed: %s", result.Message)
    }

    // Cache successful validation
    v.cache.Set(licenseKey, result.License)

    return result.License, nil
}

func getMachineID() string {
    // Generate stable machine ID based on hardware
    // For privacy: hash of MAC address + hostname
    return ""
}
```

---

## 4. Testing Strategy

### Unit Tests

```go
// internal/license/checker_test.go
func TestChecker_RequireFeature(t *testing.T) {
    tests := []struct {
        name        string
        tier        Tier
        feature     Feature
        expectError bool
    }{
        {
            name:        "free tier can use spec",
            tier:        TierFree,
            feature:     FeatureSpec,
            expectError: false,
        },
        {
            name:        "free tier cannot use approvals",
            tier:        TierFree,
            feature:     FeatureApprovals,
            expectError: true,
        },
        {
            name:        "pro tier can use approvals",
            tier:        TierPro,
            feature:     FeatureApprovals,
            expectError: false,
        },
        {
            name:        "pro tier cannot use guardians",
            tier:        TierPro,
            feature:     FeatureGuardians,
            expectError: true,
        },
        {
            name:        "enterprise tier can use guardians",
            tier:        TierEnterprise,
            feature:     FeatureGuardians,
            expectError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            checker := &Checker{
                license: &License{
                    Tier: tt.tier,
                },
            }

            err := checker.RequireFeature(tt.feature)
            if tt.expectError && err == nil {
                t.Errorf("expected error but got none")
            }
            if !tt.expectError && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

### Integration Tests

```bash
# Test license loading from file
$ cat > ~/.specular/license.yaml <<EOF
version: "1.0"
tier: pro
org_id: test-org
org_name: Test Organization
issued_at: 2025-01-01T00:00:00Z
expires_at: 2026-01-01T00:00:00Z
EOF

$ specular license status
Specular License Status
═══════════════════════

Tier:         Pro
Organization: Test Organization
Expires:      2026-01-01 (365 days remaining)

Available Features:
  ✓ spec
  ✓ plan
  ✓ build
  ✓ eval
  ✓ drift
  ✓ team_routing
  ✓ approvals
  ✓ cloud_sync
```

---

## 5. User Experience Guidelines

### Clear Error Messages

```bash
# Bad (unclear)
$ specular approve
Error: feature not available

# Good (actionable)
$ specular approve
Error: feature "approvals" requires Pro tier (current: Free)

Approval workflows are a Pro feature.

Benefits of upgrading to Pro:
  • Team routing with shared knowledge
  • Approval workflows for governance
  • Cloud sync for specs and policies
  • Drift inbox with human-in-the-loop review

Pricing starts at $49/user/month.
Visit https://specular.dev/pricing to learn more.

Run 'specular license status' to see all available features.
```

### Helpful Commands

```bash
# Show what's available
$ specular license status

# See upgrade benefits
$ specular license upgrade --show-benefits

# Try Pro features (30-day trial)
$ specular license trial --tier pro
✓ Pro trial activated (30 days remaining)
Try: specular approve --help
```

---

## 6. Privacy & Offline-First

### Principles

1. **No Telemetry by Default**: License validation works offline
2. **Optional Cloud Sync**: Users opt-in to cloud features
3. **Local-First**: All core features work without internet
4. **Clear Data Usage**: Transparent about what data is sent to cloud

### Implementation

```go
// Config option
type LicenseConfig struct {
    // Local license validation (default)
    ValidationMode string `yaml:"validation_mode"` // "local" | "cloud" | "hybrid"

    // Opt-in cloud features
    CloudSyncEnabled bool `yaml:"cloud_sync_enabled"`

    // Privacy preferences
    TelemetryEnabled bool `yaml:"telemetry_enabled"`
}
```

---

## 7. Migration Path

### Free → Pro

1. Purchase Pro license from website
2. Receive license key via email
3. Run `specular license install <key>`
4. Pro features immediately available
5. No data migration required (local-first design)

### Pro → Enterprise

1. Contact sales for Enterprise license
2. Receive enterprise license key
3. Run `specular license install <enterprise-key>`
4. Enable SSO/RBAC via config
5. Optional: Setup private SpecHub

---

## 8. Timeline & Dependencies

### v1.4.0 (Q2 2025) - License Foundation
- Core license system (Phase 1)
- Feature gating for Pro features
- License management commands
- Local license files

**Dependencies:**
- ✅ v1.2.0 CLI Enhancement complete
- ⏳ v1.3.0 Governance Bundle (for attestations)

### v1.4.1 (Q2 2025) - JWT Validation
- Cryptographic license validation (Phase 2)
- License key generation service
- Trial period support

### v1.5.0 (Q3 2025) - Cloud Validation
- Optional cloud validation (Phase 3)
- License analytics dashboard
- Seat management for Enterprise

---

## 9. Open Questions

1. **License Key Format**: JWT vs custom format?
   - **Recommendation**: JWT for Pro, custom for Enterprise (more metadata)

2. **Trial Period**: Auto-downgrade or hard cutoff?
   - **Recommendation**: 7-day grace period, then feature lock (not data loss)

3. **Seat Counting**: Per-machine or per-user?
   - **Recommendation**: Per-user (email-based) for Pro, flexible for Enterprise

4. **Offline Grace**: How long can cloud licenses work offline?
   - **Recommendation**: 30 days cached validation for Enterprise

5. **License Transfer**: Can users move licenses between machines?
   - **Recommendation**: Yes, with deactivation command

---

## 10. Success Metrics

### Technical Metrics
- License validation latency < 100ms (local)
- License validation latency < 500ms (cloud)
- 99.9% uptime for license validation API
- Zero false negatives (legitimate licenses rejected)

### Business Metrics
- Conversion rate: Free → Pro
- Trial activation rate
- Churn rate (license expirations without renewal)
- Feature adoption rate (which Pro features are used most)

---

## Next Steps

When ready to implement:

1. Create `internal/license/` package structure
2. Implement Phase 1 (Core License System)
3. Add `license` subcommand to root command
4. Write comprehensive tests
5. Update documentation
6. Create license key generation tool (server-side)
7. Setup license validation API (optional, for Enterprise)

**Estimate:** 2-3 weeks for Phase 1 implementation
