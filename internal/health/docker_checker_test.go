package health

import (
	"context"
	"testing"
)

func TestNewDockerChecker(t *testing.T) {
	checker := NewDockerChecker()

	if checker == nil {
		t.Fatal("NewDockerChecker returned nil")
	}
}

func TestDockerCheckerName(t *testing.T) {
	checker := NewDockerChecker()

	name := checker.Name()
	if name != "docker-daemon" {
		t.Errorf("Name() = %q, want %q", name, "docker-daemon")
	}
}

func TestDockerCheckerCheck(t *testing.T) {
	checker := NewDockerChecker()
	ctx := context.Background()

	result := checker.Check(ctx)

	// We can't predict if Docker is running on the test machine,
	// but we can verify the result structure is valid
	if result == nil {
		t.Fatal("Check() returned nil")
	}

	if result.Status == "" {
		t.Error("Status should not be empty")
	}

	if result.Message == "" {
		t.Error("Message should not be empty")
	}

	if result.Details == nil {
		t.Error("Details should be initialized")
	}

	// Check status is one of the valid values
	validStatuses := map[Status]bool{
		StatusHealthy:   true,
		StatusDegraded:  true,
		StatusUnhealthy: true,
	}

	if !validStatuses[result.Status] {
		t.Errorf("Status = %v, want one of [healthy, degraded, unhealthy]", result.Status)
	}

	// If healthy, should have docker_path and server_version
	if result.Status == StatusHealthy {
		if _, ok := result.Details["docker_path"]; !ok {
			t.Error("Healthy result should include docker_path")
		}

		if _, ok := result.Details["server_version"]; !ok {
			t.Error("Healthy result should include server_version")
		}
	}

	// If unhealthy, should have suggestion
	if result.Status == StatusUnhealthy {
		if _, ok := result.Details["suggestion"]; !ok {
			t.Error("Unhealthy result should include suggestion")
		}
	}
}

func TestDockerCheckerCheckWithCancelledContext(t *testing.T) {
	checker := NewDockerChecker()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := checker.Check(ctx)

	if result == nil {
		t.Fatal("Check() returned nil")
	}

	// With a cancelled context, the check should fail quickly
	// Either unhealthy or degraded depending on timing
	if result.Status == StatusHealthy {
		t.Error("Check with cancelled context should not return healthy")
	}
}
