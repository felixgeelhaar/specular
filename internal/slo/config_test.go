package slo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.False(t, cfg.Enabled)
	assert.Equal(t, Duration(30*24*60*60*1e9), cfg.DefaultWindow)
	assert.Equal(t, Duration(60*1e9), cfg.CacheTTL)
	assert.Empty(t, cfg.SLOs)
	assert.Empty(t, cfg.ConfigPath)
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config with inline SLOs", func(t *testing.T) {
		cfg := &Config{
			Enabled: true,
			SLOs: []*SLO{
				{
					Name:   "test-slo",
					Target: 0.99,
					Window: Duration(30 * 24 * time.Hour),
					SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m"},
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid config with config path", func(t *testing.T) {
		cfg := &Config{
			Enabled:    true,
			ConfigPath: "/path/to/slos.yaml",
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid - both config path and inline SLOs", func(t *testing.T) {
		cfg := &Config{
			Enabled:    true,
			ConfigPath: "/path/to/slos.yaml",
			SLOs: []*SLO{
				{
					Name:   "test-slo",
					Target: 0.99,
					Window: Duration(30 * 24 * time.Hour),
					SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m"},
				},
			},
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify both")
	})

	t.Run("invalid inline SLO", func(t *testing.T) {
		cfg := &Config{
			Enabled: true,
			SLOs: []*SLO{
				{
					Name:   "", // Invalid - empty name
					Target: 0.99,
				},
			},
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SLO")
	})
}

func TestConfig_LoadSLOs(t *testing.T) {
	t.Run("load inline SLOs", func(t *testing.T) {
		inlineSLOs := []*SLO{
			{
				Name:   "slo-1",
				Target: 0.99,
				Window: Duration(time.Hour),
				SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m1"},
			},
			{
				Name:   "slo-2",
				Target: 0.95,
				Window: Duration(time.Hour),
				SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m2"},
			},
		}

		cfg := &Config{
			Enabled: true,
			SLOs:    inlineSLOs,
		}

		slos, err := cfg.LoadSLOs()
		require.NoError(t, err)
		assert.Len(t, slos, 2)
		assert.Equal(t, inlineSLOs, slos)
	})

	t.Run("load default SLOs when none specified", func(t *testing.T) {
		cfg := &Config{
			Enabled: true,
		}

		slos, err := cfg.LoadSLOs()
		require.NoError(t, err)
		assert.Len(t, slos, 4) // Default SLOs count
	})
}

func TestLoadSLOsFromFile(t *testing.T) {
	t.Run("load valid SLO file", func(t *testing.T) {
		content := `version: "1.0"
slos:
  - name: test-slo
    description: Test SLO
    target: 0.99
    window: 30d
    sli:
      type: availability
      metric: test_metric
`
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "slos.yaml")
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		slos, err := LoadSLOsFromFile(filePath)
		require.NoError(t, err)
		assert.Len(t, slos, 1)
		assert.Equal(t, "test-slo", slos[0].Name)
		assert.Equal(t, 0.99, slos[0].Target)
		assert.Equal(t, Duration(30*24*time.Hour), slos[0].Window)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadSLOsFromFile("/nonexistent/path/slos.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read SLO file")
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(filePath, []byte("invalid: [yaml"), 0644)
		require.NoError(t, err)

		_, err = LoadSLOsFromFile(filePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse SLO file")
	})

	t.Run("invalid SLO in file", func(t *testing.T) {
		content := `version: "1.0"
slos:
  - name: ""
    target: 0.99
    window: 30d
`
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "invalid-slo.yaml")
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadSLOsFromFile(filePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SLO")
	})
}

func TestSaveSLOsToFile(t *testing.T) {
	t.Run("save SLOs to file", func(t *testing.T) {
		slos := []*SLO{
			{
				Name:        "saved-slo",
				Description: "A saved SLO",
				Target:      0.995,
				Window:      Duration(7 * 24 * time.Hour),
				SLI: SLISpec{
					Type:   SLITypeAvailability,
					Metric: "saved_metric",
				},
			},
		}

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "saved-slos.yaml")

		err := SaveSLOsToFile(filePath, slos)
		require.NoError(t, err)

		// Read it back and verify
		loadedSLOs, err := LoadSLOsFromFile(filePath)
		require.NoError(t, err)
		assert.Len(t, loadedSLOs, 1)
		assert.Equal(t, "saved-slo", loadedSLOs[0].Name)
		assert.Equal(t, 0.995, loadedSLOs[0].Target)
	})

	t.Run("save to invalid path", func(t *testing.T) {
		slos := []*SLO{}
		err := SaveSLOsToFile("/nonexistent/directory/slos.yaml", slos)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write SLO file")
	})
}

func TestConfig_LoadSLOsFromFile(t *testing.T) {
	content := `version: "1.0"
slos:
  - name: file-slo
    description: SLO from file
    target: 0.999
    window: 7d
    sli:
      type: latency
      metric: file_metric
      threshold: 500ms
`
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "slos.yaml")
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	cfg := &Config{
		Enabled:    true,
		ConfigPath: filePath,
	}

	slos, err := cfg.LoadSLOs()
	require.NoError(t, err)
	assert.Len(t, slos, 1)
	assert.Equal(t, "file-slo", slos[0].Name)
}

func TestExampleSLOFile(t *testing.T) {
	example := ExampleSLOFile()

	assert.Contains(t, example, "version:")
	assert.Contains(t, example, "slos:")
	assert.Contains(t, example, "command-success-rate")
	assert.Contains(t, example, "provider-latency-p95")
	assert.Contains(t, example, "auto-mode-success")
	assert.Contains(t, example, "burn_rate_threshold")
	assert.Contains(t, example, "short_window")
	assert.Contains(t, example, "long_window")
}

func TestSLOFile_RoundTrip(t *testing.T) {
	// Test that SLOs can be saved and loaded without data loss
	originalSLOs := []*SLO{
		{
			Name:        "roundtrip-slo",
			Description: "Test round-trip serialization",
			Target:      0.9999,
			Window:      Duration(30 * 24 * time.Hour),
			SLI: SLISpec{
				Type:   SLITypeAvailability,
				Metric: "roundtrip_metric",
			},
			AlertPolicy: &AlertPolicy{
				BurnRateThreshold: 14.4,
				ShortWindow:       Duration(5 * time.Minute),
				LongWindow:        Duration(1 * time.Hour),
				Severity:          SeverityCritical,
			},
			Labels: map[string]string{
				"team":    "platform",
				"service": "test",
			},
		},
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "roundtrip.yaml")

	// Save
	err := SaveSLOsToFile(filePath, originalSLOs)
	require.NoError(t, err)

	// Load
	loadedSLOs, err := LoadSLOsFromFile(filePath)
	require.NoError(t, err)

	// Compare
	require.Len(t, loadedSLOs, 1)
	loaded := loadedSLOs[0]
	original := originalSLOs[0]

	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Description, loaded.Description)
	assert.Equal(t, original.Target, loaded.Target)
	assert.Equal(t, original.Window, loaded.Window)
	assert.Equal(t, original.SLI.Type, loaded.SLI.Type)
	assert.Equal(t, original.SLI.Metric, loaded.SLI.Metric)

	require.NotNil(t, loaded.AlertPolicy)
	assert.Equal(t, original.AlertPolicy.BurnRateThreshold, loaded.AlertPolicy.BurnRateThreshold)
	assert.Equal(t, original.AlertPolicy.ShortWindow, loaded.AlertPolicy.ShortWindow)
	assert.Equal(t, original.AlertPolicy.LongWindow, loaded.AlertPolicy.LongWindow)
	assert.Equal(t, original.AlertPolicy.Severity, loaded.AlertPolicy.Severity)

	assert.Equal(t, original.Labels, loaded.Labels)
}
