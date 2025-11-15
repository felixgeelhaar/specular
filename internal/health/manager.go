package health

import (
	"context"
	"sync"
	"time"
)

// Manager coordinates health checks and aggregates results.
// It runs checks in parallel with timeouts and collects all results.
type Manager struct {
	checkers []Checker
	timeout  time.Duration
	mu       sync.RWMutex
}

// NewManager creates a new health check manager with default 5-second timeout.
func NewManager() *Manager {
	return &Manager{
		checkers: make([]Checker, 0),
		timeout:  5 * time.Second,
	}
}

// WithTimeout sets a custom timeout for health checks.
func (m *Manager) WithTimeout(timeout time.Duration) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeout = timeout
	return m
}

// AddChecker registers a new health checker.
// Checkers are executed in the order they are added.
func (m *Manager) AddChecker(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, checker)
}

// RemoveChecker removes a checker by name.
// Returns true if a checker was removed, false otherwise.
func (m *Manager) RemoveChecker(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, checker := range m.checkers {
		if checker.Name() == name {
			m.checkers = append(m.checkers[:i], m.checkers[i+1:]...)
			return true
		}
	}
	return false
}

// Check runs all registered health checks in parallel and returns aggregated results.
// Each check runs with a timeout to prevent hanging.
// Returns a map of checker name to result.
func (m *Manager) Check(ctx context.Context) map[string]*Result {
	m.mu.RLock()
	checkers := make([]Checker, len(m.checkers))
	copy(checkers, m.checkers)
	timeout := m.timeout
	m.mu.RUnlock()

	results := make(map[string]*Result)
	resultsMu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, checker := range checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()

			// Create context with timeout
			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			start := time.Now()
			result := c.Check(checkCtx)
			latency := time.Since(start)

			// Set latency if not already set
			if result.Latency == 0 {
				result.Latency = latency
			}

			resultsMu.Lock()
			results[c.Name()] = result
			resultsMu.Unlock()
		}(checker)
	}

	wg.Wait()
	return results
}

// OverallStatus determines the overall system health based on all check results.
// Returns:
//   - StatusHealthy if all checks are healthy
//   - StatusDegraded if any check is degraded
//   - StatusUnhealthy if any check is unhealthy
func (m *Manager) OverallStatus(results map[string]*Result) Status {
	if len(results) == 0 {
		return StatusHealthy
	}

	hasDegraded := false
	for _, result := range results {
		if result.Status == StatusUnhealthy {
			return StatusUnhealthy
		}
		if result.Status == StatusDegraded {
			hasDegraded = true
		}
	}

	if hasDegraded {
		return StatusDegraded
	}

	return StatusHealthy
}

// CheckNames returns the names of all registered checkers.
func (m *Manager) CheckNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, len(m.checkers))
	for i, checker := range m.checkers {
		names[i] = checker.Name()
	}
	return names
}

// Count returns the number of registered checkers.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.checkers)
}
