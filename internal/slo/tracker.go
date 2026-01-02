package slo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricsProvider is an interface for fetching metrics data.
// This allows the SLO tracker to work with different metrics backends.
type MetricsProvider interface {
	// Query executes a query and returns the result.
	Query(ctx context.Context, query string, time time.Time) (float64, error)

	// QueryRange executes a range query and returns aggregated results.
	QueryRange(ctx context.Context, query string, start, end time.Time) (float64, error)
}

// Tracker tracks SLOs and calculates their status.
type Tracker struct {
	mu       sync.RWMutex
	slos     map[string]*SLO
	provider MetricsProvider
	cache    map[string]*SLOStatus
	cacheTTL time.Duration
}

// TrackerOption configures the Tracker.
type TrackerOption func(*Tracker)

// WithMetricsProvider sets the metrics provider for the tracker.
func WithMetricsProvider(provider MetricsProvider) TrackerOption {
	return func(t *Tracker) {
		t.provider = provider
	}
}

// WithCacheTTL sets the cache TTL for SLO status.
func WithCacheTTL(ttl time.Duration) TrackerOption {
	return func(t *Tracker) {
		t.cacheTTL = ttl
	}
}

// NewTracker creates a new SLO tracker.
func NewTracker(opts ...TrackerOption) *Tracker {
	t := &Tracker{
		slos:     make(map[string]*SLO),
		cache:    make(map[string]*SLOStatus),
		cacheTTL: 1 * time.Minute, // Default cache TTL
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Register registers an SLO for tracking.
func (t *Tracker) Register(slo *SLO) error {
	if err := slo.Validate(); err != nil {
		return fmt.Errorf("invalid SLO: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.slos[slo.Name] = slo
	return nil
}

// RegisterAll registers multiple SLOs for tracking.
func (t *Tracker) RegisterAll(slos []*SLO) error {
	for _, slo := range slos {
		if err := t.Register(slo); err != nil {
			return err
		}
	}
	return nil
}

// Unregister removes an SLO from tracking.
func (t *Tracker) Unregister(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.slos, name)
	delete(t.cache, name)
}

// Get returns an SLO by name.
func (t *Tracker) Get(name string) (*SLO, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	slo, ok := t.slos[name]
	return slo, ok
}

// List returns all registered SLOs.
func (t *Tracker) List() []*SLO {
	t.mu.RLock()
	defer t.mu.RUnlock()

	slos := make([]*SLO, 0, len(t.slos))
	for _, slo := range t.slos {
		slos = append(slos, slo)
	}
	return slos
}

// Status returns the current status of an SLO.
// If a metrics provider is set, it will fetch real metrics.
// Otherwise, it returns a status based on cached or default values.
func (t *Tracker) Status(ctx context.Context, name string) (*SLOStatus, error) {
	t.mu.RLock()
	slo, ok := t.slos[name]
	cached, hasCached := t.cache[name]
	t.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("SLO not found: %s", name)
	}

	// Check cache
	if hasCached && time.Since(cached.LastUpdated) < t.cacheTTL {
		return cached, nil
	}

	// Calculate fresh status
	status, err := t.calculateStatus(ctx, slo)
	if err != nil {
		return nil, err
	}

	// Update cache
	t.mu.Lock()
	t.cache[name] = status
	t.mu.Unlock()

	return status, nil
}

// StatusAll returns the status of all registered SLOs.
func (t *Tracker) StatusAll(ctx context.Context) ([]*SLOStatus, error) {
	t.mu.RLock()
	sloNames := make([]string, 0, len(t.slos))
	for name := range t.slos {
		sloNames = append(sloNames, name)
	}
	t.mu.RUnlock()

	statuses := make([]*SLOStatus, 0, len(sloNames))
	for _, name := range sloNames {
		status, err := t.Status(ctx, name)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// calculateStatus calculates the SLO status.
func (t *Tracker) calculateStatus(ctx context.Context, slo *SLO) (*SLOStatus, error) {
	// If no metrics provider, return a placeholder status
	if t.provider == nil {
		return t.placeholderStatus(slo), nil
	}

	// Get current SLI value
	sli, err := t.fetchSLI(ctx, slo, slo.Window.Duration())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SLI for %s: %w", slo.Name, err)
	}

	// Get short and long window SLIs for alerting
	var shortWindowSLI, longWindowSLI *SLIValue
	if slo.AlertPolicy != nil {
		shortSLI, err := t.fetchSLI(ctx, slo, slo.AlertPolicy.ShortWindow.Duration())
		if err == nil {
			shortWindowSLI = &shortSLI
		}

		longSLI, err := t.fetchSLI(ctx, slo, slo.AlertPolicy.LongWindow.Duration())
		if err == nil {
			longWindowSLI = &longSLI
		}
	}

	return CalculateStatus(slo, sli, shortWindowSLI, longWindowSLI), nil
}

// fetchSLI fetches the SLI value from the metrics provider.
func (t *Tracker) fetchSLI(ctx context.Context, slo *SLO, window time.Duration) (SLIValue, error) {
	now := time.Now()
	start := now.Add(-window)

	var goodEvents, totalEvents float64
	var err error

	switch slo.SLI.Type {
	case SLITypeAvailability, SLITypeErrorRate:
		// For availability: good events are successful events
		goodQuery := fmt.Sprintf(`sum(increase(%s{success="true"}[%s]))`,
			slo.SLI.Metric, formatDuration(window))
		totalQuery := fmt.Sprintf(`sum(increase(%s[%s]))`,
			slo.SLI.Metric, formatDuration(window))

		goodEvents, err = t.provider.Query(ctx, goodQuery, now)
		if err != nil {
			return SLIValue{}, fmt.Errorf("failed to query good events: %w", err)
		}

		totalEvents, err = t.provider.Query(ctx, totalQuery, now)
		if err != nil {
			return SLIValue{}, fmt.Errorf("failed to query total events: %w", err)
		}

	case SLITypeLatency:
		// For latency: good events are requests within threshold
		threshold := slo.SLI.Threshold.Seconds()
		goodQuery := fmt.Sprintf(`sum(increase(%s_bucket{le="%g"}[%s]))`,
			slo.SLI.Metric, threshold, formatDuration(window))
		totalQuery := fmt.Sprintf(`sum(increase(%s_count[%s]))`,
			slo.SLI.Metric, formatDuration(window))

		goodEvents, err = t.provider.Query(ctx, goodQuery, now)
		if err != nil {
			return SLIValue{}, fmt.Errorf("failed to query good events: %w", err)
		}

		totalEvents, err = t.provider.Query(ctx, totalQuery, now)
		if err != nil {
			return SLIValue{}, fmt.Errorf("failed to query total events: %w", err)
		}

	case SLITypeCustom:
		goodEvents, err = t.provider.QueryRange(ctx, slo.SLI.GoodQuery, start, now)
		if err != nil {
			return SLIValue{}, fmt.Errorf("failed to query good events: %w", err)
		}

		totalEvents, err = t.provider.QueryRange(ctx, slo.SLI.TotalQuery, start, now)
		if err != nil {
			return SLIValue{}, fmt.Errorf("failed to query total events: %w", err)
		}

	default:
		return SLIValue{}, fmt.Errorf("unsupported SLI type: %s", slo.SLI.Type)
	}

	return CalculateSLI(goodEvents, totalEvents, window), nil
}

// placeholderStatus returns a placeholder status when no metrics provider is available.
func (t *Tracker) placeholderStatus(slo *SLO) *SLOStatus {
	return &SLOStatus{
		SLO: slo,
		Current: SLIValue{
			Value:       0,
			GoodEvents:  0,
			TotalEvents: 0,
			BadEvents:   0,
			Timestamp:   time.Now(),
			Window:      slo.Window.Duration(),
		},
		BurnRate:             0,
		ErrorBudgetRemaining: 1.0,
		ErrorBudgetConsumed:  0,
		IsHealthy:            true,
		AlertFiring:          false,
		StatusMessage:        fmt.Sprintf("NO DATA: %s has no metrics data available", slo.Name),
		LastUpdated:          time.Now(),
	}
}

// formatDuration formats a duration for PromQL queries.
func formatDuration(d time.Duration) string {
	if d >= 24*time.Hour {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
	if d >= time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%dh", hours)
	}
	if d >= time.Minute {
		minutes := int(d.Minutes())
		return fmt.Sprintf("%dm", minutes)
	}
	seconds := int(d.Seconds())
	return fmt.Sprintf("%ds", seconds)
}

// Summary provides a summary of all SLO statuses.
type Summary struct {
	TotalSLOs     int           `json:"total_slos"`
	HealthySLOs   int           `json:"healthy_slos"`
	DegradedSLOs  int           `json:"degraded_slos"`
	AlertingSLOs  int           `json:"alerting_slos"`
	Statuses      []*SLOStatus  `json:"statuses"`
	OverallHealth OverallHealth `json:"overall_health"`
	LastUpdated   time.Time     `json:"last_updated"`
}

// OverallHealth represents the overall health status.
type OverallHealth string

const (
	OverallHealthHealthy  OverallHealth = "healthy"
	OverallHealthDegraded OverallHealth = "degraded"
	OverallHealthCritical OverallHealth = "critical"
)

// GetSummary returns a summary of all SLO statuses.
func (t *Tracker) GetSummary(ctx context.Context) (*Summary, error) {
	statuses, err := t.StatusAll(ctx)
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		TotalSLOs:   len(statuses),
		Statuses:    statuses,
		LastUpdated: time.Now(),
	}

	for _, status := range statuses {
		if status.AlertFiring {
			summary.AlertingSLOs++
		} else if status.IsHealthy {
			summary.HealthySLOs++
		} else {
			summary.DegradedSLOs++
		}
	}

	// Determine overall health
	if summary.AlertingSLOs > 0 {
		summary.OverallHealth = OverallHealthCritical
	} else if summary.DegradedSLOs > 0 {
		summary.OverallHealth = OverallHealthDegraded
	} else {
		summary.OverallHealth = OverallHealthHealthy
	}

	return summary, nil
}

// ClearCache clears the status cache.
func (t *Tracker) ClearCache() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache = make(map[string]*SLOStatus)
}
