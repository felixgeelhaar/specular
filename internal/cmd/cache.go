package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/cache"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var (
	cacheType     string
	cachePruneAge string
	cacheForce    bool
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage Specular cache",
	Long: `Manage the Specular CLI cache.

The cache stores downloaded models, build bundles, and execution traces
to improve performance and reduce repeated downloads.

Examples:
  # Show cache status and size
  specular cache info

  # List cached items
  specular cache list

  # Clear all cache
  specular cache clear

  # Clear only model cache
  specular cache clear --type models

  # Remove items older than 7 days
  specular cache prune --age 7d
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var cacheInfoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"status", "size"},
	Short:   "Show cache information",
	Long: `Display cache location, size, and breakdown by type.

Examples:
  specular cache info
  specular cache info --format json
`,
	RunE: runCacheInfo,
}

var cacheListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List cached items",
	Long: `List all items in the cache with size and age information.

Examples:
  # List all cached items
  specular cache list

  # List only model cache
  specular cache list --type models

  # List only bundle cache
  specular cache list --type bundles
`,
	RunE: runCacheList,
}

var cacheClearCmd = &cobra.Command{
	Use:     "clear",
	Aliases: []string{"clean", "rm"},
	Short:   "Clear the cache",
	Long: `Remove cached items to free up disk space.

By default, clears all cache types. Use --type to clear specific types.
Use --force to skip confirmation prompt.

Examples:
  # Clear all cache
  specular cache clear

  # Clear only models
  specular cache clear --type models

  # Clear without confirmation
  specular cache clear --force
`,
	RunE: runCacheClear,
}

var cachePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old cache entries",
	Long: `Remove cache entries older than a specified age.

Age format: 1d (days), 1w (weeks), 1h (hours)

Examples:
  # Remove entries older than 7 days
  specular cache prune --age 7d

  # Remove entries older than 1 week
  specular cache prune --age 1w
