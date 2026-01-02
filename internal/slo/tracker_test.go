package slo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMetricsProvider is a mock implementation of MetricsProvider for testing.
type MockMetricsProvider struct {
	QueryFunc      func(ctx context.Context, query string, t time.Time) (float64, error)
	QueryRangeFunc func(ctx context.Context, query string, start, end time.Time) (float64, error)
}

func (m *MockMetricsProvider) Query(ctx context.Context, query string, t time.Time) (float64, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, query, t)
	}
	return 0, nil
}

func (m *MockMetricsProvider) QueryRange(ctx context.Context, query string, start, end time.Time) (float64, error) {
	if m.QueryRangeFunc != nil {
		return m.QueryRangeFunc(ctx, query, start, end)
	}
	return 0, nil
}

func TestNewTracker(t *testing.T) {
	t.Run("default tracker", func(t *testing.T) {
		tracker := NewTracker()
		assert.NotNil(t, tracker)
		assert.NotNil(t, tracker.slos)
		assert.NotNil(t, tracker.cache)
		assert.Equal(t, time.Minute, tracker.cacheTTL)
	})

	t.Run("with metrics provider", func(t *testing.T) {
		provider := &MockMetricsProvider{}
		tracker := NewTracker(WithMetricsProvider(provider))
		assert.Equal(t, provider, tracker.provider)
	})

	t.Run("with cache TTL", func(t *testing.T) {
		tracker := NewTracker(WithCacheTTL(5 * time.Minute))
		assert.Equal(t, 5*time.Minute, tracker.cacheTTL)
	})
}

func TestTracker_Register(t *testing.T) {
	tracker := NewTracker()

	t.Run("register valid SLO", func(t *testing.T) {
		slo := &SLO{
			Name:   "test-slo",
			Target: 0.99,
			Window: Duration(30 * 24 * time.Hour),
			SLI: SLISpec{
				Type:   SLITypeAvailability,
				Metric: "test_metric",
			},
		}

		err := tracker.Register(slo)
		assert.NoError(t, err)

		registered, ok := tracker.Get("test-slo")
		assert.True(t, ok)
		assert.Equal(t, slo, registered)
	})

	t.Run("register invalid SLO", func(t *testing.T) {
		slo := &SLO{
			Name:   "",
			Target: 0.99,
		}

		err := tracker.Register(slo)
		assert.Error(t, err)
	})
}

func TestTracker_RegisterAll(t *testing.T) {
	tracker := NewTracker()

	slos := []*SLO{
		{
			Name:   "slo-1",
			Target: 0.99,
			Window: Duration(30 * 24 * time.Hour),
			SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m1"},
		},
		{
			Name:   "slo-2",
			Target: 0.95,
			Window: Duration(7 * 24 * time.Hour),
			SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m2"},
		},
	}

	err := tracker.RegisterAll(slos)
	assert.NoError(t, err)

	list := tracker.List()
	assert.Len(t, list, 2)
}

func TestTracker_Unregister(t *testing.T) {
	tracker := NewTracker()

	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
		SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m"},
	}

	_ = tracker.Register(slo)
	tracker.Unregister("test-slo")

	_, ok := tracker.Get("test-slo")
	assert.False(t, ok)
}

func TestTracker_List(t *testing.T) {
	tracker := NewTracker()

	slos := []*SLO{
		{Name: "slo-1", Target: 0.99, Window: Duration(time.Hour), SLI: SLISpec{Type: SLITypeAvailability, Metric: "m1"}},
		{Name: "slo-2", Target: 0.95, Window: Duration(time.Hour), SLI: SLISpec{Type: SLITypeAvailability, Metric: "m2"}},
		{Name: "slo-3", Target: 0.999, Window: Duration(time.Hour), SLI: SLISpec{Type: SLITypeAvailability, Metric: "m3"}},
	}

	_ = tracker.RegisterAll(slos)
	list := tracker.List()
	assert.Len(t, list, 3)
}

