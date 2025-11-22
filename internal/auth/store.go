package auth

import (
	"context"
	"sync"
	"time"
)

// SessionStore defines the interface for session persistence.
//
// Implementations must be thread-safe and handle concurrent access.
// Sessions should be indexed by user ID for efficient lookup.
type SessionStore interface {
	// Store saves a session.
	// If a session with the same user ID exists, it is replaced.
	Store(ctx context.Context, userID string, session *Session) error

	// Get retrieves a session by user ID.
	// Returns ErrSessionNotFound if the session doesn't exist.
	Get(ctx context.Context, userID string) (*Session, error)

	// Delete removes a session by user ID.
	// Returns nil if the session doesn't exist.
	Delete(ctx context.Context, userID string) error

	// Cleanup removes expired sessions.
	// Should be called periodically to prevent memory leaks.
	// Returns the number of sessions removed.
	Cleanup(ctx context.Context) (int, error)
}

// MemoryStore implements in-memory session storage.
//
// This is suitable for single-instance deployments and development.
// For distributed deployments, use RedisStore (future: M10.2).
//
// Sessions are stored in a concurrent map with automatic cleanup.
type MemoryStore struct {
	sessions sync.Map
}

// NewMemoryStore creates a new in-memory session store.
//
// Starts a background goroutine that periodically cleans up expired sessions.
// The cleanup interval is 5 minutes by default.
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{}

	// Start background cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			_, _ = store.Cleanup(context.Background()) //nolint:errcheck // Background cleanup, errors logged
		}
	}()

	return store
}

// Store saves a session.
func (m *MemoryStore) Store(ctx context.Context, userID string, session *Session) error {
	if userID == "" {
		return NewError(ErrSessionInvalid, "user ID cannot be empty", nil)
	}
	if session == nil {
		return NewError(ErrSessionInvalid, "session cannot be nil", nil)
	}

	m.sessions.Store(userID, session)
	return nil
}

// Get retrieves a session by user ID.
func (m *MemoryStore) Get(ctx context.Context, userID string) (*Session, error) {
	if userID == "" {
		return nil, NewError(ErrSessionInvalid, "user ID cannot be empty", nil)
	}

	value, ok := m.sessions.Load(userID)
	if !ok {
		return nil, NewError(ErrSessionNotFound, "session not found", map[string]interface{}{
			"user_id": userID,
		})
	}

	session, ok := value.(*Session)
	if !ok {
		return nil, NewError(ErrSessionInvalid, "invalid session data", map[string]interface{}{
			"user_id": userID,
		})
	}

	// Check if session is expired
	if session.IsExpired() {
		// Remove expired session
		m.sessions.Delete(userID)
		return nil, NewError(ErrSessionExpired, "session has expired", map[string]interface{}{
			"user_id":    userID,
			"expires_at": session.ExpiresAt,
		})
	}

	return session, nil
}

// Delete removes a session by user ID.
func (m *MemoryStore) Delete(ctx context.Context, userID string) error {
	if userID == "" {
		return NewError(ErrSessionInvalid, "user ID cannot be empty", nil)
	}

	m.sessions.Delete(userID)
	return nil
}

// Cleanup removes expired sessions.
func (m *MemoryStore) Cleanup(ctx context.Context) (int, error) {
	count := 0
	now := time.Now()

	m.sessions.Range(func(key, value interface{}) bool {
		session, ok := value.(*Session)
		if !ok {
			// Invalid session data, remove it
			m.sessions.Delete(key)
			count++
			return true
		}

		if now.After(session.ExpiresAt) {
			m.sessions.Delete(key)
			count++
		}

		return true
	})

	return count, nil
}

// Count returns the number of sessions in the store.
// This is useful for monitoring and testing.
func (m *MemoryStore) Count() int {
	count := 0
	m.sessions.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
