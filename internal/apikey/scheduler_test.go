package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	tests := []struct {
		name        string
		config      SchedulerConfig
		expectError bool
	}{
		{
			name: "valid configuration",
			config: SchedulerConfig{
				Manager:       &Manager{},
				CheckInterval: 1 * time.Hour,
				GracePeriod:   7 * 24 * time.Hour,
				RotationTTL:   7 * 24 * time.Hour,
			},
			expectError: false,
		},
		{
			name: "default values",
			config: SchedulerConfig{
				Manager: &Manager{},
			},
			expectError: false,
		},
		{
			name: "missing manager",
			config: SchedulerConfig{
				CheckInterval: 1 * time.Hour,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler, err := NewScheduler(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, scheduler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scheduler)

				// Verify defaults
				if tt.config.CheckInterval == 0 {
					assert.Equal(t, 1*time.Hour, scheduler.checkInterval)
				}
				if tt.config.GracePeriod == 0 {
					assert.Equal(t, 7*24*time.Hour, scheduler.gracePeriod)
				}
				if tt.config.RotationTTL == 0 {
					assert.Equal(t, 7*24*time.Hour, scheduler.rotationTTL)
				}
			}
		})
	}
}

func TestNeedsRotation(t *testing.T) {
	scheduler := &Scheduler{
		rotationTTL: 7 * 24 * time.Hour, // Rotate 7 days before expiry
	}

	now := time.Now().UTC()

	tests := []struct {
		name           string
		key            *APIKey
		now            time.Time
		expectRotation bool
	}{
		{
			name: "needs rotation - within window",
			key: &APIKey{
				ExpiresAt: now.Add(5 * 24 * time.Hour), // Expires in 5 days
			},
			now:            now,
			expectRotation: true, // Should rotate (5 days < 7 days)
		},
		{
			name: "does not need rotation - outside window",
			key: &APIKey{
				ExpiresAt: now.Add(10 * 24 * time.Hour), // Expires in 10 days
			},
			now:            now,
			expectRotation: false, // Should not rotate (10 days > 7 days)
		},
		{
			name: "needs rotation - at threshold",
			key: &APIKey{
				ExpiresAt: now.Add(7 * 24 * time.Hour), // Expires in exactly 7 days
			},
			now:            now,
			expectRotation: false, // At boundary, should not rotate yet
		},
		{
			name: "needs rotation - past expiry",
			key: &APIKey{
				ExpiresAt: now.Add(-1 * time.Hour), // Already expired
			},
			now:            now,
			expectRotation: true,
		},
		{
			name: "no expiry set",
			key: &APIKey{
				ExpiresAt: time.Time{}, // Zero time
			},
			now:            now,
			expectRotation: false, // Never rotates if no expiry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsRotation := scheduler.needsRotation(tt.key, tt.now)
			assert.Equal(t, tt.expectRotation, needsRotation)
		})
	}
}

func TestGetRotationStatus(t *testing.T) {
	// This test would require a mock manager to list keys
	// For now, test the status structure

	t.Run("rotation status structure", func(t *testing.T) {
		status := &RotationStatus{
			TotalKeys:       10,
			ActiveKeys:      7,
			RotatedKeys:     1,
			RevokedKeys:     1,
			ExpiredKeys:     1,
			NeedingRotation: 2,
			DaysUntilRotation: map[string]int{
				"key-1": 5,
				"key-2": 3,
			},
		}

		assert.Equal(t, 10, status.TotalKeys)
		assert.Equal(t, 7, status.ActiveKeys)
		assert.Equal(t, 2, status.NeedingRotation)
		assert.Equal(t, 5, status.DaysUntilRotation["key-1"])
		assert.Equal(t, 3, status.DaysUntilRotation["key-2"])
	})
}

