package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// setupTestCommand creates a command with the required flags for testing
func setupTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "test",
	}

	// Add all the persistent flags that NewCommandContext expects
	cmd.Flags().Bool("verbose", false, "verbose output")
	cmd.Flags().Bool("quiet", false, "quiet mode")
	cmd.Flags().String("format", "text", "output format")
	cmd.Flags().Bool("no-color", false, "disable colors")
	cmd.Flags().Bool("explain", false, "show reasoning")
	cmd.Flags().String("trace", "", "trace ID")
	cmd.Flags().String("home", "", "specular home")
	cmd.Flags().String("log-level", "info", "log level")

	return cmd
}

// TestNewCommandContext_Defaults tests default values
func TestNewCommandContext_Defaults(t *testing.T) {
	cmd := setupTestCommand()

	ctx, err := NewCommandContext(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Verbose {
		t.Error("Verbose should default to false")
	}

	if ctx.Quiet {
		t.Error("Quiet should default to false")
	}

	if ctx.Format != "text" {
		t.Errorf("Format = %q, want %q", ctx.Format, "text")
	}

	if ctx.NoColor {
		t.Error("NoColor should default to false")
	}

	if ctx.Explain {
		t.Error("Explain should default to false")
	}

	if ctx.Trace != "" {
		t.Errorf("Trace = %q, want empty", ctx.Trace)
	}

	if ctx.SpecularHome != "" {
		t.Errorf("SpecularHome = %q, want empty", ctx.SpecularHome)
	}

	if ctx.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", ctx.LogLevel, "info")
	}
}

// TestNewCommandContext_SetFlags tests that flags are properly extracted
func TestNewCommandContext_SetFlags(t *testing.T) {
	cmd := setupTestCommand()

	// Set all flags
	if err := cmd.Flags().Set("verbose", "true"); err != nil {
		t.Fatalf("failed to set verbose: %v", err)
	}
	if err := cmd.Flags().Set("quiet", "true"); err != nil {
		t.Fatalf("failed to set quiet: %v", err)
	}
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("failed to set format: %v", err)
	}
	if err := cmd.Flags().Set("no-color", "true"); err != nil {
		t.Fatalf("failed to set no-color: %v", err)
	}
	if err := cmd.Flags().Set("explain", "true"); err != nil {
		t.Fatalf("failed to set explain: %v", err)
	}
	if err := cmd.Flags().Set("trace", "trace-123"); err != nil {
		t.Fatalf("failed to set trace: %v", err)
	}
	if err := cmd.Flags().Set("home", "/custom/home"); err != nil {
		t.Fatalf("failed to set home: %v", err)
	}
	if err := cmd.Flags().Set("log-level", "debug"); err != nil {
		t.Fatalf("failed to set log-level: %v", err)
	}

	ctx, err := NewCommandContext(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ctx.Verbose {
		t.Error("Verbose should be true")
	}

	if !ctx.Quiet {
		t.Error("Quiet should be true")
	}

	if ctx.Format != "json" {
		t.Errorf("Format = %q, want %q", ctx.Format, "json")
	}

	if !ctx.NoColor {
		t.Error("NoColor should be true")
	}

	if !ctx.Explain {
		t.Error("Explain should be true")
	}

	if ctx.Trace != "trace-123" {
		t.Errorf("Trace = %q, want %q", ctx.Trace, "trace-123")
	}

	if ctx.SpecularHome != "/custom/home" {
		t.Errorf("SpecularHome = %q, want %q", ctx.SpecularHome, "/custom/home")
	}

	if ctx.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", ctx.LogLevel, "debug")
	}
}

// TestNewCommandContext_MissingFlag tests error handling for missing flags
func TestNewCommandContext_MissingFlag(t *testing.T) {
	tests := []struct {
		name        string
		missingFlag string
	}{
		{"missing_verbose", "verbose"},
		{"missing_quiet", "quiet"},
		{"missing_format", "format"},
		{"missing_no_color", "no-color"},
		{"missing_explain", "explain"},
		{"missing_trace", "trace"},
		{"missing_home", "home"},
		{"missing_log_level", "log-level"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a command missing the specific flag
			incompleteCmd := &cobra.Command{Use: "test"}

			// Add all flags except the one we're testing
			flags := map[string]struct {
				isBool   bool
				defValue string
			}{
				"verbose":   {true, "false"},
				"quiet":     {true, "false"},
				"format":    {false, "text"},
				"no-color":  {true, "false"},
				"explain":   {true, "false"},
				"trace":     {false, ""},
				"home":      {false, ""},
				"log-level": {false, "info"},
			}

			for flagName, info := range flags {
				if flagName != tc.missingFlag {
					if info.isBool {
						incompleteCmd.Flags().Bool(flagName, info.defValue == "true", "")
					} else {
						incompleteCmd.Flags().String(flagName, info.defValue, "")
					}
				}
			}

			// This should return an error because the flag is missing
			_, err := NewCommandContext(incompleteCmd)
			if err == nil {
				t.Errorf("expected error for missing %s flag", tc.missingFlag)
			}
		})
	}
}

// TestNewCommandContext_FormatValues tests various format values
func TestNewCommandContext_FormatValues(t *testing.T) {
	formats := []string{"text", "json", "yaml", "table"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cmd := setupTestCommand()
			if err := cmd.Flags().Set("format", format); err != nil {
				t.Fatalf("failed to set format: %v", err)
			}

			ctx, err := NewCommandContext(cmd)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ctx.Format != format {
				t.Errorf("Format = %q, want %q", ctx.Format, format)
			}
		})
	}
}

// TestNewCommandContext_LogLevelValues tests various log level values
func TestNewCommandContext_LogLevelValues(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cmd := setupTestCommand()
			if err := cmd.Flags().Set("log-level", level); err != nil {
				t.Fatalf("failed to set log-level: %v", err)
			}

			ctx, err := NewCommandContext(cmd)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ctx.LogLevel != level {
				t.Errorf("LogLevel = %q, want %q", ctx.LogLevel, level)
			}
		})
	}
}

// TestCommandContext_Struct tests the struct fields are properly typed
func TestCommandContext_Struct(t *testing.T) {
	ctx := &CommandContext{
		Verbose:      true,
		Quiet:        false,
		Format:       "json",
		NoColor:      true,
		Explain:      false,
		Trace:        "abc-123",
		SpecularHome: "/home/test/.specular",
		LogLevel:     "debug",
	}

	// Verify types by accessing fields
	var b bool = ctx.Verbose
	b = ctx.Quiet
	b = ctx.NoColor
	b = ctx.Explain

	var s string = ctx.Format
	s = ctx.Trace
	s = ctx.SpecularHome
	s = ctx.LogLevel

	// Avoid unused variable errors
	_ = b
	_ = s
}
