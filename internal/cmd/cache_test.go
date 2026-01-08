package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestCacheCommand(t *testing.T) {
	// Test that cache command is registered at root level
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "cache" {
			found = true
			break
		}
	}
	if !found {
		t.Error("cache command not registered under root")
	}
}

func TestCacheSubcommands(t *testing.T) {
	// Find the cache command
	var cacheCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "cache" {
			cacheCommand = c
			break
		}
	}
	if cacheCommand == nil {
		t.Fatal("cache command not found")
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
		"info":  false,
		"list":  false,
		"clear": false,
		"prune": false,
	}

	for _, sub := range cacheCommand.Commands() {
		if _, ok := expectedSubcommands[sub.Use]; ok {
			expectedSubcommands[sub.Use] = true
		}
	}

	for name, found := range expectedSubcommands {
		if !found {
			t.Errorf("Missing subcommand: %s", name)
		}
	}
}

func TestCacheInfoAliases(t *testing.T) {
	var cacheCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "cache" {
			cacheCommand = c
			break
		}
	}
	if cacheCommand == nil {
		t.Fatal("cache command not found")
	}

	var infoCmd *cobra.Command
	for _, sub := range cacheCommand.Commands() {
		if sub.Use == "info" {
			infoCmd = sub
			break
		}
	}
	if infoCmd == nil {
		t.Fatal("info subcommand not found")
	}

	// Check aliases
	expectedAliases := map[string]bool{
		"status": false,
		"size":   false,
	}

	for _, alias := range infoCmd.Aliases {
		if _, ok := expectedAliases[alias]; ok {
			expectedAliases[alias] = true
		}
	}

	for alias, found := range expectedAliases {
		if !found {
			t.Errorf("Missing alias for info: %s", alias)
		}
	}
}

func TestCacheListAliases(t *testing.T) {
	var cacheCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "cache" {
			cacheCommand = c
			break
		}
	}
	if cacheCommand == nil {
		t.Fatal("cache command not found")
	}

	var listCmd *cobra.Command
	for _, sub := range cacheCommand.Commands() {
		if sub.Use == "list" {
			listCmd = sub
			break
		}
	}
	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}

	// Check aliases
	hasLsAlias := false
	for _, alias := range listCmd.Aliases {
		if alias == "ls" {
			hasLsAlias = true
			break
		}
	}
	if !hasLsAlias {
		t.Error("list command should have 'ls' alias")
	}
}

func TestCacheClearFlags(t *testing.T) {
	var cacheCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "cache" {
			cacheCommand = c
			break
		}
	}
	if cacheCommand == nil {
		t.Fatal("cache command not found")
	}

	var clearCmd *cobra.Command
	for _, sub := range cacheCommand.Commands() {
		if sub.Use == "clear" {
			clearCmd = sub
			break
		}
	}
	if clearCmd == nil {
		t.Fatal("clear subcommand not found")
	}

	// Check flags
	if clearCmd.Flags().Lookup("type") == nil {
		t.Error("clear command missing --type flag")
	}
	if clearCmd.Flags().Lookup("force") == nil {
		t.Error("clear command missing --force flag")
	}
}

func TestCachePruneFlags(t *testing.T) {
	var cacheCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "cache" {
			cacheCommand = c
			break
		}
	}
	if cacheCommand == nil {
		t.Fatal("cache command not found")
	}

	var pruneCmd *cobra.Command
	for _, sub := range cacheCommand.Commands() {
		if sub.Use == "prune" {
			pruneCmd = sub
			break
		}
	}
	if pruneCmd == nil {
		t.Fatal("prune subcommand not found")
	}

	// Check flags
	if pruneCmd.Flags().Lookup("age") == nil {
		t.Error("prune command missing --age flag")
	}
	if pruneCmd.Flags().Lookup("force") == nil {
		t.Error("prune command missing --force flag")
	}
}

func TestParseCacheType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"models", "models"},
		{"Models", "models"},
		{"MODELS", "models"},
		{"bundles", "bundles"},
		{"traces", "traces"},
		{"all", "all"},
		{"unknown", "all"}, // defaults to all
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseCacheType(tt.input)
			if string(result) != tt.expected {
				t.Errorf("parseCacheType(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseAge(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"1h", time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"invalid", 0, true},
		{"1x", 0, true}, // unknown unit
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseAge(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseAge(%s) should error", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseAge(%s) error = %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("parseAge(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{2 * time.Hour, "2h ago"},
		{36 * time.Hour, "1d ago"},
		{10 * 24 * time.Hour, "1w ago"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatAge(tt.duration)
			if result != tt.expected {
				t.Errorf("formatAge(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a longer string", 10, "this is..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.max)
			if result != tt.expected {
				t.Errorf("truncate(%s, %d) = %s, want %s", tt.input, tt.max, result, tt.expected)
			}
		})
	}
}

func TestGetCacheTypes(t *testing.T) {
	types := GetCacheTypes()

	if len(types) != 4 {
		t.Errorf("GetCacheTypes() returned %d types, want 4", len(types))
	}

	expected := map[string]bool{
		"all":     false,
		"models":  false,
		"bundles": false,
		"traces":  false,
	}

	for _, ct := range types {
		if _, ok := expected[ct]; ok {
			expected[ct] = true
		}
	}

	for ct, found := range expected {
		if !found {
			t.Errorf("GetCacheTypes() missing %s", ct)
		}
	}
}

func TestCacheCommandDescription(t *testing.T) {
	var cacheCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "cache" {
			cacheCommand = c
			break
		}
	}
	if cacheCommand == nil {
		t.Fatal("cache command not found")
	}

	// Check Short description
	if !strings.Contains(cacheCommand.Short, "cache") {
		t.Error("Short description should mention cache")
	}

	// Check Long description has examples
	if !strings.Contains(cacheCommand.Long, "Examples:") {
		t.Error("Long description should include examples")
	}
}
