package apikey

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Scheduler manages automatic API key rotation.
type Scheduler struct {
	manager       *Manager
	checkInterval time.Duration
	gracePeriod   time.Duration
	rotationTTL   time.Duration // Time before expiry to trigger rotation
	stopChan      chan struct{}
	done          chan struct{}
}

// SchedulerConfig holds configuration for the rotation scheduler.
type SchedulerConfig struct {
	Manager       *Manager
	CheckInterval time.Duration // How often to check for keys needing rotation (default: 1 hour)
	GracePeriod   time.Duration // Grace period for old keys after rotation (default: 7 days)
	RotationTTL   time.Duration // Rotate keys this far before expiry (default: 7 days)
}

// NewScheduler creates a new rotation scheduler.
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error) {
	if cfg.Manager == nil {
		return nil, fmt.Errorf("manager is required")
	}

	// Set defaults
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 1 * time.Hour
	}
	if cfg.GracePeriod == 0 {
		cfg.GracePeriod = 7 * 24 * time.Hour // 7 days
	}
	if cfg.RotationTTL == 0 {
		cfg.RotationTTL = 7 * 24 * time.Hour // Rotate 7 days before expiry
	}

	return &Scheduler{
		manager:       cfg.Manager,
		checkInterval: cfg.CheckInterval,
		gracePeriod:   cfg.GracePeriod,
		rotationTTL:   cfg.RotationTTL,
		stopChan:      make(chan struct{}),
		done:          make(chan struct{}),
	}, nil
}

// Start starts the rotation scheduler.
func (s *Scheduler) Start(ctx context.Context) {
	log.Printf("Starting API key rotation scheduler (check interval: %v)", s.checkInterval)

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()
	defer close(s.done)

	// Run immediately on start
	s.checkRotations(ctx)

	for {
		select {
		case <-ticker.C:
			s.checkRotations(ctx)
		case <-s.stopChan:
			log.Println("Stopping API key rotation scheduler")
			return
		case <-ctx.Done():
			log.Println("Context cancelled, stopping API key rotation scheduler")
			return
		}
	}
}

// Stop stops the rotation scheduler.
func (s *Scheduler) Stop() {
	close(s.stopChan)
	<-s.done
}

// checkRotations checks all organizations for keys that need rotation.
func (s *Scheduler) checkRotations(ctx context.Context) {
	log.Println("Checking for API keys needing rotation...")

	// In a real implementation, you would iterate through all organizations
	// For now, this is a placeholder that would need organization listing capability

	// TODO: Implement organization iteration
	// For now, we'll just log
	log.Println("Rotation check completed (organization iteration not yet implemented)")
}

// checkOrganizationKeys checks and rotates keys for a specific organization.
func (s *Scheduler) checkOrganizationKeys(ctx context.Context, orgID string) error {
	keys, err := s.manager.ListKeys(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to list keys for org %s: %w", orgID, err)
	}

	now := time.Now().UTC()
	rotatedCount := 0
	errorCount := 0

	for _, key := range keys {
		// Skip non-active keys
		if key.Status != StatusActive {
			continue
		}

		// Check if key needs rotation
		if s.needsRotation(key, now) {
			log.Printf("Rotating API key %s for organization %s (expires: %v)",
				key.ID, key.OrganizationID, key.ExpiresAt)

			if _, err := s.manager.RotateKey(ctx, orgID, key.ID, s.gracePeriod); err != nil {
				log.Printf("ERROR: Failed to rotate key %s: %v", key.ID, err)
				errorCount++
			} else {
				rotatedCount++
			}
		}
	}

	if rotatedCount > 0 {
		log.Printf("Rotated %d API keys for organization %s", rotatedCount, orgID)
	}

	if errorCount > 0 {
		return fmt.Errorf("failed to rotate %d keys", errorCount)
	}

	return nil
}

// needsRotation determines if an API key needs rotation.
func (s *Scheduler) needsRotation(key *APIKey, now time.Time) bool {
	// Don't rotate if no expiry set
	if key.ExpiresAt.IsZero() {
		return false
	}

	// Check if we're within the rotation window (rotationTTL before expiry)
	rotationThreshold := key.ExpiresAt.Add(-s.rotationTTL)

	return now.After(rotationThreshold)
}

