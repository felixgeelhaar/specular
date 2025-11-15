package log

import (
	"bytes"
	"os"
	"testing"
)

func TestFormatString(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatJSON, "json"},
		{FormatText, "text"},
		{Format(999), "json"}, // Invalid format defaults to json
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.format.String()
			if got != tt.want {
				t.Errorf("Format.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  Format
	}{
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"text", FormatText},
		{"TEXT", FormatText},
		{"console", FormatText},
		{"invalid", FormatJSON}, // Invalid input defaults to JSON
		{"", FormatJSON},        // Empty input defaults to JSON
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseFormat(tt.input)
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatRoundTrip(t *testing.T) {
	formats := []Format{FormatJSON, FormatText}

	for _, format := range formats {
		t.Run(format.String(), func(t *testing.T) {
			str := format.String()
			parsed := ParseFormat(str)
			if parsed != format {
				t.Errorf("roundtrip failed: %v -> %q -> %v", format, str, parsed)
			}
		})
	}
}

func TestNewOutput(t *testing.T) {
	var buf bytes.Buffer
	output := NewOutput(&buf)

	if output.Writer() != &buf {
		t.Error("NewOutput did not return the correct writer")
	}
}

func TestOutputStdout(t *testing.T) {
	output := OutputStdout()
	if output.Writer() != os.Stdout {
		t.Error("OutputStdout did not return stdout")
	}
}

func TestOutputStderr(t *testing.T) {
	output := OutputStderr()
	if output.Writer() != os.Stderr {
		t.Error("OutputStderr did not return stderr")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Level != LevelInfo {
		t.Errorf("DefaultConfig.Level = %v, want %v", config.Level, LevelInfo)
	}
	if config.Format != FormatJSON {
		t.Errorf("DefaultConfig.Format = %v, want %v", config.Format, FormatJSON)
	}
	if config.Output.Writer() != os.Stdout {
		t.Error("DefaultConfig.Output should be stdout")
	}
	if config.AddSource {
		t.Error("DefaultConfig.AddSource should be false")
	}
	if config.ServiceName != "specular" {
		t.Errorf("DefaultConfig.ServiceName = %q, want %q", config.ServiceName, "specular")
	}
	if config.ServiceVersion != "dev" {
		t.Errorf("DefaultConfig.ServiceVersion = %q, want %q", config.ServiceVersion, "dev")
	}
}

func TestDevelopmentConfig(t *testing.T) {
	config := DevelopmentConfig()

	if config.Level != LevelDebug {
		t.Errorf("DevelopmentConfig.Level = %v, want %v", config.Level, LevelDebug)
	}
	if config.Format != FormatText {
		t.Errorf("DevelopmentConfig.Format = %v, want %v", config.Format, FormatText)
	}
	if config.Output.Writer() != os.Stdout {
		t.Error("DevelopmentConfig.Output should be stdout")
	}
	if !config.AddSource {
		t.Error("DevelopmentConfig.AddSource should be true")
	}
	if config.ServiceName != "specular" {
		t.Errorf("DevelopmentConfig.ServiceName = %q, want %q", config.ServiceName, "specular")
	}
	if config.ServiceVersion != "dev" {
		t.Errorf("DevelopmentConfig.ServiceVersion = %q, want %q", config.ServiceVersion, "dev")
	}
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()

	if config.Level != LevelInfo {
		t.Errorf("ProductionConfig.Level = %v, want %v", config.Level, LevelInfo)
	}
	if config.Format != FormatJSON {
		t.Errorf("ProductionConfig.Format = %v, want %v", config.Format, FormatJSON)
	}
	if config.Output.Writer() != os.Stdout {
		t.Error("ProductionConfig.Output should be stdout")
	}
	if config.AddSource {
		t.Error("ProductionConfig.AddSource should be false")
	}
	if config.ServiceName != "specular" {
		t.Errorf("ProductionConfig.ServiceName = %q, want %q", config.ServiceName, "specular")
	}
	if config.ServiceVersion != "unknown" {
		t.Errorf("ProductionConfig.ServiceVersion = %q, want %q", config.ServiceVersion, "unknown")
	}
}

func TestConfigDefaults(t *testing.T) {
	// Verify that all config constructors return valid configs
	configs := []struct {
		name   string
		config Config
	}{
		{"default", DefaultConfig()},
		{"development", DevelopmentConfig()},
		{"production", ProductionConfig()},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.config.Output.Writer() == nil {
				t.Error("config.Output.Writer() should not be nil")
			}
			if tc.config.ServiceName == "" {
				t.Error("config.ServiceName should not be empty")
			}
			if tc.config.ServiceVersion == "" {
				t.Error("config.ServiceVersion should not be empty")
			}
		})
	}
}