func TestTracker_Status_NoProvider(t *testing.T) {
	tracker := NewTracker()

	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
		SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m"},
	}

	_ = tracker.Register(slo)

	ctx := context.Background()
	status, err := tracker.Status(ctx, "test-slo")

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, slo, status.SLO)
	assert.Contains(t, status.StatusMessage, "NO DATA")
	assert.True(t, status.IsHealthy)
}

func TestTracker_Status_NotFound(t *testing.T) {
	tracker := NewTracker()

	ctx := context.Background()
	_, err := tracker.Status(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTracker_Status_WithProvider(t *testing.T) {
	provider := &MockMetricsProvider{
		QueryFunc: func(ctx context.Context, query string, t time.Time) (float64, error) {
			// Return good events and total events
			if query == `sum(increase(test_metric{success="true"}[30d]))` {
				return 99, nil
			}
			if query == `sum(increase(test_metric[30d]))` {
				return 100, nil
			}
			return 0, nil
		},
	}

	tracker := NewTracker(WithMetricsProvider(provider))

	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
		SLI: SLISpec{
			Type:   SLITypeAvailability,
			Metric: "test_metric",
		},
	}

	_ = tracker.Register(slo)

	ctx := context.Background()
	status, err := tracker.Status(ctx, "test-slo")

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, float64(99), status.Current.GoodEvents)
	assert.Equal(t, float64(100), status.Current.TotalEvents)
}

func TestTracker_Status_Cached(t *testing.T) {
	callCount := 0
	provider := &MockMetricsProvider{
		QueryFunc: func(ctx context.Context, query string, t time.Time) (float64, error) {
			callCount++
			return 100, nil
		},
	}

	tracker := NewTracker(
		WithMetricsProvider(provider),
		WithCacheTTL(1*time.Hour),
	)

	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
		SLI: SLISpec{
			Type:   SLITypeAvailability,
			Metric: "test_metric",
		},
	}

	_ = tracker.Register(slo)

	ctx := context.Background()

	// First call - should fetch
	_, _ = tracker.Status(ctx, "test-slo")
	firstCallCount := callCount

	// Second call - should use cache
	_, _ = tracker.Status(ctx, "test-slo")
	assert.Equal(t, firstCallCount, callCount, "should not call provider again due to cache")
}

func TestTracker_StatusAll(t *testing.T) {
	tracker := NewTracker()

	slos := []*SLO{
		{Name: "slo-1", Target: 0.99, Window: Duration(time.Hour), SLI: SLISpec{Type: SLITypeAvailability, Metric: "m1"}},
		{Name: "slo-2", Target: 0.95, Window: Duration(time.Hour), SLI: SLISpec{Type: SLITypeAvailability, Metric: "m2"}},
	}

	_ = tracker.RegisterAll(slos)

	ctx := context.Background()
	statuses, err := tracker.StatusAll(ctx)

	require.NoError(t, err)
	assert.Len(t, statuses, 2)
}

func TestTracker_GetSummary(t *testing.T) {
	tracker := NewTracker()

	slos := []*SLO{
		{Name: "slo-1", Target: 0.99, Window: Duration(time.Hour), SLI: SLISpec{Type: SLITypeAvailability, Metric: "m1"}},
		{Name: "slo-2", Target: 0.95, Window: Duration(time.Hour), SLI: SLISpec{Type: SLITypeAvailability, Metric: "m2"}},
		{Name: "slo-3", Target: 0.999, Window: Duration(time.Hour), SLI: SLISpec{Type: SLITypeAvailability, Metric: "m3"}},
	}

	_ = tracker.RegisterAll(slos)

	ctx := context.Background()
	summary, err := tracker.GetSummary(ctx)

	require.NoError(t, err)
	assert.Equal(t, 3, summary.TotalSLOs)
	assert.Equal(t, 3, summary.HealthySLOs) // All healthy since no provider
	assert.Equal(t, OverallHealthHealthy, summary.OverallHealth)
}

