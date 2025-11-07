package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/felixgeelhaar/specular/internal/prd"
	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Specification management commands",
	Long:  `Generate, validate, and manage product specifications.`,
}

var specGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate spec from PRD markdown",
	Long:  `Convert a Product Requirements Document (PRD) in markdown format into a structured specification using AI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("in")
		out, _ := cmd.Flags().GetString("out")
		configPath, _ := cmd.Flags().GetString("config")

		fmt.Printf("Generating spec from PRD: %s\n", in)

		// Read PRD file
		prdContent, err := os.ReadFile(in)
		if err != nil {
			return fmt.Errorf("failed to read PRD file: %w", err)
		}

		// Load provider configuration
		if configPath == "" {
			configPath = ".specular/providers.yaml"
		}

		fmt.Println("Loading provider configuration...")
		registry, err := provider.LoadRegistryFromConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load providers: %w", err)
		}

		// Load provider config to get strategy settings
		providerConfig, err := provider.LoadProvidersConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load provider config: %w", err)
		}

		// Create router config from provider strategy
		routerConfig := &router.RouterConfig{
			BudgetUSD:    providerConfig.Strategy.Budget.MaxCostPerDay,
			MaxLatencyMs: providerConfig.Strategy.Performance.MaxLatencyMs,
			PreferCheap:  providerConfig.Strategy.Performance.PreferCheap,
		}

		// Set defaults if not specified
		if routerConfig.BudgetUSD == 0 {
			routerConfig.BudgetUSD = 20.0
		}
		if routerConfig.MaxLatencyMs == 0 {
			routerConfig.MaxLatencyMs = 60000
		}

		// Create router with providers
		r, err := router.NewRouterWithProviders(routerConfig, registry)
		if err != nil {
			return fmt.Errorf("failed to create router: %w", err)
		}

		// Create PRD parser (router handles provider access internally)
		parser := prd.NewParser(r)

		// Parse PRD to spec
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		fmt.Println("Parsing PRD with AI (this may take 30-60 seconds)...")
		productSpec, err := parser.ParsePRD(ctx, string(prdContent))
		if err != nil {
			return fmt.Errorf("failed to parse PRD: %w", err)
		}

		// Save spec
		if err := spec.SaveSpec(productSpec, out); err != nil {
			return fmt.Errorf("failed to save spec: %w", err)
		}

		fmt.Printf("✓ Generated spec with %d features\n", len(productSpec.Features))
		fmt.Printf("✓ Saved to: %s\n", out)

		return nil
	},
}

var specValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a specification file",
	Long:  `Validate a specification against the schema and semantic rules.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("in")

		// Load spec
		s, err := spec.LoadSpec(in)
		if err != nil {
			return fmt.Errorf("failed to load spec: %w", err)
		}

		// Basic validation
		if s.Product == "" {
			return fmt.Errorf("validation failed: product name is required")
		}

		if len(s.Features) == 0 {
			return fmt.Errorf("validation failed: at least one feature is required")
		}

		// Validate each feature
		for _, feature := range s.Features {
			if feature.ID == "" {
				return fmt.Errorf("validation failed: feature missing ID")
			}
			if feature.Title == "" {
				return fmt.Errorf("validation failed: feature %s missing title", feature.ID)
			}
			if feature.Priority == "" {
				return fmt.Errorf("validation failed: feature %s missing priority", feature.ID)
			}
			if feature.Priority != "P0" && feature.Priority != "P1" && feature.Priority != "P2" {
				return fmt.Errorf("validation failed: feature %s has invalid priority %s (must be P0, P1, or P2)",
					feature.ID, feature.Priority)
			}
		}

		fmt.Printf("✓ Spec is valid (%d features)\n", len(s.Features))
		return nil
	},
}

var specLockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Generate SpecLock from specification",
	Long:  `Create a canonical, hashed SpecLock file with blake3 hashes for drift detection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("in")
		out, _ := cmd.Flags().GetString("out")
		version, _ := cmd.Flags().GetString("version")

		// Load spec
		s, err := spec.LoadSpec(in)
		if err != nil {
			return fmt.Errorf("failed to load spec: %w", err)
		}

		// Generate SpecLock
		lock, err := spec.GenerateSpecLock(*s, version)
		if err != nil {
			return fmt.Errorf("failed to generate SpecLock: %w", err)
		}

		// Save SpecLock
		if err := spec.SaveSpecLock(lock, out); err != nil {
			return fmt.Errorf("failed to save SpecLock: %w", err)
		}

		fmt.Printf("✓ Generated SpecLock with %d features\n", len(lock.Features))
		for featureID, lockedFeature := range lock.Features {
			fmt.Printf("  %s: %s\n", featureID, lockedFeature.Hash[:16]+"...")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(specCmd)
	specCmd.AddCommand(specGenerateCmd)
	specCmd.AddCommand(specValidateCmd)
	specCmd.AddCommand(specLockCmd)

	specGenerateCmd.Flags().StringP("in", "i", "PRD.md", "Input PRD file")
	specGenerateCmd.Flags().StringP("out", "o", ".specular/spec.yaml", "Output spec file")
	specGenerateCmd.Flags().String("config", ".specular/providers.yaml", "Provider configuration file")

	specValidateCmd.Flags().StringP("in", "i", ".specular/spec.yaml", "Spec file to validate")

	specLockCmd.Flags().StringP("in", "i", ".specular/spec.yaml", "Input spec file")
	specLockCmd.Flags().StringP("out", "o", ".specular/spec.lock.json", "Output SpecLock file")
	specLockCmd.Flags().String("version", "1.0", "SpecLock version")
}
