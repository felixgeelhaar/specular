package slo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateSLI(t *testing.T) {
	tests := []struct {
		name        string
		goodEvents  float64
		totalEvents float64
		window      time.Duration
		expected    float64
	}{
		{"100% success", 100, 100, time.Hour, 1.0},
		{"99% success", 99, 100, time.Hour, 0.99},
		{"50% success", 50, 100, time.Hour, 0.5},
		{"0% success", 0, 100, time.Hour, 0.0},
		{"no events", 0, 0, time.Hour, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sli := CalculateSLI(tt.goodEvents, tt.totalEvents, tt.window)
			assert.InDelta(t, tt.expected, sli.Value, 0.0001)
			assert.Equal(t, tt.goodEvents, sli.GoodEvents)
			assert.Equal(t, tt.totalEvents, sli.TotalEvents)
			assert.Equal(t, tt.totalEvents-tt.goodEvents, sli.BadEvents)
			assert.Equal(t, tt.window, sli.Window)
		})
	}
}

func TestCalculateSLIFromErrors(t *testing.T) {
	tests := []struct {
		name        string
		errorEvents float64
		totalEvents float64
		window      time.Duration
		expected    float64
	}{
		{"0% errors", 0, 100, time.Hour, 1.0},
		{"1% errors", 1, 100, time.Hour, 0.99},
		{"50% errors", 50, 100, time.Hour, 0.5},
		{"100% errors", 100, 100, time.Hour, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sli := CalculateSLIFromErrors(tt.errorEvents, tt.totalEvents, tt.window)
			assert.InDelta(t, tt.expected, sli.Value, 0.0001)
		})
	}
}

func TestCalculateBurnRate(t *testing.T) {
	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99, // 1% error budget
		Window: Duration(30 * 24 * time.Hour),
	}

	tests := []struct {
		name         string
		sli          SLIValue
		expectedRate float64
	}{
		{
			"no errors - zero burn rate",
			SLIValue{Value: 1.0, TotalEvents: 100},
			0,
		},
		{
			"exactly at error budget - burn rate 1.0",
			SLIValue{Value: 0.99, TotalEvents: 100},
			1.0,
		},
		{
			"double error rate - burn rate 2.0",
			SLIValue{Value: 0.98, TotalEvents: 100},
			2.0,
		},
		{
			"no events - zero burn rate",
			SLIValue{Value: 0, TotalEvents: 0},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			burnRate := CalculateBurnRate(slo, tt.sli)
			assert.InDelta(t, tt.expectedRate, burnRate, 0.0001)
		})
	}
}

func TestCalculateBurnRate_100PercentTarget(t *testing.T) {
	slo := &SLO{
		Name:   "100-percent-slo",
		Target: 1.0, // 0% error budget
		Window: Duration(30 * 24 * time.Hour),
	}

	t.Run("any errors with 100% target", func(t *testing.T) {
		sli := SLIValue{Value: 0.99, TotalEvents: 100}
		burnRate := CalculateBurnRate(slo, sli)
		assert.Equal(t, float64(999999), burnRate)
	})

	t.Run("no errors with 100% target", func(t *testing.T) {
		sli := SLIValue{Value: 1.0, TotalEvents: 100}
		burnRate := CalculateBurnRate(slo, sli)
		assert.Equal(t, float64(0), burnRate)
	})
}

func TestCalculateErrorBudgetConsumed(t *testing.T) {
	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99, // 1% error budget
		Window: Duration(30 * 24 * time.Hour),
	}

	tests := []struct {
		name             string
		sli              SLIValue
		expectedConsumed float64
	}{
		{
			"no errors consumed",
			SLIValue{Value: 1.0, TotalEvents: 100, Window: 30 * 24 * time.Hour},
			0,
		},
		{
			"exactly at budget - 100% consumed",
			SLIValue{Value: 0.99, TotalEvents: 100, Window: 30 * 24 * time.Hour},
			1.0,
		},
		{
			"double budget consumed - 200% consumed",
			SLIValue{Value: 0.98, TotalEvents: 100, Window: 30 * 24 * time.Hour},
			2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumed := CalculateErrorBudgetConsumed(slo, tt.sli)
			assert.InDelta(t, tt.expectedConsumed, consumed, 0.0001)
		})
	}
}