// RotateAllKeys manually triggers rotation for all keys in an organization
// that are within the rotation window.
func (s *Scheduler) RotateAllKeys(ctx context.Context, orgID string) (int, error) {
	return s.rotateKeys(ctx, orgID, false)
}

// ForceRotateAllKeys manually forces rotation for all active keys in an organization,
// regardless of expiry time.
func (s *Scheduler) ForceRotateAllKeys(ctx context.Context, orgID string) (int, error) {
	return s.rotateKeys(ctx, orgID, true)
}

// rotateKeys rotates keys for an organization.
func (s *Scheduler) rotateKeys(ctx context.Context, orgID string, force bool) (int, error) {
	keys, err := s.manager.ListKeys(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to list keys: %w", err)
	}

	now := time.Now().UTC()
	rotatedCount := 0

	for _, key := range keys {
		// Skip non-active keys
		if key.Status != StatusActive {
			continue
		}

		// Check if rotation is needed (or forced)
		if force || s.needsRotation(key, now) {
			if _, err := s.manager.RotateKey(ctx, orgID, key.ID, s.gracePeriod); err != nil {
				log.Printf("ERROR: Failed to rotate key %s: %v", key.ID, err)
				continue
			}
			rotatedCount++
		}
	}

	return rotatedCount, nil
}

// GetRotationStatus returns rotation status for all keys in an organization.
type RotationStatus struct {
	TotalKeys         int
	ActiveKeys        int
	RotatedKeys       int
	RevokedKeys       int
	ExpiredKeys       int
	NeedingRotation   int
	DaysUntilRotation map[string]int // keyID -> days
}

// GetRotationStatus gets the rotation status for an organization.
func (s *Scheduler) GetRotationStatus(ctx context.Context, orgID string) (*RotationStatus, error) {
	keys, err := s.manager.ListKeys(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	now := time.Now().UTC()
	status := &RotationStatus{
		TotalKeys:         len(keys),
		DaysUntilRotation: make(map[string]int),
	}

	for _, key := range keys {
		switch key.Status {
		case StatusActive:
			status.ActiveKeys++

			// Check if needs rotation
			if s.needsRotation(key, now) {
				status.NeedingRotation++
			}

			// Calculate days until rotation
			if !key.ExpiresAt.IsZero() {
				rotationTime := key.ExpiresAt.Add(-s.rotationTTL)
				daysUntil := int(rotationTime.Sub(now).Hours() / 24)
				status.DaysUntilRotation[key.ID] = daysUntil
			}

		case StatusRotated:
			status.RotatedKeys++

		case StatusRevoked:
			status.RevokedKeys++

		case StatusExpired:
			status.ExpiredKeys++
		}
	}

	return status, nil
}

// CleanupExpiredKeys removes keys that have been expired or revoked
// for longer than the cleanup threshold.
func (s *Scheduler) CleanupExpiredKeys(ctx context.Context, orgID string, cleanupThreshold time.Duration) (int, error) {
	keys, err := s.manager.ListKeys(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to list keys: %w", err)
	}

	now := time.Now().UTC()
	cleanedCount := 0

	for _, key := range keys {
		shouldCleanup := false

		// Cleanup revoked keys after threshold
		if key.Status == StatusRevoked && !key.RevokedAt.IsZero() {
			if now.Sub(key.RevokedAt) > cleanupThreshold {
				shouldCleanup = true
			}
		}

		// Cleanup expired keys after threshold
		if key.Status == StatusExpired && !key.ExpiresAt.IsZero() {
			if now.Sub(key.ExpiresAt) > cleanupThreshold {
				shouldCleanup = true
			}
		}

		if shouldCleanup {
			log.Printf("Cleaning up %s key %s (org: %s)", key.Status, key.ID, orgID)

			if err := s.manager.DeleteKey(ctx, orgID, key.ID); err != nil {
				log.Printf("ERROR: Failed to delete key %s: %v", key.ID, err)
				continue
			}
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		log.Printf("Cleaned up %d expired/revoked keys for organization %s", cleanedCount, orgID)
	}

	return cleanedCount, nil
}
