package slo

import (
	"fmt"
	"time"
)

// SLIValue represents a calculated SLI value at a point in time.
type SLIValue struct {
	// Value is the SLI ratio (0.0-1.0)
	Value float64 `json:"value"`

	// GoodEvents is the count of successful/good events
	GoodEvents float64 `json:"good_events"`

	// TotalEvents is the count of all events
	TotalEvents float64 `json:"total_events"`

	// BadEvents is the count of bad/failed events
	BadEvents float64 `json:"bad_events"`

	// Timestamp is when this SLI value was calculated
	Timestamp time.Time `json:"timestamp"`

	// Window is the time window over which this was calculated
	Window time.Duration `json:"window"`
}

// SLOStatus represents the current status of an SLO.
type SLOStatus struct {
	// SLO is the SLO definition
	SLO *SLO `json:"slo"`

	// Current is the current SLI value
	Current SLIValue `json:"current"`

	// BurnRate is the current error budget burn rate
	// A burn rate of 1.0 means consuming the error budget at the expected rate
	// A burn rate > 1.0 means consuming faster than expected
	BurnRate float64 `json:"burn_rate"`

	// ErrorBudgetRemaining is the remaining error budget as a ratio (0.0-1.0)
	ErrorBudgetRemaining float64 `json:"error_budget_remaining"`

	// ErrorBudgetConsumed is the consumed error budget as a ratio (0.0-1.0)
	ErrorBudgetConsumed float64 `json:"error_budget_consumed"`

	// IsHealthy indicates if the SLO is currently being met
	IsHealthy bool `json:"is_healthy"`

	// AlertFiring indicates if an alert should be firing
	AlertFiring bool `json:"alert_firing"`

	// StatusMessage provides a human-readable status
	StatusMessage string `json:"status_message"`

	// LastUpdated is when this status was last calculated
	LastUpdated time.Time `json:"last_updated"`
}

// CalculateSLI calculates the SLI value from good and total event counts.
func CalculateSLI(goodEvents, totalEvents float64, window time.Duration) SLIValue {
	var value float64
	if totalEvents > 0 {
		value = goodEvents / totalEvents
	}

	return SLIValue{
		Value:       value,
		GoodEvents:  goodEvents,
		TotalEvents: totalEvents,
		BadEvents:   totalEvents - goodEvents,
		Timestamp:   time.Now(),
		Window:      window,
	}
}

// CalculateSLIFromErrors calculates the SLI value from error and total event counts.
// This is useful when you have error counts rather than success counts.
func CalculateSLIFromErrors(errorEvents, totalEvents float64, window time.Duration) SLIValue {
	goodEvents := totalEvents - errorEvents
	return CalculateSLI(goodEvents, totalEvents, window)
}

// CalculateBurnRate calculates the error budget burn rate.
// A burn rate of 1.0 means the error budget is being consumed at exactly
// the expected rate (i.e., if this continues, the budget will be exhausted
// exactly at the end of the SLO window).
func CalculateBurnRate(slo *SLO, sli SLIValue) float64 {
	if sli.TotalEvents == 0 {
		return 0
	}

	// Current error rate
	errorRate := 1 - sli.Value

	// Allowed error rate (error budget)
	errorBudget := slo.ErrorBudget()

	if errorBudget == 0 {
		if errorRate > 0 {
			return 999999 // Infinite burn rate
		}
		return 0
	}

	return errorRate / errorBudget
}

// CalculateErrorBudgetConsumed calculates how much of the error budget has been consumed.
// Returns a value between 0.0 (no budget consumed) and 1.0+ (budget fully consumed or exceeded).
func CalculateErrorBudgetConsumed(slo *SLO, sli SLIValue) float64 {
	errorBudget := slo.ErrorBudget()
	if errorBudget == 0 {
		if sli.Value < 1.0 {
			return 1.0 // 100% consumed if any errors with 100% target
		}
		return 0
	}

	// How much of the window has elapsed
	windowProgress := sli.Window.Seconds() / slo.Window.Duration().Seconds()
	if windowProgress > 1.0 {
		windowProgress = 1.0
	}

	// Expected budget consumed based on time elapsed
	expectedConsumed := windowProgress

	// Actual budget consumed based on errors
	actualErrorRate := 1 - sli.Value
	actualConsumed := actualErrorRate / errorBudget

	// Return the ratio of actual to expected consumption
	if expectedConsumed == 0 {
		return actualConsumed
	}

	return actualConsumed
}

// CalculateErrorBudgetRemaining calculates the remaining error budget.
// Returns a value between 0.0 (no budget remaining) and 1.0 (full budget remaining).
// Can be negative if the budget is exceeded.
func CalculateErrorBudgetRemaining(slo *SLO, sli SLIValue) float64 {
	consumed := CalculateErrorBudgetConsumed(slo, sli)
	return 1.0 - consumed
}

