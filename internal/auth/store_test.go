package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_Store(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	session := &Session{
		UserID:    "user-123",
		Email:     "user@example.com",
		Provider:  "test_provider",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := store.Store(ctx, session.UserID, session)
	require.NoError(t, err)

	// Verify session can be retrieved
	retrieved, err := store.Get(ctx, session.UserID)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Email, retrieved.Email)
	assert.Equal(t, session.Provider, retrieved.Provider)
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	session := &Session{
		UserID:    "user-123",
		Email:     "user@example.com",
		Provider:  "test_provider",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Get non-existent session
	retrieved, err := store.Get(ctx, "nonexistent")
	require.Error(t, err)
	assert.Nil(t, retrieved)

	authErr, ok := err.(*AuthError)
	require.True(t, ok)
	assert.Equal(t, ErrSessionNotFound, authErr.Code)

	// Store and get session
	err = store.Store(ctx, session.UserID, session)
	require.NoError(t, err)

	retrieved, err = store.Get(ctx, session.UserID)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, retrieved.UserID)
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	session := &Session{
		UserID:    "user-123",
		Email:     "user@example.com",
		Provider:  "test_provider",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Delete non-existent session (should not error)
	err := store.Delete(ctx, "nonexistent")
	require.NoError(t, err)

	// Store session
	err = store.Store(ctx, session.UserID, session)
	require.NoError(t, err)

	// Verify session exists
	retrieved, err := store.Get(ctx, session.UserID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Delete session
	err = store.Delete(ctx, session.UserID)
	require.NoError(t, err)

	// Verify session is gone
	retrieved, err = store.Get(ctx, session.UserID)
	require.Error(t, err)
	assert.Nil(t, retrieved)
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	session1 := &Session{
		UserID:    "user-123",
		Email:     "user@example.com",
		Provider:  "provider1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Store initial session
	err := store.Store(ctx, session1.UserID, session1)
	require.NoError(t, err)

	// Update session
	session2 := &Session{
		UserID:    "user-123",
		Email:     "updated@example.com",
		Provider:  "provider2",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}

	err = store.Store(ctx, session2.UserID, session2)
	require.NoError(t, err)

	// Verify updated session
	retrieved, err := store.Get(ctx, session2.UserID)
	require.NoError(t, err)
	assert.Equal(t, session2.Email, retrieved.Email)
	assert.Equal(t, session2.Provider, retrieved.Provider)
}

func TestMemoryStore_Cleanup(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	now := time.Now()

	// Add expired session
	expiredSession := &Session{
		UserID:    "expired-user",
		Email:     "expired@example.com",
		Provider:  "test_provider",
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour), // Expired 1 hour ago
	}

	// Add active session
	activeSession := &Session{
		UserID:    "active-user",
		Email:     "active@example.com",
		Provider:  "test_provider",
		CreatedAt: now,
		ExpiresAt: now.Add(1 * time.Hour), // Expires in 1 hour
	}

	err := store.Store(ctx, expiredSession.UserID, expiredSession)
	require.NoError(t, err)

	err = store.Store(ctx, activeSession.UserID, activeSession)
	require.NoError(t, err)

	// Run cleanup
	deleted, err := store.Cleanup(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Verify expired session is gone
	_, err = store.Get(ctx, expiredSession.UserID)
	require.Error(t, err)

	// Verify active session still exists
	retrieved, err := store.Get(ctx, activeSession.UserID)
	require.NoError(t, err)
	assert.Equal(t, activeSession.UserID, retrieved.UserID)
}

func TestMemoryStore_Cleanup_NoExpiredSessions(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	now := time.Now()

	// Add only active sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			UserID:    "user-" + string(rune(i)),
			Email:     "user@example.com",
			Provider:  "test_provider",
			CreatedAt: now,
			ExpiresAt: now.Add(1 * time.Hour),
		}
		err := store.Store(ctx, session.UserID, session)
		require.NoError(t, err)
	}

	// Run cleanup
	deleted, err := store.Cleanup(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted)
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	const numGoroutines = 10
	const numOpsPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOpsPerGoroutine; j++ {
				session := &Session{
					UserID:    "user-" + string(rune(id)),
					Email:     "user@example.com",
					Provider:  "test_provider",
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				store.Store(ctx, session.UserID, session)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all sessions exist
	for i := 0; i < numGoroutines; i++ {
		userID := "user-" + string(rune(i))
		_, err := store.Get(ctx, userID)
		assert.NoError(t, err)
	}
}

func TestMemoryStore_ConcurrentReadWrite(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Pre-populate store
	for i := 0; i < 10; i++ {
		session := &Session{
			UserID:    "user-" + string(rune(i)),
			Email:     "user@example.com",
			Provider:  "test_provider",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		store.Store(ctx, session.UserID, session)
	}

	done := make(chan bool, 20)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			userID := "user-" + string(rune(id))
			for j := 0; j < 100; j++ {
				store.Get(ctx, userID)
			}
			done <- true
		}(i)
	}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			userID := "user-" + string(rune(id))
			for j := 0; j < 100; j++ {
				session := &Session{
					UserID:    userID,
					Email:     "user@example.com",
					Provider:  "test_provider",
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				store.Store(ctx, session.UserID, session)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// No assertions needed - just verify no race conditions
	t.Log("Concurrent read/write test passed")
}