`,
	RunE: runCachePrune,
}

func init() {
	// Cache list flags
	cacheListCmd.Flags().StringVar(&cacheType, "type", "all", "Cache type (all, models, bundles, traces)")

	// Cache clear flags
	cacheClearCmd.Flags().StringVar(&cacheType, "type", "all", "Cache type (all, models, bundles, traces)")
	cacheClearCmd.Flags().BoolVarP(&cacheForce, "force", "f", false, "Skip confirmation prompt")

	// Cache prune flags
	cachePruneCmd.Flags().StringVar(&cachePruneAge, "age", "7d", "Remove entries older than this (e.g., 7d, 1w)")
	cachePruneCmd.Flags().BoolVarP(&cacheForce, "force", "f", false, "Skip confirmation prompt")

	// Add subcommands
	cacheCmd.AddCommand(cacheInfoCmd)
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cachePruneCmd)

	// Register at root
	rootCmd.AddCommand(cacheCmd)
}

func runCacheInfo(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	mgr, err := cache.NewManager()
	if err != nil {
		return ux.FormatError(err, "initializing cache manager")
	}

	info, err := mgr.GetInfo()
	if err != nil {
		return ux.FormatError(err, "getting cache info")
	}

	// JSON/YAML output
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(info)
	}

	// Text output
	fmt.Println()
	fmt.Println("Cache Information")
	fmt.Println("=================")
	fmt.Printf("Location:    %s\n", info.Location)
	fmt.Printf("Total Size:  %s\n", cache.FormatSize(info.TotalSize))
	fmt.Printf("Entry Count: %d\n", info.EntryCount)
	fmt.Println()

	if len(info.Types) > 0 {
		fmt.Println("Breakdown by Type:")
		for _, t := range info.Types {
			fmt.Printf("  %-10s %s (%d items)\n", t.Type, cache.FormatSize(t.Size), t.EntryCount)
		}
		fmt.Println()
	}

	return nil
}

func runCacheList(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	mgr, err := cache.NewManager()
	if err != nil {
		return ux.FormatError(err, "initializing cache manager")
	}

	ct := parseCacheType(cacheType)
	entries, err := mgr.List(ct)
	if err != nil {
		return ux.FormatError(err, "listing cache")
	}

	// JSON/YAML output
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(entries)
	}

	// Text output
	if len(entries) == 0 {
		fmt.Println("Cache is empty")
		return nil
	}

	fmt.Println()
	fmt.Printf("%-40s %-10s %-10s %s\n", "NAME", "TYPE", "SIZE", "AGE")
	fmt.Println(strings.Repeat("-", 80))

	for _, e := range entries {
		age := formatAge(time.Since(e.CreatedAt))
		name := truncate(e.Name, 40)
		fmt.Printf("%-40s %-10s %-10s %s\n", name, e.Type, cache.FormatSize(e.Size), age)
	}
	fmt.Println()

	return nil
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	mgr, err := cache.NewManager()
	if err != nil {
		return ux.FormatError(err, "initializing cache manager")
	}

	ct := parseCacheType(cacheType)

	// Get info before clearing
	info, _ := mgr.GetInfo()
	if info.TotalSize == 0 {
		fmt.Println("Cache is already empty")
		return nil
	}

	// Confirm if not forced
	if !cacheForce {
		fmt.Printf("This will clear %s of cached data.\n", cache.FormatSize(info.TotalSize))
		fmt.Print("Continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	result, err := mgr.Clear(ct)
	if err != nil {
		return ux.FormatError(err, "clearing cache")
	}

	fmt.Printf("Cleared %s (%d files)\n", cache.FormatSize(result.BytesCleared), result.FilesCleared)

	if len(result.Errors) > 0 {
		fmt.Println("\nWarnings:")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

func runCachePrune(cmd *cobra.Command, args []string) error {
	mgr, err := cache.NewManager()
	if err != nil {
		return ux.FormatError(err, "initializing cache manager")
	}

	// Parse age
	age, err := parseAge(cachePruneAge)
	if err != nil {
		return fmt.Errorf("invalid age format: %w", err)
	}

	// Get entries that would be pruned
	entries, _ := mgr.List(cache.CacheTypeAll)
	cutoff := time.Now().Add(-age)
	var toRemove int64
	var count int
	for _, e := range entries {
		if e.CreatedAt.Before(cutoff) {
			toRemove += e.Size
			count++
		}
	}

	if count == 0 {
		fmt.Printf("No cache entries older than %s\n", cachePruneAge)
		return nil
	}

	// Confirm if not forced
	if !cacheForce {
		fmt.Printf("This will remove %d entries (%s) older than %s.\n", count, cache.FormatSize(toRemove), cachePruneAge)
		fmt.Print("Continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	result, err := mgr.Prune(age)
	if err != nil {
		return ux.FormatError(err, "pruning cache")
	}

	fmt.Printf("Pruned %s (%d files)\n", cache.FormatSize(result.BytesPruned), result.FilesPruned)

	if len(result.Errors) > 0 {
		fmt.Println("\nWarnings:")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

// parseCacheType converts a string to CacheType
func parseCacheType(s string) cache.CacheType {
	switch strings.ToLower(s) {
	case "models":
		return cache.CacheTypeModels
	case "bundles":
		return cache.CacheTypeBundles
	case "traces":
		return cache.CacheTypeTraces
	default:
		return cache.CacheTypeAll
	}
}

// parseAge parses an age string like "7d" or "1w" to a duration
func parseAge(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid age format")
	}

	unit := s[len(s)-1]
	value := s[:len(s)-1]

	var n int
	if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
		return 0, err
	}

	switch unit {
	case 'h', 'H':
		return time.Duration(n) * time.Hour, nil
	case 'd', 'D':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w', 'W':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit: %c (use h, d, or w)", unit)
	}
}

// formatAge formats a duration as a human-readable age string
func formatAge(d time.Duration) string {
	switch {
	case d >= 7*24*time.Hour:
		weeks := int(d / (7 * 24 * time.Hour))
		return fmt.Sprintf("%dw ago", weeks)
	case d >= 24*time.Hour:
		days := int(d / (24 * time.Hour))
		return fmt.Sprintf("%dd ago", days)
	case d >= time.Hour:
		hours := int(d / time.Hour)
		return fmt.Sprintf("%dh ago", hours)
	case d >= time.Minute:
		minutes := int(d / time.Minute)
		return fmt.Sprintf("%dm ago", minutes)
	default:
		return "just now"
	}
}

// truncate truncates a string to a maximum length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// GetCacheTypes returns the list of supported cache types for shell completion
func GetCacheTypes() []string {
	return []string{"all", "models", "bundles", "traces"}
}