func TestTracker_ClearCache(t *testing.T) {
	tracker := NewTracker()

	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(time.Hour),
		SLI:    SLISpec{Type: SLITypeAvailability, Metric: "m"},
	}

	_ = tracker.Register(slo)

	ctx := context.Background()
	_, _ = tracker.Status(ctx, "test-slo")

	// Verify cache has entry
	tracker.mu.RLock()
	assert.Len(t, tracker.cache, 1)
	tracker.mu.RUnlock()

	// Clear cache
	tracker.ClearCache()

	// Verify cache is empty
	tracker.mu.RLock()
	assert.Len(t, tracker.cache, 0)
	tracker.mu.RUnlock()
}

func TestTracker_FetchSLI_Latency(t *testing.T) {
	provider := &MockMetricsProvider{
		QueryFunc: func(ctx context.Context, query string, t time.Time) (float64, error) {
			if query == `sum(increase(latency_metric_bucket{le="30"}[30d]))` {
				return 95, nil
			}
			if query == `sum(increase(latency_metric_count[30d]))` {
				return 100, nil
			}
			return 0, nil
		},
	}

	tracker := NewTracker(WithMetricsProvider(provider))

	slo := &SLO{
		Name:   "latency-slo",
		Target: 0.95,
		Window: Duration(30 * 24 * time.Hour),
		SLI: SLISpec{
			Type:      SLITypeLatency,
			Metric:    "latency_metric",
			Threshold: 30 * time.Second,
		},
	}

	_ = tracker.Register(slo)

	ctx := context.Background()
	status, err := tracker.Status(ctx, "latency-slo")

	require.NoError(t, err)
	assert.InDelta(t, 0.95, status.Current.Value, 0.001)
}

func TestTracker_FetchSLI_Custom(t *testing.T) {
	provider := &MockMetricsProvider{
		QueryRangeFunc: func(ctx context.Context, query string, start, end time.Time) (float64, error) {
			if query == "custom_good_query" {
				return 98, nil
			}
			if query == "custom_total_query" {
				return 100, nil
			}
			return 0, nil
		},
	}

	tracker := NewTracker(WithMetricsProvider(provider))

	slo := &SLO{
		Name:   "custom-slo",
		Target: 0.95,
		Window: Duration(30 * 24 * time.Hour),
		SLI: SLISpec{
			Type:       SLITypeCustom,
			GoodQuery:  "custom_good_query",
			TotalQuery: "custom_total_query",
		},
	}

	_ = tracker.Register(slo)

	ctx := context.Background()
	status, err := tracker.Status(ctx, "custom-slo")

	require.NoError(t, err)
	assert.InDelta(t, 0.98, status.Current.Value, 0.001)
}

func TestTracker_FetchSLI_Error(t *testing.T) {
	provider := &MockMetricsProvider{
		QueryFunc: func(ctx context.Context, query string, t time.Time) (float64, error) {
			return 0, errors.New("metrics provider error")
		},
	}

	tracker := NewTracker(WithMetricsProvider(provider))

	slo := &SLO{
		Name:   "test-slo",
		Target: 0.99,
		Window: Duration(30 * 24 * time.Hour),
		SLI: SLISpec{
			Type:   SLITypeAvailability,
			Metric: "test_metric",
		},
	}

	_ = tracker.Register(slo)

	ctx := context.Background()
	_, err := tracker.Status(ctx, "test-slo")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch SLI")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * 24 * time.Hour, "30d"},
		{7 * 24 * time.Hour, "7d"},
		{24 * time.Hour, "1d"},
		{2 * time.Hour, "2h"},
		{30 * time.Minute, "30m"},
		{45 * time.Second, "45s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOverallHealth(t *testing.T) {
	assert.Equal(t, OverallHealth("healthy"), OverallHealthHealthy)
	assert.Equal(t, OverallHealth("degraded"), OverallHealthDegraded)
	assert.Equal(t, OverallHealth("critical"), OverallHealthCritical)
}
