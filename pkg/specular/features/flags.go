package features

import "sync"

// Flag represents a feature flag that can be enabled or disabled
type Flag string

const (
	// FlagBasicSpec enables basic spec generation (free tier feature, always enabled in public builds)
	FlagBasicSpec Flag = "basic_spec"
	// FlagBasicPlan enables basic plan generation
	FlagBasicPlan Flag = "basic_plan"
	// FlagLocalExec enables local Docker execution
	FlagLocalExec Flag = "local_exec"
	// FlagBasicPolicy enables basic policy enforcement
	FlagBasicPolicy Flag = "basic_policy"
	// FlagCheckpoint enables checkpoint/resume functionality
	FlagCheckpoint Flag = "checkpoint"
	// FlagPatchGeneration enables patch generation
	FlagPatchGeneration Flag = "patch_generation"

	// FlagMultiTenancy enables multi-tenant architecture (enterprise feature, disabled in public builds)
	FlagMultiTenancy Flag = "multi_tenancy"
	// FlagSSOAuth enables SSO/SAML authentication
	FlagSSOAuth Flag = "sso_auth"
	// FlagRBAC enables role-based access control
	FlagRBAC Flag = "rbac"
	// FlagWebDashboard enables web UI dashboard
	FlagWebDashboard Flag = "web_dashboard"
	// FlagPlatformAPI enables platform API server
	FlagPlatformAPI Flag = "platform_api"
	// FlagAdvancedPolicy enables enterprise policy engine v2
	FlagAdvancedPolicy Flag = "advanced_policy"
	// FlagIntegrations enables enterprise integrations (Slack, Jira, etc.)
	FlagIntegrations Flag = "integrations"
	// FlagAnalytics enables usage analytics and reporting
	FlagAnalytics Flag = "analytics"
	// FlagAuditLog enables comprehensive audit logging
	FlagAuditLog Flag = "audit_log"
	// FlagHighAvailability enables HA and disaster recovery
	FlagHighAvailability Flag = "high_availability"
)

var (
	// globalFlags holds the current feature flag state
	globalFlags = &flagState{
		flags: make(map[Flag]bool),
	}
)

// flagState holds the enabled/disabled state of all features
type flagState struct {
	mu    sync.RWMutex
	flags map[Flag]bool
}

// IsEnabled checks if a feature flag is enabled
func IsEnabled(flag Flag) bool {
	globalFlags.mu.RLock()
	defer globalFlags.mu.RUnlock()

	// If not explicitly set, check defaults
	enabled, ok := globalFlags.flags[flag]
	if !ok {
		return isEnabledByDefault(flag)
	}
	return enabled
}

// Enable enables a feature flag
func Enable(flag Flag) {
	globalFlags.mu.Lock()
	defer globalFlags.mu.Unlock()
	globalFlags.flags[flag] = true
}

// Disable disables a feature flag
func Disable(flag Flag) {
	globalFlags.mu.Lock()
	defer globalFlags.mu.Unlock()
	globalFlags.flags[flag] = false
}

// Reset resets all feature flags to their defaults
func Reset() {
	globalFlags.mu.Lock()
	defer globalFlags.mu.Unlock()
	globalFlags.flags = make(map[Flag]bool)
}

// isEnabledByDefault returns the default state of a feature flag
// Free tier features are enabled by default, enterprise features are disabled
func isEnabledByDefault(flag Flag) bool {
	switch flag {
	// Free tier features - always enabled by default
	case FlagBasicSpec,
		FlagBasicPlan,
		FlagLocalExec,
		FlagBasicPolicy,
		FlagCheckpoint,
		FlagPatchGeneration:
		return true

	// Enterprise features - disabled by default in public builds
	case FlagMultiTenancy,
		FlagSSOAuth,
		FlagRBAC,
		FlagWebDashboard,
		FlagPlatformAPI,
		FlagAdvancedPolicy,
		FlagIntegrations,
		FlagAnalytics,
		FlagAuditLog,
		FlagHighAvailability:
		return false

	default:
		// Unknown flags are disabled by default
		return false
	}
}

// GetAllFlags returns the current state of all feature flags
func GetAllFlags() map[Flag]bool {
	globalFlags.mu.RLock()
	defer globalFlags.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[Flag]bool)

	// Add all known flags with their current state
	allFlags := []Flag{
		FlagBasicSpec, FlagBasicPlan, FlagLocalExec, FlagBasicPolicy,
		FlagCheckpoint, FlagPatchGeneration, FlagMultiTenancy, FlagSSOAuth,
		FlagRBAC, FlagWebDashboard, FlagPlatformAPI, FlagAdvancedPolicy,
		FlagIntegrations, FlagAnalytics, FlagAuditLog, FlagHighAvailability,
	}

	for _, flag := range allFlags {
		enabled, ok := globalFlags.flags[flag]
		if !ok {
			result[flag] = isEnabledByDefault(flag)
		} else {
			result[flag] = enabled
		}
	}

	return result
}

// Edition represents the Specular edition (Free or Enterprise)
type Edition string

const (
	// EditionFree represents the free CLI edition
	EditionFree Edition = "free"
	// EditionEnterprise represents the enterprise platform edition
	EditionEnterprise Edition = "enterprise"
)

// currentEdition stores the current edition
var currentEdition = EditionFree

// SetEdition sets the current edition and enables/disables features accordingly
func SetEdition(edition Edition) {
	currentEdition = edition

	if edition == EditionEnterprise {
		// Enable all enterprise features
		Enable(FlagMultiTenancy)
		Enable(FlagSSOAuth)
		Enable(FlagRBAC)
		Enable(FlagWebDashboard)
		Enable(FlagPlatformAPI)
		Enable(FlagAdvancedPolicy)
		Enable(FlagIntegrations)
		Enable(FlagAnalytics)
		Enable(FlagAuditLog)
		Enable(FlagHighAvailability)
	}
}

// GetEdition returns the current edition
func GetEdition() Edition {
	return currentEdition
}
