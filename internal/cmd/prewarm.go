package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/plan"
)

var prewarmCmd = &cobra.Command{
	Use:   "prewarm",
	Short: "Pre-warm Docker image cache",
	Long: `Pre-warm the Docker image cache by pulling commonly used images.

This command pulls Docker images in parallel to speed up subsequent builds.
Use this command before running builds in CI/CD to cache images.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planFile := cmd.Flags().Lookup("plan").Value.String()
		cacheDir := cmd.Flags().Lookup("cache-dir").Value.String()
		concurrencyStr := cmd.Flags().Lookup("concurrency").Value.String()
		concurrency, _ := strconv.Atoi(concurrencyStr) //nolint:errcheck // Has default value
		maxAgeStr := cmd.Flags().Lookup("max-age").Value.String()
		maxAge, _ := time.ParseDuration(maxAgeStr) //nolint:errcheck // Has default value
		exportDir := cmd.Flags().Lookup("export").Value.String()
		importDir := cmd.Flags().Lookup("import").Value.String()
		prune := cmd.Flags().Lookup("prune").Value.String() == "true"
		verbose := cmd.Flags().Lookup("verbose").Value.String() == "true"
		all := cmd.Flags().Lookup("all").Value.String() == "true"

		// Create cache manager
		cache := exec.NewImageCache(cacheDir, maxAge)

		// Load existing manifest
		if err := cache.LoadManifest(); err != nil {
			fmt.Printf("Warning: failed to load cache manifest: %v\n", err)
		}

		// Import cached images if requested
		if importDir != "" {
			if err := cache.ImportImages(importDir, verbose); err != nil {
				return fmt.Errorf("import images: %w", err)
			}
		}

		// Prune old images if requested
		if prune {
			if err := cache.PruneCache(maxAge, verbose); err != nil {
				return fmt.Errorf("prune cache: %w", err)
			}
		}

		// Determine which images to pre-warm
		var images []string

		if all {
			// Pre-warm all common images
			images = []string{
				"golang:1.22",
				"node:20",
				"alpine:latest",
				"postgres:15",
			}
		} else if planFile != "" {
			// Extract images from plan
			p, err := plan.LoadPlan(planFile)
			if err != nil {
				return fmt.Errorf("load plan: %w", err)
			}

			// Convert plan tasks to skill tasks
			skillTasks := make([]struct{ Skill string }, len(p.Tasks))
			for i, task := range p.Tasks {
				skillTasks[i].Skill = task.Skill
			}

			images = exec.GetRequiredImages(skillTasks)
		} else if len(args) > 0 {
			// Use images from command line args
			images = args
		} else {
			// Default to common images
			images = []string{
				"golang:1.22",
				"node:20",
				"alpine:latest",
			}
		}

		// Pre-warm images
		if len(images) > 0 {
			if err := cache.PrewarmImages(images, concurrency, verbose); err != nil {
				return fmt.Errorf("prewarm images: %w", err)
			}
		}

		// Export images if requested
		if exportDir != "" {
			if err := cache.ExportImages(images, exportDir, verbose); err != nil {
				return fmt.Errorf("export images: %w", err)
			}
		}

		// Print cache stats
		if verbose {
			fmt.Println("\nCache Statistics:")
			stats := cache.GetStats()
			for key, value := range stats {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(prewarmCmd)

	prewarmCmd.Flags().String("plan", "", "Plan file to extract required images from")
	prewarmCmd.Flags().String("cache-dir", ".specular/cache", "Cache directory")
	prewarmCmd.Flags().Int("concurrency", 3, "Number of concurrent image pulls")
	prewarmCmd.Flags().Duration("max-age", 7*24*time.Hour, "Maximum cache age (e.g., 7d, 24h)")
	prewarmCmd.Flags().String("export", "", "Export images to directory (for CI/CD caching)")
	prewarmCmd.Flags().String("import", "", "Import images from directory (for CI/CD caching)")
	prewarmCmd.Flags().Bool("prune", false, "Prune old cached images")
	prewarmCmd.Flags().Bool("verbose", false, "Verbose output")
	prewarmCmd.Flags().Bool("all", false, "Pre-warm all common images")
}
