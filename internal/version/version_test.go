package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	origDate := Date

	// Set test values
	Version = "1.0.0"
	Commit = "abc123def456"
	Date = "2024-01-01T12:00:00Z"

	// Restore original values after test
	defer func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	}()

	info := GetInfo()

	// Verify version info
	if info.Version != "1.0.0" {
		t.Errorf("GetInfo().Version = %v, want 1.0.0", info.Version)
	}

	if info.Commit != "abc123def456" {
		t.Errorf("GetInfo().Commit = %v, want abc123def456", info.Commit)
	}

	if info.Date != "2024-01-01T12:00:00Z" {
		t.Errorf("GetInfo().Date = %v, want 2024-01-01T12:00:00Z", info.Date)
	}

	// Verify Go version matches runtime
	if info.GoVersion != runtime.Version() {
		t.Errorf("GetInfo().GoVersion = %v, want %v", info.GoVersion, runtime.Version())
	}

	// Verify platform is correct
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("GetInfo().Platform = %v, want %v", info.Platform, expectedPlatform)
	}
}

func TestInfoString(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want []string // Substrings that should be present
	}{
		{
			name: "full version info",
			info: Info{
				Version:   "1.0.0",
				Commit:    "abc123def456",
				Date:      "2024-01-01T12:00:00Z",
				GoVersion: "go1.22.0",
				Platform:  "linux/amd64",
			},
			want: []string{
				"Specular",
				"1.0.0",
				"abc123de", // Truncated commit
				"2024-01-01T12:00:00Z",
				"go1.22.0",
				"linux/amd64",
			},
		},
		{
			name: "short commit hash",
			info: Info{
				Version:   "1.0.0",
				Commit:    "abc123", // Less than 8 chars
				Date:      "2024-01-01",
				GoVersion: "go1.22.0",
				Platform:  "darwin/arm64",
			},
			want: []string{
				"Specular",
				"1.0.0",
				"abc123",
				"darwin/arm64",
			},
		},
		{
			name: "dev version",
			info: Info{
				Version:   "dev",
				Commit:    "unknown",
				Date:      "unknown",
				GoVersion: runtime.Version(),
				Platform:  runtime.GOOS + "/" + runtime.GOARCH,
			},
			want: []string{
				"Specular",
				"dev",
				"unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.String()

			for _, substr := range tt.want {
				if !strings.Contains(got, substr) {
					t.Errorf("Info.String() = %v, missing substring %v", got, substr)
				}
			}
		})
	}
}

func TestInfoShort(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want string
	}{
		{
			name: "release version",
			info: Info{Version: "1.0.0"},
			want: "1.0.0",
		},
		{
			name: "dev version",
			info: Info{Version: "dev"},
			want: "dev",
		},
		{
			name: "pre-release version",
			info: Info{Version: "1.0.0-rc1"},
			want: "1.0.0-rc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.Short()
			if got != tt.want {
				t.Errorf("Info.Short() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultValues(t *testing.T) {
	// Test that default values are set (when not overridden by ldflags)
	info := GetInfo()

	if info.Version == "" {
		t.Error("GetInfo().Version should have a default value")
	}

	if info.Commit == "" {
		t.Error("GetInfo().Commit should have a default value")
	}

	if info.Date == "" {
		t.Error("GetInfo().Date should have a default value")
	}

	if info.GoVersion == "" {
		t.Error("GetInfo().GoVersion should not be empty")
	}

	if info.Platform == "" {
		t.Error("GetInfo().Platform should not be empty")
	}
}

func TestInfoStringFormat(t *testing.T) {
	info := Info{
		Version:   "1.0.0",
		Commit:    "abc123def456",
		Date:      "2024-01-01",
		GoVersion: "go1.22.0",
		Platform:  "linux/amd64",
	}

	str := info.String()

	// Verify format: "Specular <version> (<commit>) built <date> with <go> for <platform>"
	parts := []string{
		"Specular",
		"(",
		")",
		"built",
		"with",
		"for",
	}

	for _, part := range parts {
		if !strings.Contains(str, part) {
			t.Errorf("Info.String() missing expected part: %v in %v", part, str)
		}
	}
}