// ShouldAlert determines if an alert should fire based on multi-window burn rate.
// This implements Google's SRE multi-window, multi-burn-rate alerting strategy.
func ShouldAlert(slo *SLO, shortWindowSLI, longWindowSLI SLIValue) bool {
	if slo.AlertPolicy == nil {
		return false
	}

	shortBurnRate := CalculateBurnRate(slo, shortWindowSLI)
	longBurnRate := CalculateBurnRate(slo, longWindowSLI)

	// Alert if both windows exceed the threshold
	return shortBurnRate >= slo.AlertPolicy.BurnRateThreshold &&
		longBurnRate >= slo.AlertPolicy.BurnRateThreshold
}

// CalculateStatus calculates the full SLO status.
func CalculateStatus(slo *SLO, sli SLIValue, shortWindowSLI, longWindowSLI *SLIValue) *SLOStatus {
	burnRate := CalculateBurnRate(slo, sli)
	budgetRemaining := CalculateErrorBudgetRemaining(slo, sli)
	budgetConsumed := CalculateErrorBudgetConsumed(slo, sli)
	isHealthy := sli.Value >= slo.Target

	var alertFiring bool
	if shortWindowSLI != nil && longWindowSLI != nil {
		alertFiring = ShouldAlert(slo, *shortWindowSLI, *longWindowSLI)
	}

	statusMessage := generateStatusMessage(slo, sli, burnRate, budgetRemaining, isHealthy, alertFiring)

	return &SLOStatus{
		SLO:                  slo,
		Current:              sli,
		BurnRate:             burnRate,
		ErrorBudgetRemaining: budgetRemaining,
		ErrorBudgetConsumed:  budgetConsumed,
		IsHealthy:            isHealthy,
		AlertFiring:          alertFiring,
		StatusMessage:        statusMessage,
		LastUpdated:          time.Now(),
	}
}

// generateStatusMessage creates a human-readable status message.
func generateStatusMessage(slo *SLO, sli SLIValue, burnRate, budgetRemaining float64, isHealthy, alertFiring bool) string {
	if alertFiring {
		return fmt.Sprintf("ALERT: %s is burning error budget at %.1fx rate (%.1f%% remaining)",
			slo.Name, burnRate, budgetRemaining*100)
	}

	if !isHealthy {
		return fmt.Sprintf("DEGRADED: %s is below target (%.2f%% vs %.2f%% target)",
			slo.Name, sli.Value*100, slo.Target*100)
	}

	if budgetRemaining < 0.2 {
		return fmt.Sprintf("WARNING: %s has low error budget (%.1f%% remaining)",
			slo.Name, budgetRemaining*100)
	}

	return fmt.Sprintf("HEALTHY: %s is meeting target (%.2f%% vs %.2f%% target)",
		slo.Name, sli.Value*100, slo.Target*100)
}

// TimeToExhaustion calculates how long until the error budget is exhausted
// at the current burn rate. Returns 0 if burn rate is 0 or negative.
func TimeToExhaustion(slo *SLO, sli SLIValue) time.Duration {
	burnRate := CalculateBurnRate(slo, sli)
	if burnRate <= 0 {
		return 0 // No errors or negative (shouldn't happen)
	}

	budgetRemaining := CalculateErrorBudgetRemaining(slo, sli)
	if budgetRemaining <= 0 {
		return 0 // Already exhausted
	}

	// Time remaining in the window
	windowRemaining := slo.Window.Duration() - sli.Window

	// At current burn rate, how long until budget is exhausted
	if burnRate >= 1.0 {
		// Burning faster than expected
		timeToExhaust := time.Duration(float64(windowRemaining) / burnRate)
		return timeToExhaust
	}

	// Burning slower than expected - won't exhaust in this window
	return windowRemaining
}

// ProjectedBudgetAtWindowEnd projects the remaining error budget at the end
// of the SLO window if the current burn rate continues.
func ProjectedBudgetAtWindowEnd(slo *SLO, sli SLIValue) float64 {
	burnRate := CalculateBurnRate(slo, sli)

	// How much of the window is remaining
	windowProgress := sli.Window.Seconds() / slo.Window.Duration().Seconds()
	windowRemaining := 1.0 - windowProgress

	// Current budget consumed
	currentConsumed := CalculateErrorBudgetConsumed(slo, sli)

	// Projected additional consumption
	additionalConsumption := burnRate * windowRemaining

	// Projected total consumption
	projectedConsumed := currentConsumed + additionalConsumption

	return 1.0 - projectedConsumed
}
