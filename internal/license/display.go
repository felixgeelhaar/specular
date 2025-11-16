package license

import (
	"fmt"
	"os"
)

// DisplayUpgradeMessage shows a formatted upgrade message when a feature is gated
func DisplayUpgradeMessage(err error, featureName string) {
	gateErr, ok := err.(*FeatureGateError)
	if !ok {
		return
	}

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "🔒 Feature requires %s tier\n", gateErr.RequiredTier)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "The '%s' feature requires Specular %s.\n", featureName, formatTier(gateErr.RequiredTier))
	fmt.Fprintf(os.Stderr, "Current tier: %s\n", formatTier(gateErr.CurrentTier))
	fmt.Fprintf(os.Stderr, "\n")

	// Show tier-specific benefits
	switch gateErr.RequiredTier {
	case TierPro:
		fmt.Fprintf(os.Stderr, "✨ Specular Pro includes:\n")
		fmt.Fprintf(os.Stderr, "  • Full governance features (init, doctor, status)\n")
		fmt.Fprintf(os.Stderr, "  • Policy management (init, validate, approve, diff)\n")
		fmt.Fprintf(os.Stderr, "  • Cryptographic approvals and attestations\n")
		fmt.Fprintf(os.Stderr, "  • Bundle governance gates with exit codes\n")
		fmt.Fprintf(os.Stderr, "  • Advanced hooks (webhooks, Slack integration)\n")
		fmt.Fprintf(os.Stderr, "  • Cloud provider support and intelligent routing\n")
		fmt.Fprintf(os.Stderr, "  • Security features (encryption, audit logs)\n")
		fmt.Fprintf(os.Stderr, "  • Advanced autonomous mode with checkpoints\n")
		fmt.Fprintf(os.Stderr, "\n")
	case TierEnterprise:
		fmt.Fprintf(os.Stderr, "🏢 Specular Enterprise includes:\n")
		fmt.Fprintf(os.Stderr, "  • All Pro features\n")
		fmt.Fprintf(os.Stderr, "  • Role-Based Access Control (RBAC)\n")
		fmt.Fprintf(os.Stderr, "  • Multi-tenancy support\n")
		fmt.Fprintf(os.Stderr, "  • SSO integration (SAML, OIDC)\n")
		fmt.Fprintf(os.Stderr, "  • SOC2 compliance reporting\n")
		fmt.Fprintf(os.Stderr, "  • Advanced policy engine with custom rules\n")
		fmt.Fprintf(os.Stderr, "  • Detailed audit exports\n")
		fmt.Fprintf(os.Stderr, "  • Vault integration for secrets\n")
		fmt.Fprintf(os.Stderr, "  • Prometheus/Grafana monitoring\n")
		fmt.Fprintf(os.Stderr, "  • Priority support with dedicated assistance\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	fmt.Fprintf(os.Stderr, "Learn more: https://specular.dev/pricing\n")
	fmt.Fprintf(os.Stderr, "\n")
}

// DisplayCurrentTier shows the current license tier
func DisplayCurrentTier() {
	tier, err := GetTier()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not determine license tier: %v\n", err)
		return
	}

	switch tier {
	case TierFree:
		fmt.Println("Current tier: Free")
		fmt.Println("\nUpgrade to Pro for governance features:")
		fmt.Println("  https://specular.dev/pricing")
	case TierPro:
		fmt.Println("Current tier: Pro ✅")
	case TierEnterprise:
		fmt.Println("Current tier: Enterprise 🏢")
	}
}

// DisplayFeatureMatrix shows available features by tier
func DisplayFeatureMatrix() {
	currentTier, _ := GetTier()

	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Specular Feature Matrix                   ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Current Tier: %-47s║\n", formatTier(currentTier))
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	printFeatureCategory("Core Features", []featureRow{
		{"Spec generation & locking", true, true, true},
		{"Plan creation & review", true, true, true},
		{"Basic bundle creation", true, true, true},
		{"Drift detection", true, true, true},
		{"Local providers (Ollama)", true, true, true},
		{"Basic autonomous mode", true, true, true},
	})

	printFeatureCategory("Governance (Pro)", []featureRow{
		{"Governance init/doctor/status", false, true, true},
		{"Policy management", false, true, true},
		{"Cryptographic approvals", false, true, true},
		{"Bundle governance gates", false, true, true},
		{"Attestations (ECDSA)", false, true, true},
	})

	printFeatureCategory("Integration (Pro)", []featureRow{
		{"Cloud providers", false, true, true},
		{"Intelligent routing", false, true, true},
		{"Webhooks", false, true, true},
		{"Slack integration", false, true, true},
		{"Security encryption", false, true, true},
		{"Audit logging", false, true, true},
	})

	printFeatureCategory("Enterprise", []featureRow{
		{"RBAC", false, false, true},
		{"Multi-tenancy", false, false, true},
		{"SSO (SAML/OIDC)", false, false, true},
		{"SOC2 compliance", false, false, true},
		{"Advanced policies", false, false, true},
		{"Vault integration", false, false, true},
		{"Priority support", false, false, true},
	})

	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

type featureRow struct {
	name       string
	free       bool
	pro        bool
	enterprise bool
}

func printFeatureCategory(category string, features []featureRow) {
	fmt.Printf("║ %-60s ║\n", category)
	fmt.Println("╟──────────────────────────────────────────────────────────────╢")

	for _, f := range features {
		freeSymbol := checkOrX(f.free)
		proSymbol := checkOrX(f.pro)
		entSymbol := checkOrX(f.enterprise)

		fmt.Printf("║ %-40s %2s %2s %2s        ║\n",
			f.name, freeSymbol, proSymbol, entSymbol)
	}

	fmt.Println("╟──────────────────────────────────────────────────────────────╢")
}

func checkOrX(available bool) string {
	if available {
		return "✓"
	}
	return "✗"
}

func formatTier(tier Tier) string {
	switch tier {
	case TierFree:
		return "Free"
	case TierPro:
		return "Pro"
	case TierEnterprise:
		return "Enterprise"
	default:
		return string(tier)
	}
}
