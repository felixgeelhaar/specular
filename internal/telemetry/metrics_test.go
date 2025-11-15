package telemetry

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// setupTestMetrics initializes metrics with in-memory reader for testing
func setupTestMetrics(t *testing.T) (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	t.Helper()

	// Create manual reader for testing
	reader := sdkmetric.NewManualReader()

	cfg := DefaultConfig()
	res, err := createResource(cfg)
	if err != nil {
		t.Fatalf("createResource failed: %v", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	// Set global provider and reset metricsOnce to allow re-initialization
	meterMu.Lock()
	globalMeterProvider = mp
	metricsOnce = sync.Once{} // Reset to allow initMetrics to run again
	meterMu.Unlock()

	if err := initMetrics(); err != nil {
		t.Fatalf("initMetrics failed: %v", err)
	}

	return mp, reader
}

// collectMetrics collects and returns metrics data
func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) *metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	return &rm
}

func TestRecordCommandInvocation(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() {
		_ = mp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	commandName := "test-command"
	status := "started"

	// Record command invocation
	RecordCommandInvocation(ctx, commandName, status,
		attribute.String("profile", "dev"),
	)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify metric was recorded
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "specular.command.invocations" {
				found = true

				// Verify it's a counter
				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64], got %T", m.Data)
					continue
				}

				// Verify data points
				if len(data.DataPoints) == 0 {
					t.Error("expected data points, got none")
					continue
				}

				// Verify attributes
				dp := data.DataPoints[0]
				hasCommand := false
				hasStatus := false
				hasProfile := false

				for _, attr := range dp.Attributes.ToSlice() {
					if string(attr.Key) == "command" && attr.Value.AsString() == commandName {
						hasCommand = true
					}
					if string(attr.Key) == "status" && attr.Value.AsString() == status {
						hasStatus = true
					}
					if string(attr.Key) == "profile" && attr.Value.AsString() == "dev" {
						hasProfile = true
					}
				}

				if !hasCommand {
					t.Error("missing 'command' attribute")
				}
				if !hasStatus {
					t.Error("missing 'status' attribute")
				}
				if !hasProfile {
					t.Error("missing 'profile' attribute")
				}

				// Verify value
				if dp.Value != 1 {
					t.Errorf("counter value = %d, want 1", dp.Value)
				}
			}
		}
	}

	if !found {
		t.Error("metric 'specular.command.invocations' not found")
	}
}

func TestRecordCommandDuration(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() {
		_ = mp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	commandName := "test-command"
	duration := 2500 * time.Millisecond

	// Record command duration
	RecordCommandDuration(ctx, commandName, duration)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify metric was recorded
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "specular.command.duration" {
				found = true

				// Verify it's a histogram
				data, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Errorf("expected Histogram[float64], got %T", m.Data)
					continue
				}

				// Verify data points
				if len(data.DataPoints) == 0 {
					t.Error("expected data points, got none")
					continue
				}

				dp := data.DataPoints[0]

				// Verify attributes
				hasCommand := false
				for _, attr := range dp.Attributes.ToSlice() {
					if string(attr.Key) == "command" && attr.Value.AsString() == commandName {
						hasCommand = true
					}
				}

				if !hasCommand {
					t.Error("missing 'command' attribute")
				}

				// Verify count
				if dp.Count != 1 {
					t.Errorf("histogram count = %d, want 1", dp.Count)
				}

				// Verify sum (duration in seconds)
				expectedSum := duration.Seconds()
				if dp.Sum != expectedSum {
					t.Errorf("histogram sum = %f, want %f", dp.Sum, expectedSum)
				}
			}
		}
	}

	if !found {
		t.Error("metric 'specular.command.duration' not found")
	}
}

func TestRecordCommandError(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() {
		_ = mp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	commandName := "test-command"
	errorType := "execution_error"

	// Record command error
	RecordCommandError(ctx, commandName, errorType)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify metric was recorded
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "specular.command.errors" {
				found = true

				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64], got %T", m.Data)
					continue
				}

				if len(data.DataPoints) == 0 {
					t.Error("expected data points, got none")
					continue
				}

				dp := data.DataPoints[0]

				// Verify attributes
				hasCommand := false
				hasErrorType := false

				for _, attr := range dp.Attributes.ToSlice() {
					if string(attr.Key) == "command" && attr.Value.AsString() == commandName {
						hasCommand = true
					}
					if string(attr.Key) == "error_type" && attr.Value.AsString() == errorType {
						hasErrorType = true
					}
				}

				if !hasCommand {
					t.Error("missing 'command' attribute")
				}
				if !hasErrorType {
					t.Error("missing 'error_type' attribute")
				}

				if dp.Value != 1 {
					t.Errorf("counter value = %d, want 1", dp.Value)
				}
			}
		}
	}

	if !found {
		t.Error("metric 'specular.command.errors' not found")
	}
}

