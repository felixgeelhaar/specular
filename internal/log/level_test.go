package log

import (
	"log/slog"
	"testing"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(999), "UNKNOWN"}, // Invalid level
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("Level.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLevelToSlogLevel(t *testing.T) {
	tests := []struct {
		level Level
		want  slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarn, slog.LevelWarn},
		{LevelError, slog.LevelError},
		{Level(999), slog.LevelInfo}, // Invalid level defaults to Info
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			got := tt.level.ToSlogLevel()
			if got != tt.want {
				t.Errorf("Level.ToSlogLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"warning", LevelWarn},
		{"WARNING", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"invalid", LevelInfo}, // Invalid input defaults to Info
		{"", LevelInfo},        // Empty input defaults to Info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLevelRoundTrip(t *testing.T) {
	levels := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			str := level.String()
			parsed := ParseLevel(str)
			if parsed != level {
				t.Errorf("roundtrip failed: %v -> %q -> %v", level, str, parsed)
			}
		})
	}
}