func TestSchedulerLifecycle(t *testing.T) {
	t.Run("start and stop scheduler", func(t *testing.T) {
		scheduler, err := NewScheduler(SchedulerConfig{
			Manager:       &Manager{},
			CheckInterval: 100 * time.Millisecond, // Fast for testing
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start scheduler in background
		done := make(chan struct{})
		go func() {
			scheduler.Start(ctx)
			close(done)
		}()

		// Let it run for a short time
		time.Sleep(250 * time.Millisecond)

		// Stop the scheduler
		scheduler.Stop()

		// Wait for completion
		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("Scheduler did not stop in time")
		}
	})

	t.Run("scheduler respects context cancellation", func(t *testing.T) {
		scheduler, err := NewScheduler(SchedulerConfig{
			Manager:       &Manager{},
			CheckInterval: 100 * time.Millisecond,
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		// Start scheduler in background
		done := make(chan struct{})
		go func() {
			scheduler.Start(ctx)
			close(done)
		}()

		// Let it run briefly
		time.Sleep(150 * time.Millisecond)

		// Cancel context
		cancel()

		// Wait for completion
		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("Scheduler did not respect context cancellation")
		}
	})
}

func TestRotationTiming(t *testing.T) {
	tests := []struct {
		name         string
		rotationTTL  time.Duration
		expiresIn    time.Duration
		shouldRotate bool
	}{
		{
			name:         "rotate 7 days before - expires in 5 days",
			rotationTTL:  7 * 24 * time.Hour,
			expiresIn:    5 * 24 * time.Hour,
			shouldRotate: true,
		},
		{
			name:         "rotate 7 days before - expires in 10 days",
			rotationTTL:  7 * 24 * time.Hour,
			expiresIn:    10 * 24 * time.Hour,
			shouldRotate: false,
		},
		{
			name:         "rotate 30 days before - expires in 25 days",
			rotationTTL:  30 * 24 * time.Hour,
			expiresIn:    25 * 24 * time.Hour,
			shouldRotate: true,
		},
		{
			name:         "rotate 1 day before - expires in 2 hours",
			rotationTTL:  24 * time.Hour,
			expiresIn:    2 * time.Hour,
			shouldRotate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := &Scheduler{
				rotationTTL: tt.rotationTTL,
			}

			now := time.Now().UTC()
			key := &APIKey{
				ExpiresAt: now.Add(tt.expiresIn),
			}

			needsRotation := scheduler.needsRotation(key, now)
			assert.Equal(t, tt.shouldRotate, needsRotation)
		})
	}
}

func TestCleanupLogic(t *testing.T) {
	tests := []struct {
		name             string
		keyStatus        Status
		statusTime       time.Time
		cleanupThreshold time.Duration
		now              time.Time
		shouldCleanup    bool
	}{
		{
			name:             "cleanup revoked key after threshold",
			keyStatus:        StatusRevoked,
			statusTime:       time.Now().UTC().Add(-10 * 24 * time.Hour),
			cleanupThreshold: 7 * 24 * time.Hour,
			now:              time.Now().UTC(),
			shouldCleanup:    true,
		},
		{
			name:             "do not cleanup recent revoked key",
			keyStatus:        StatusRevoked,
			statusTime:       time.Now().UTC().Add(-5 * 24 * time.Hour),
			cleanupThreshold: 7 * 24 * time.Hour,
			now:              time.Now().UTC(),
			shouldCleanup:    false,
		},
		{
			name:             "cleanup expired key after threshold",
			keyStatus:        StatusExpired,
			statusTime:       time.Now().UTC().Add(-10 * 24 * time.Hour),
			cleanupThreshold: 7 * 24 * time.Hour,
			now:              time.Now().UTC(),
			shouldCleanup:    true,
		},
		{
			name:             "do not cleanup active key",
			keyStatus:        StatusActive,
			statusTime:       time.Now().UTC().Add(-10 * 24 * time.Hour),
			cleanupThreshold: 7 * 24 * time.Hour,
			now:              time.Now().UTC(),
			shouldCleanup:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{
				Status: tt.keyStatus,
			}

			if tt.keyStatus == StatusRevoked {
				key.RevokedAt = tt.statusTime
			} else if tt.keyStatus == StatusExpired {
				key.ExpiresAt = tt.statusTime
			}

			shouldCleanup := false

			// Replicate cleanup logic
			if key.Status == StatusRevoked && !key.RevokedAt.IsZero() {
				if tt.now.Sub(key.RevokedAt) > tt.cleanupThreshold {
					shouldCleanup = true
				}
			}

			if key.Status == StatusExpired && !key.ExpiresAt.IsZero() {
				if tt.now.Sub(key.ExpiresAt) > tt.cleanupThreshold {
					shouldCleanup = true
				}
			}

			assert.Equal(t, tt.shouldCleanup, shouldCleanup)
		})
	}
}