func TestCalculateErrorBudgetRemaining(t *testing.T) {
	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
	}

	tests := []struct {
		name              string
		sli               SLIValue
		expectedRemaining float64
	}{
		{
			"full budget remaining",
			SLIValue{Value: 1.0, TotalEvents: 100, Window: 30 * 24 * time.Hour},
			1.0,
		},
		{
			"no budget remaining",
			SLIValue{Value: 0.99, TotalEvents: 100, Window: 30 * 24 * time.Hour},
			0.0,
		},
		{
			"negative budget (exceeded)",
			SLIValue{Value: 0.98, TotalEvents: 100, Window: 30 * 24 * time.Hour},
			-1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := CalculateErrorBudgetRemaining(slo, tt.sli)
			assert.InDelta(t, tt.expectedRemaining, remaining, 0.0001)
		})
	}
}

func TestShouldAlert(t *testing.T) {
	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
		AlertPolicy: &AlertPolicy{
			BurnRateThreshold: 14.4,
			ShortWindow:       Duration(5 * time.Minute),
			LongWindow:        Duration(1 * time.Hour),
			Severity:          SeverityHigh,
		},
	}

	// For 14.4x burn rate with 1% error budget (0.01):
	// error_rate = burn_rate * error_budget = 14.4 * 0.01 = 0.144
	// SLI value = 1 - error_rate = 1 - 0.144 = 0.856
	// So SLI of 0.855 would give burn rate of (1-0.855)/0.01 = 14.5 (above threshold)
	tests := []struct {
		name        string
		shortSLI    SLIValue
		longSLI     SLIValue
		shouldAlert bool
	}{
		{
			"both windows below threshold",
			SLIValue{Value: 0.99, TotalEvents: 100},
			SLIValue{Value: 0.99, TotalEvents: 100},
			false,
		},
		{
			"both windows above threshold",
			SLIValue{Value: 0.85, TotalEvents: 100}, // 15% error = 15x burn rate > 14.4
			SLIValue{Value: 0.85, TotalEvents: 100},
			true,
		},
		{
			"short window above, long below",
			SLIValue{Value: 0.85, TotalEvents: 100},
			SLIValue{Value: 0.99, TotalEvents: 100},
			false,
		},
		{
			"short window below, long above",
			SLIValue{Value: 0.99, TotalEvents: 100},
			SLIValue{Value: 0.85, TotalEvents: 100},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldAlert(slo, tt.shortSLI, tt.longSLI)
			assert.Equal(t, tt.shouldAlert, result)
		})
	}
}

func TestShouldAlert_NoPolicy(t *testing.T) {
	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
		// No AlertPolicy
	}

	shortSLI := SLIValue{Value: 0.5, TotalEvents: 100}
	longSLI := SLIValue{Value: 0.5, TotalEvents: 100}

	result := ShouldAlert(slo, shortSLI, longSLI)
	assert.False(t, result, "should not alert when no policy is defined")
}

func TestCalculateStatus(t *testing.T) {
	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
		AlertPolicy: &AlertPolicy{
			BurnRateThreshold: 14.4,
			ShortWindow:       Duration(5 * time.Minute),
			LongWindow:        Duration(1 * time.Hour),
			Severity:          SeverityHigh,
		},
	}

	t.Run("healthy status", func(t *testing.T) {
		sli := SLIValue{Value: 0.995, TotalEvents: 100, Window: 30 * 24 * time.Hour}
		status := CalculateStatus(slo, sli, nil, nil)

		assert.True(t, status.IsHealthy)
		assert.False(t, status.AlertFiring)
		assert.Contains(t, status.StatusMessage, "HEALTHY")
	})

	t.Run("degraded status", func(t *testing.T) {
		sli := SLIValue{Value: 0.98, TotalEvents: 100, Window: 30 * 24 * time.Hour}
		status := CalculateStatus(slo, sli, nil, nil)

		assert.False(t, status.IsHealthy)
		assert.False(t, status.AlertFiring)
		assert.Contains(t, status.StatusMessage, "DEGRADED")
	})

	t.Run("alert firing status", func(t *testing.T) {
		// Use 0.85 which gives 15x burn rate > 14.4 threshold
		sli := SLIValue{Value: 0.85, TotalEvents: 100, Window: 30 * 24 * time.Hour}
		shortSLI := SLIValue{Value: 0.85, TotalEvents: 100}
		longSLI := SLIValue{Value: 0.85, TotalEvents: 100}
		status := CalculateStatus(slo, sli, &shortSLI, &longSLI)

		assert.False(t, status.IsHealthy)
		assert.True(t, status.AlertFiring)
		assert.Contains(t, status.StatusMessage, "ALERT")
	})

	t.Run("low budget warning", func(t *testing.T) {
		// Value that meets target but has consumed most of budget
		sli := SLIValue{Value: 0.992, TotalEvents: 100, Window: 30 * 24 * time.Hour}
		status := CalculateStatus(slo, sli, nil, nil)

		assert.True(t, status.IsHealthy)
		assert.Contains(t, status.StatusMessage, "WARNING")
	})
}

