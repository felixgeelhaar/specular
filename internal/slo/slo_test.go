package slo

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSLIType_IsValid(t *testing.T) {
	tests := []struct {
		sliType SLIType
		valid   bool
	}{
		{SLITypeAvailability, true},
		{SLITypeLatency, true},
		{SLITypeErrorRate, true},
		{SLITypeThroughput, true},
		{SLITypeCustom, true},
		{SLIType("invalid"), false},
		{SLIType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.sliType), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.sliType.IsValid())
		})
	}
}

func TestSeverity_IsValid(t *testing.T) {
	tests := []struct {
		severity Severity
		valid    bool
	}{
		{SeverityCritical, true},
		{SeverityHigh, true},
		{SeverityWarning, true},
		{SeverityInfo, true},
		{Severity("invalid"), false},
		{Severity(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.severity.IsValid())
		})
	}
}

func TestDuration_ParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1h", time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"5s", 5 * time.Second, false},
		{"1h30m", 90 * time.Minute, false},
		{"", 0, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDuration_String(t *testing.T) {
	tests := []struct {
		duration Duration
		expected string
	}{
		{Duration(30 * 24 * time.Hour), "30d"},
		{Duration(7 * 24 * time.Hour), "7d"},
		{Duration(24 * time.Hour), "1d"},
		{Duration(time.Hour), "1h0m0s"},
		{Duration(30 * time.Minute), "30m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.duration.String())
		})
	}
}

func TestDuration_YAML(t *testing.T) {
	type testStruct struct {
		Window Duration `yaml:"window"`
	}

	t.Run("unmarshal", func(t *testing.T) {
		yaml := `window: 30d`
		var s testStruct
		err := yamlUnmarshal([]byte(yaml), &s)
		require.NoError(t, err)
		assert.Equal(t, Duration(30*24*time.Hour), s.Window)
	})

	t.Run("marshal", func(t *testing.T) {
		s := testStruct{Window: Duration(7 * 24 * time.Hour)}
		data, err := yamlMarshal(s)
		require.NoError(t, err)
		assert.Contains(t, string(data), "7d")
	})
}

func TestDuration_JSON(t *testing.T) {
	t.Run("unmarshal", func(t *testing.T) {
		jsonStr := `"30d"`
		var d Duration
		err := json.Unmarshal([]byte(jsonStr), &d)
		require.NoError(t, err)
		assert.Equal(t, Duration(30*24*time.Hour), d)
	})

	t.Run("marshal", func(t *testing.T) {
		d := Duration(7 * 24 * time.Hour)
		data, err := json.Marshal(d)
		require.NoError(t, err)
		assert.Equal(t, `"7d"`, string(data))
	})
}

func TestSLO_Validate(t *testing.T) {
	validSLO := &SLO{
		Name:        "test-slo",
		Description: "Test SLO",
		Target:      0.99,
		Window:      Duration(30 * 24 * time.Hour),
		SLI: SLISpec{
			Type:   SLITypeAvailability,
			Metric: "test_metric",
		},
	}

	t.Run("valid SLO", func(t *testing.T) {
		err := validSLO.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		slo := *validSLO
		slo.Name = ""
		err := slo.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("invalid target - zero", func(t *testing.T) {
		slo := *validSLO
		slo.Target = 0
		err := slo.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target must be between")
	})

	t.Run("invalid target - greater than 1", func(t *testing.T) {
		slo := *validSLO
		slo.Target = 1.5
		err := slo.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target must be between")
	})

	t.Run("invalid window", func(t *testing.T) {
		slo := *validSLO
		slo.Window = 0
		err := slo.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "window must be positive")
	})

	t.Run("invalid SLI type", func(t *testing.T) {
		slo := *validSLO
		slo.SLI.Type = "invalid"
		err := slo.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SLI type")
	})

	t.Run("latency SLI without threshold", func(t *testing.T) {
		slo := *validSLO
		slo.SLI.Type = SLITypeLatency
		slo.SLI.Threshold = 0
		err := slo.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "latency SLI requires a positive threshold")
	})

	t.Run("custom SLI without queries", func(t *testing.T) {
		slo := *validSLO
		slo.SLI.Type = SLITypeCustom
		err := slo.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "custom SLI requires")
	})

	t.Run("valid latency SLO", func(t *testing.T) {
		slo := *validSLO
		slo.SLI.Type = SLITypeLatency
		slo.SLI.Threshold = 500 * time.Millisecond
		err := slo.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid custom SLO", func(t *testing.T) {
		slo := *validSLO
		slo.SLI.Type = SLITypeCustom
		slo.SLI.GoodQuery = "good_query"
		slo.SLI.TotalQuery = "total_query"
		err := slo.Validate()
		assert.NoError(t, err)
	})
}

func TestAlertPolicy_Validate(t *testing.T) {
	validPolicy := &AlertPolicy{
		BurnRateThreshold: 14.4,
		ShortWindow:       Duration(5 * time.Minute),
		LongWindow:        Duration(1 * time.Hour),
		Severity:          SeverityHigh,
	}

	t.Run("valid policy", func(t *testing.T) {
		err := validPolicy.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid burn rate", func(t *testing.T) {
		p := *validPolicy
		p.BurnRateThreshold = 0
		err := p.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "burn rate threshold must be positive")
	})

	t.Run("invalid short window", func(t *testing.T) {
		p := *validPolicy
		p.ShortWindow = 0
		err := p.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "short window must be positive")
	})

	t.Run("short window longer than long window", func(t *testing.T) {
		p := *validPolicy
		p.ShortWindow = Duration(2 * time.Hour)
		err := p.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "short window must be shorter")
	})

	t.Run("invalid severity", func(t *testing.T) {
		p := *validPolicy
		p.Severity = "invalid"
		err := p.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid severity")
	})
}

func TestSLO_ErrorBudget(t *testing.T) {
	tests := []struct {
		target         float64
		expectedBudget float64
	}{
		{0.999, 0.001},
		{0.99, 0.01},
		{0.95, 0.05},
		{0.9, 0.1},
		{1.0, 0.0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			slo := &SLO{Target: tt.target}
			assert.InDelta(t, tt.expectedBudget, slo.ErrorBudget(), 0.0001)
		})
	}
}

func TestSLO_ErrorBudgetMinutes(t *testing.T) {
	slo := &SLO{
		Target: 0.99,                          // 99% target = 1% error budget
		Window: Duration(30 * 24 * time.Hour), // 30 days
	}

	// 30 days = 43200 minutes
	// 1% of 43200 = 432 minutes
	expected := 432.0
	assert.InDelta(t, expected, slo.ErrorBudgetMinutes(), 0.1)
}

func TestDefaultSLOs(t *testing.T) {
	slos := DefaultSLOs()

	assert.Len(t, slos, 4)

	// Validate all default SLOs
	for _, slo := range slos {
		err := slo.Validate()
		assert.NoError(t, err, "SLO %s should be valid", slo.Name)
	}

	// Check specific SLOs exist
	names := make([]string, len(slos))
	for i, slo := range slos {
		names[i] = slo.Name
	}

	assert.Contains(t, names, "command-success-rate")
	assert.Contains(t, names, "provider-latency-p95")
	assert.Contains(t, names, "auto-mode-success")
	assert.Contains(t, names, "plan-generation-latency")
}

// Helper functions for YAML marshaling in tests
func yamlUnmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}

func yamlMarshal(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}