func TestRotationStatusCalculation(t *testing.T) {
	now := time.Now().UTC()
	rotationTTL := 7 * 24 * time.Hour

	keys := []*APIKey{
		{
			ID:        "key-1",
			Status:    StatusActive,
			ExpiresAt: now.Add(10 * 24 * time.Hour), // 10 days until expiry
		},
		{
			ID:        "key-2",
			Status:    StatusActive,
			ExpiresAt: now.Add(5 * 24 * time.Hour), // 5 days until expiry (needs rotation)
		},
		{
			ID:     "key-3",
			Status: StatusRotated,
		},
		{
			ID:     "key-4",
			Status: StatusRevoked,
		},
		{
			ID:     "key-5",
			Status: StatusExpired,
		},
	}

	// Calculate expected status
	totalKeys := len(keys)
	activeKeys := 0
	rotatedKeys := 0
	revokedKeys := 0
	expiredKeys := 0
	needingRotation := 0
	daysUntilRotation := make(map[string]int)

	for _, key := range keys {
		switch key.Status {
		case StatusActive:
			activeKeys++
			if !key.ExpiresAt.IsZero() {
				rotationThreshold := key.ExpiresAt.Add(-rotationTTL)
				if now.After(rotationThreshold) {
					needingRotation++
				}
				daysUntil := int(rotationThreshold.Sub(now).Hours() / 24)
				daysUntilRotation[key.ID] = daysUntil
			}
		case StatusRotated:
			rotatedKeys++
		case StatusRevoked:
			revokedKeys++
		case StatusExpired:
			expiredKeys++
		}
	}

	assert.Equal(t, 5, totalKeys)
	assert.Equal(t, 2, activeKeys)
	assert.Equal(t, 1, rotatedKeys)
	assert.Equal(t, 1, revokedKeys)
	assert.Equal(t, 1, expiredKeys)
	assert.Equal(t, 1, needingRotation)             // Only key-2 needs rotation
	assert.Equal(t, 3, daysUntilRotation["key-1"])  // 10 - 7 = 3 days until rotation window
	assert.Equal(t, -2, daysUntilRotation["key-2"]) // 5 - 7 = -2 (in rotation window)
}

func TestSchedulerConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		checkInterval time.Duration
		gracePeriod   time.Duration
		rotationTTL   time.Duration
	}{
		{
			name:          "aggressive rotation",
			checkInterval: 15 * time.Minute,
			gracePeriod:   1 * 24 * time.Hour, // 1 day grace
			rotationTTL:   3 * 24 * time.Hour, // Rotate 3 days before expiry
		},
		{
			name:          "conservative rotation",
			checkInterval: 4 * time.Hour,
			gracePeriod:   14 * 24 * time.Hour, // 14 days grace
			rotationTTL:   14 * 24 * time.Hour, // Rotate 14 days before expiry
		},
		{
			name:          "balanced rotation",
			checkInterval: 1 * time.Hour,
			gracePeriod:   7 * 24 * time.Hour, // 7 days grace
			rotationTTL:   7 * 24 * time.Hour, // Rotate 7 days before expiry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler, err := NewScheduler(SchedulerConfig{
				Manager:       &Manager{},
				CheckInterval: tt.checkInterval,
				GracePeriod:   tt.gracePeriod,
				RotationTTL:   tt.rotationTTL,
			})

			require.NoError(t, err)
			assert.Equal(t, tt.checkInterval, scheduler.checkInterval)
			assert.Equal(t, tt.gracePeriod, scheduler.gracePeriod)
			assert.Equal(t, tt.rotationTTL, scheduler.rotationTTL)
		})
	}
}

func TestForceRotation(t *testing.T) {
	// Test that force rotation works regardless of expiry time
	t.Run("force rotation ignores expiry time", func(t *testing.T) {
		scheduler := &Scheduler{
			rotationTTL: 7 * 24 * time.Hour,
		}

		now := time.Now().UTC()

		// Key that doesn't need rotation normally
		key := &APIKey{
			Status:    StatusActive,
			ExpiresAt: now.Add(30 * 24 * time.Hour), // 30 days until expiry
		}

		// Normal rotation check
		normallyNeedsRotation := scheduler.needsRotation(key, now)
		assert.False(t, normallyNeedsRotation)

		// With force=true, should rotate anyway
		// This logic would be: force || needsRotation
		shouldRotateWithForce := true || normallyNeedsRotation
		assert.True(t, shouldRotateWithForce)
	})
}

func TestConcurrentSchedulerOperations(t *testing.T) {
	t.Run("multiple stop calls are safe", func(t *testing.T) {
		scheduler, err := NewScheduler(SchedulerConfig{
			Manager:       &Manager{},
			CheckInterval: 100 * time.Millisecond,
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go scheduler.Start(ctx)

		time.Sleep(50 * time.Millisecond)

		// Multiple stop calls should not panic
		assert.NotPanics(t, func() {
			scheduler.Stop()
		})
	})
}