func TestTimeToExhaustion(t *testing.T) {
	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
	}

	t.Run("no burn rate - returns zero", func(t *testing.T) {
		sli := SLIValue{Value: 1.0, TotalEvents: 100, Window: 15 * 24 * time.Hour}
		tte := TimeToExhaustion(slo, sli)
		assert.Equal(t, time.Duration(0), tte)
	})

	t.Run("budget already exhausted", func(t *testing.T) {
		sli := SLIValue{Value: 0.98, TotalEvents: 100, Window: 30 * 24 * time.Hour}
		tte := TimeToExhaustion(slo, sli)
		assert.Equal(t, time.Duration(0), tte)
	})
}

func TestProjectedBudgetAtWindowEnd(t *testing.T) {
	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
	}

	t.Run("no errors - full budget projected", func(t *testing.T) {
		sli := SLIValue{Value: 1.0, TotalEvents: 100, Window: 15 * 24 * time.Hour}
		projected := ProjectedBudgetAtWindowEnd(slo, sli)
		assert.InDelta(t, 1.0, projected, 0.0001)
	})

	t.Run("burning at expected rate", func(t *testing.T) {
		// With SLI = 0.99 (exactly at target) halfway through the window:
		// - Current consumed = (1-0.99)/(1-0.99) = 1.0 (100% of allowed)
		// - Burn rate = 1.0
		// - Window remaining = 0.5
		// - Additional = 1.0 * 0.5 = 0.5
		// - Total consumed = 1.0 + 0.5 = 1.5
		// - Remaining = 1.0 - 1.5 = -0.5
		sli := SLIValue{Value: 0.99, TotalEvents: 100, Window: 15 * 24 * time.Hour}
		projected := ProjectedBudgetAtWindowEnd(slo, sli)
		assert.InDelta(t, -0.5, projected, 0.0001)
	})

	t.Run("on track - budget will be exhausted at window end", func(t *testing.T) {
		// If we're exactly on track (using 50% of budget in 50% of time)
		// with 50% of window elapsed and 0.5% error rate (half the 1% budget)
		// - Error rate = 0.005 (half of 1% budget)
		// - SLI = 1 - 0.005 = 0.995
		// - Current consumed = 0.005 / 0.01 = 0.5 (50% of budget)
		// - Burn rate = 0.5
		// - Window remaining = 0.5
		// - Additional = 0.5 * 0.5 = 0.25
		// - Total consumed = 0.5 + 0.25 = 0.75
		// - Remaining = 1.0 - 0.75 = 0.25
		sli := SLIValue{Value: 0.995, TotalEvents: 100, Window: 15 * 24 * time.Hour}
		projected := ProjectedBudgetAtWindowEnd(slo, sli)
		assert.InDelta(t, 0.25, projected, 0.0001)
	})
}

func TestGenerateStatusMessage(t *testing.T) {
	slo := &SLO{Name: "test-slo", Target: 0.99}

	tests := []struct {
		name            string
		sli             SLIValue
		burnRate        float64
		budgetRemaining float64
		isHealthy       bool
		alertFiring     bool
		expectedPrefix  string
	}{
		{
			"alert message",
			SLIValue{Value: 0.85},
			14.4,
			0.1,
			false,
			true,
			"ALERT:",
		},
		{
			"degraded message",
			SLIValue{Value: 0.95},
			2.0,
			0.5,
			false,
			false,
			"DEGRADED:",
		},
		{
			"warning message",
			SLIValue{Value: 0.995},
			0.5,
			0.1,
			true,
			false,
			"WARNING:",
		},
		{
			"healthy message",
			SLIValue{Value: 0.999},
			0.1,
			0.9,
			true,
			false,
			"HEALTHY:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := generateStatusMessage(slo, tt.sli, tt.burnRate, tt.budgetRemaining, tt.isHealthy, tt.alertFiring)
			assert.Contains(t, msg, tt.expectedPrefix)
		})
	}
}