func TestRecordProviderCall(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() {
		_ = mp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	provider := "test-provider"
	operation := "generate"
	status := "success"

	// Record provider call
	RecordProviderCall(ctx, provider, operation, status,
		attribute.String("model", "test-model"),
	)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify metric was recorded
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "specular.provider.calls" {
				found = true

				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64], got %T", m.Data)
					continue
				}

				if len(data.DataPoints) == 0 {
					t.Error("expected data points, got none")
					continue
				}

				dp := data.DataPoints[0]

				// Verify attributes
				hasProvider := false
				hasOperation := false
				hasStatus := false
				hasModel := false

				for _, attr := range dp.Attributes.ToSlice() {
					key := string(attr.Key)
					if key == "provider" && attr.Value.AsString() == provider {
						hasProvider = true
					}
					if key == "operation" && attr.Value.AsString() == operation {
						hasOperation = true
					}
					if key == "status" && attr.Value.AsString() == status {
						hasStatus = true
					}
					if key == "model" && attr.Value.AsString() == "test-model" {
						hasModel = true
					}
				}

				if !hasProvider {
					t.Error("missing 'provider' attribute")
				}
				if !hasOperation {
					t.Error("missing 'operation' attribute")
				}
				if !hasStatus {
					t.Error("missing 'status' attribute")
				}
				if !hasModel {
					t.Error("missing 'model' attribute")
				}

				if dp.Value != 1 {
					t.Errorf("counter value = %d, want 1", dp.Value)
				}
			}
		}
	}

	if !found {
		t.Error("metric 'specular.provider.calls' not found")
	}
}

func TestRecordProviderLatency(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() {
		_ = mp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	provider := "test-provider"
	operation := "generate"
	duration := 1200 * time.Millisecond

	// Record provider latency
	RecordProviderLatency(ctx, provider, operation, duration)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify metric was recorded
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "specular.provider.latency" {
				found = true

				data, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Errorf("expected Histogram[float64], got %T", m.Data)
					continue
				}

				if len(data.DataPoints) == 0 {
					t.Error("expected data points, got none")
					continue
				}

				dp := data.DataPoints[0]

				// Verify count and sum
				if dp.Count != 1 {
					t.Errorf("histogram count = %d, want 1", dp.Count)
				}

				expectedSum := duration.Seconds()
				if dp.Sum != expectedSum {
					t.Errorf("histogram sum = %f, want %f", dp.Sum, expectedSum)
				}
			}
		}
	}

	if !found {
		t.Error("metric 'specular.provider.latency' not found")
	}
}

func TestRecordProviderError(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() {
		_ = mp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	provider := "test-provider"
	operation := "generate"
	errorType := "api_error"

	// Record provider error
	RecordProviderError(ctx, provider, operation, errorType)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify metric was recorded
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "specular.provider.errors" {
				found = true

				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64], got %T", m.Data)
					continue
				}

				if len(data.DataPoints) == 0 {
					t.Error("expected data points, got none")
					continue
				}

				dp := data.DataPoints[0]

				// Verify attributes
				hasProvider := false
				hasOperation := false
				hasErrorType := false

				for _, attr := range dp.Attributes.ToSlice() {
					key := string(attr.Key)
					if key == "provider" && attr.Value.AsString() == provider {
						hasProvider = true
					}
					if key == "operation" && attr.Value.AsString() == operation {
						hasOperation = true
					}
					if key == "error_type" && attr.Value.AsString() == errorType {
						hasErrorType = true
					}
				}

				if !hasProvider {
					t.Error("missing 'provider' attribute")
				}
				if !hasOperation {
					t.Error("missing 'operation' attribute")
				}
				if !hasErrorType {
					t.Error("missing 'error_type' attribute")
				}

				if dp.Value != 1 {
					t.Errorf("counter value = %d, want 1", dp.Value)
				}
			}
		}
	}

	if !found {
		t.Error("metric 'specular.provider.errors' not found")
	}
}

func TestRecordProviderTokens(t *testing.T) {
	mp, reader := setupTestMetrics(t)
	defer func() {
		_ = mp.Shutdown(context.Background())
	}()

	ctx := context.Background()
	provider := "test-provider"
	model := "test-model"
	tokenType := "input"
	count := 1234

	// Record provider tokens
	RecordProviderTokens(ctx, provider, model, tokenType, count)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify metric was recorded
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "specular.provider.tokens" {
				found = true

				data, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("expected Sum[int64], got %T", m.Data)
					continue
				}

				if len(data.DataPoints) == 0 {
					t.Error("expected data points, got none")
					continue
				}

				dp := data.DataPoints[0]

				// Verify attributes
				hasProvider := false
				hasModel := false
				hasTokenType := false

				for _, attr := range dp.Attributes.ToSlice() {
					key := string(attr.Key)
					if key == "provider" && attr.Value.AsString() == provider {
						hasProvider = true
					}
					if key == "model" && attr.Value.AsString() == model {
						hasModel = true
					}
					if key == "token_type" && attr.Value.AsString() == tokenType {
						hasTokenType = true
					}
				}

				if !hasProvider {
					t.Error("missing 'provider' attribute")
				}
				if !hasModel {
					t.Error("missing 'model' attribute")
				}
				if !hasTokenType {
					t.Error("missing 'token_type' attribute")
				}

				if dp.Value != int64(count) {
					t.Errorf("counter value = %d, want %d", dp.Value, count)
				}
			}
		}
	}

	if !found {
		t.Error("metric 'specular.provider.tokens' not found")
	}
}

func TestGetMetricsBeforeInit(t *testing.T) {
	// Reset global metrics
	meterMu.Lock()
	metrics = nil
	meterMu.Unlock()

	// GetMetrics should return empty metrics, not panic
	m := GetMetrics()
	if m == nil {
		t.Error("GetMetrics returned nil, expected empty metrics")
	}

	// Should be safe to call with nil counters (no-op)
	ctx := context.Background()
	RecordCommandInvocation(ctx, "test", "started") // Should not panic
}
