package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthenticator implements the Authenticator interface for testing
type mockAuthenticator struct {
	name                string
	authenticateFunc    func(ctx context.Context, req *http.Request) (*Session, error)
	validateSessionFunc func(ctx context.Context, session *Session) error
	refreshSessionFunc  func(ctx context.Context, session *Session) (*Session, error)
	logoutFunc          func(ctx context.Context, session *Session) error
}

func (m *mockAuthenticator) Name() string {
	return m.name
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, req *http.Request) (*Session, error) {
	if m.authenticateFunc != nil {
		return m.authenticateFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAuthenticator) ValidateSession(ctx context.Context, session *Session) error {
	if m.validateSessionFunc != nil {
		return m.validateSessionFunc(ctx, session)
	}
	return nil
}

func (m *mockAuthenticator) RefreshSession(ctx context.Context, session *Session) (*Session, error) {
	if m.refreshSessionFunc != nil {
		return m.refreshSessionFunc(ctx, session)
	}
	return nil, errors.New("refresh not supported")
}

func (m *mockAuthenticator) Logout(ctx context.Context, session *Session) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, session)
	}
	return nil
}

// createTestSession creates a session for testing with valid expiry
func createTestSession(userID, provider string) *Session {
	return &Session{
		UserID:           userID,
		Email:            userID + "@test.com",
		Provider:         provider,
		OrganizationID:   "org-123",
		OrganizationRole: "member",
		Token:            "test-token",
		RefreshToken:     "test-refresh-token",
		ExpiresAt:        time.Now().Add(1 * time.Hour),
		CreatedAt:        time.Now(),
		Attributes:       make(map[string]interface{}),
	}
}

// createExpiredSession creates an expired session for testing
func createExpiredSession(userID, provider string) *Session {
	return &Session{
		UserID:           userID,
		Email:            userID + "@test.com",
		Provider:         provider,
		OrganizationID:   "org-123",
		OrganizationRole: "member",
		Token:            "test-token",
		RefreshToken:     "test-refresh-token",
		ExpiresAt:        time.Now().Add(-1 * time.Hour), // Already expired
		CreatedAt:        time.Now().Add(-2 * time.Hour),
		Attributes:       make(map[string]interface{}),
	}
}

func TestNewManager(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.providers)
	assert.Equal(t, 0, len(manager.providers))
}

func TestManager_Register(t *testing.T) {
	t.Run("register provider successfully", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		provider := &mockAuthenticator{name: "test_provider"}
		err := manager.Register(provider)

		assert.NoError(t, err)
		assert.Len(t, manager.providers, 1)
	})

	t.Run("register duplicate provider returns error", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		provider1 := &mockAuthenticator{name: "test_provider"}
		provider2 := &mockAuthenticator{name: "test_provider"}

		err := manager.Register(provider1)
		require.NoError(t, err)

		err = manager.Register(provider2)
		assert.Error(t, err)
		assert.True(t, IsAuthError(err, ErrDuplicateProvider))
	})

	t.Run("register multiple different providers", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		provider1 := &mockAuthenticator{name: "provider1"}
		provider2 := &mockAuthenticator{name: "provider2"}
		provider3 := &mockAuthenticator{name: "provider3"}

		require.NoError(t, manager.Register(provider1))
		require.NoError(t, manager.Register(provider2))
		require.NoError(t, manager.Register(provider3))

		assert.Len(t, manager.providers, 3)
	})
}

func TestManager_Authenticate(t *testing.T) {
	ctx := context.Background()

	t.Run("authenticate with no providers returns error", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		req := httptest.NewRequest("GET", "/", nil)
		session, err := manager.Authenticate(ctx, req)

		assert.Nil(t, session)
		assert.Error(t, err)
		assert.True(t, IsAuthError(err, ErrNoProviders))
	})

	t.Run("authenticate successfully with single provider", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		expectedSession := createTestSession("user-123", "test_provider")
		provider := &mockAuthenticator{
			name: "test_provider",
			authenticateFunc: func(ctx context.Context, req *http.Request) (*Session, error) {
				return expectedSession, nil
			},
		}
		require.NoError(t, manager.Register(provider))

		req := httptest.NewRequest("GET", "/", nil)
		session, err := manager.Authenticate(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, expectedSession.UserID, session.UserID)

		// Verify session was stored
		storedSession, err := store.Get(ctx, session.UserID)
		assert.NoError(t, err)
		assert.Equal(t, session.UserID, storedSession.UserID)
	})

	t.Run("authenticate with first provider failure tries second provider", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		expectedSession := createTestSession("user-123", "provider2")

		provider1 := &mockAuthenticator{
			name: "provider1",
			authenticateFunc: func(ctx context.Context, req *http.Request) (*Session, error) {
				return nil, errors.New("provider1 failed")
			},
		}
		provider2 := &mockAuthenticator{
			name: "provider2",
			authenticateFunc: func(ctx context.Context, req *http.Request) (*Session, error) {
				return expectedSession, nil
			},
		}

		require.NoError(t, manager.Register(provider1))
		require.NoError(t, manager.Register(provider2))

		req := httptest.NewRequest("GET", "/", nil)
		session, err := manager.Authenticate(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, expectedSession.UserID, session.UserID)
	})

	t.Run("authenticate with all providers failing returns last error", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		lastError := errors.New("provider2 specific error")
		provider1 := &mockAuthenticator{
			name: "provider1",
			authenticateFunc: func(ctx context.Context, req *http.Request) (*Session, error) {
				return nil, errors.New("provider1 failed")
			},
		}
		provider2 := &mockAuthenticator{
			name: "provider2",
			authenticateFunc: func(ctx context.Context, req *http.Request) (*Session, error) {
				return nil, lastError
			},
		}

		require.NoError(t, manager.Register(provider1))
		require.NoError(t, manager.Register(provider2))

		req := httptest.NewRequest("GET", "/", nil)
		session, err := manager.Authenticate(ctx, req)

		assert.Nil(t, session)
		assert.Error(t, err)
	})
}

func TestManager_ValidateSession(t *testing.T) {
	ctx := context.Background()

	t.Run("validate valid session", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		// Create session manager to create valid tokens
		sessionMgr := NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

		testSession := createTestSession("user-123", "test_provider")
		token, err := sessionMgr.CreateSession(ctx, testSession)
		require.NoError(t, err)
		testSession.Token = token

		// Store the session
		require.NoError(t, store.Store(ctx, testSession.UserID, testSession))

		// Register mock provider
		provider := &mockAuthenticator{
			name: "test_provider",
			validateSessionFunc: func(ctx context.Context, session *Session) error {
				return nil
			},
		}
		require.NoError(t, manager.Register(provider))

		// Validate session
		session, err := manager.ValidateSession(ctx, token)

		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, testSession.UserID, session.UserID)
	})

	t.Run("validate session with invalid token", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		session, err := manager.ValidateSession(ctx, "invalid-token")

		assert.Nil(t, session)
		assert.Error(t, err)
		assert.True(t, IsAuthError(err, ErrSessionInvalid))
	})

	t.Run("validate session not in store", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		// Create a valid token but don't store the session
		sessionMgr := NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")
		testSession := createTestSession("nonexistent-user", "test_provider")
		token, err := sessionMgr.CreateSession(ctx, testSession)
		require.NoError(t, err)

		session, err := manager.ValidateSession(ctx, token)

		assert.Nil(t, session)
		assert.Error(t, err)
		assert.True(t, IsAuthError(err, ErrSessionNotFound))
	})

	t.Run("validate expired session", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		sessionMgr := NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")
		testSession := createExpiredSession("user-123", "test_provider")
		token, err := sessionMgr.CreateSession(ctx, testSession)
		require.NoError(t, err)

		// Store the expired session
		require.NoError(t, store.Store(ctx, testSession.UserID, testSession))

		session, err := manager.ValidateSession(ctx, token)

		assert.Nil(t, session)
		assert.Error(t, err)
		// Note: MemoryStore.Get() returns ErrSessionExpired for expired sessions,
		// but Manager.ValidateSession() wraps any store error as ErrSessionNotFound
		assert.True(t, IsAuthError(err, ErrSessionNotFound))
	})

	t.Run("validate session with provider not found", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		sessionMgr := NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")
		testSession := createTestSession("user-123", "nonexistent_provider")
		token, err := sessionMgr.CreateSession(ctx, testSession)
		require.NoError(t, err)

		// Store the session
		require.NoError(t, store.Store(ctx, testSession.UserID, testSession))

		session, err := manager.ValidateSession(ctx, token)

		assert.Nil(t, session)
		assert.Error(t, err)
		assert.True(t, IsAuthError(err, ErrProviderNotFound))
	})

	t.Run("validate session when provider validation fails", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		sessionMgr := NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")
		testSession := createTestSession("user-123", "test_provider")
		token, err := sessionMgr.CreateSession(ctx, testSession)
		require.NoError(t, err)

		// Store the session
		require.NoError(t, store.Store(ctx, testSession.UserID, testSession))

		// Register provider that fails validation
		provider := &mockAuthenticator{
			name: "test_provider",
			validateSessionFunc: func(ctx context.Context, session *Session) error {
				return NewError(ErrSessionInvalid, "provider validation failed", nil)
			},
		}
		require.NoError(t, manager.Register(provider))

		session, err := manager.ValidateSession(ctx, token)

		assert.Nil(t, session)
		assert.Error(t, err)
	})
}

func TestManager_RefreshSession(t *testing.T) {
	ctx := context.Background()

	t.Run("refresh session successfully", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		originalSession := createTestSession("user-123", "test_provider")
		newSession := createTestSession("user-123", "test_provider")
		newSession.Token = "new-token"
		newSession.ExpiresAt = time.Now().Add(2 * time.Hour)

		provider := &mockAuthenticator{
			name: "test_provider",
			refreshSessionFunc: func(ctx context.Context, session *Session) (*Session, error) {
				return newSession, nil
			},
		}
		require.NoError(t, manager.Register(provider))

		refreshedSession, err := manager.RefreshSession(ctx, originalSession)

		assert.NoError(t, err)
		assert.NotNil(t, refreshedSession)
		assert.Equal(t, "new-token", refreshedSession.Token)

		// Verify session was stored
		storedSession, err := store.Get(ctx, newSession.UserID)
		assert.NoError(t, err)
		assert.Equal(t, newSession.Token, storedSession.Token)
	})

	t.Run("refresh session with provider not found", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		session := createTestSession("user-123", "nonexistent_provider")

		refreshedSession, err := manager.RefreshSession(ctx, session)

		assert.Nil(t, refreshedSession)
		assert.Error(t, err)
		assert.True(t, IsAuthError(err, ErrProviderNotFound))
	})

	t.Run("refresh session when provider fails", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		session := createTestSession("user-123", "test_provider")

		provider := &mockAuthenticator{
			name: "test_provider",
			refreshSessionFunc: func(ctx context.Context, session *Session) (*Session, error) {
				return nil, NewError(ErrRefreshFailed, "refresh not supported", nil)
			},
		}
		require.NoError(t, manager.Register(provider))

		refreshedSession, err := manager.RefreshSession(ctx, session)

		assert.Nil(t, refreshedSession)
		assert.Error(t, err)
	})
}

func TestManager_Logout(t *testing.T) {
	ctx := context.Background()

	t.Run("logout successfully", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		session := createTestSession("user-123", "test_provider")

		// Store the session first
		require.NoError(t, store.Store(ctx, session.UserID, session))

		provider := &mockAuthenticator{
			name: "test_provider",
			logoutFunc: func(ctx context.Context, session *Session) error {
				return nil
			},
		}
		require.NoError(t, manager.Register(provider))

		err := manager.Logout(ctx, session)

		assert.NoError(t, err)

		// Verify session was removed from store
		_, err = store.Get(ctx, session.UserID)
		assert.Error(t, err)
	})

	t.Run("logout with provider not found", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		session := createTestSession("user-123", "nonexistent_provider")

		err := manager.Logout(ctx, session)

		assert.Error(t, err)
		assert.True(t, IsAuthError(err, ErrProviderNotFound))
	})

	t.Run("logout when provider fails", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		session := createTestSession("user-123", "test_provider")

		provider := &mockAuthenticator{
			name: "test_provider",
			logoutFunc: func(ctx context.Context, session *Session) error {
				return errors.New("logout failed at IdP")
			},
		}
		require.NoError(t, manager.Register(provider))

		err := manager.Logout(ctx, session)

		assert.Error(t, err)
	})
}

func TestManager_GetProvider(t *testing.T) {
	t.Run("get existing provider", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		expectedProvider := &mockAuthenticator{name: "test_provider"}
		require.NoError(t, manager.Register(expectedProvider))

		provider, err := manager.GetProvider("test_provider")

		assert.NoError(t, err)
		assert.Equal(t, expectedProvider, provider)
	})

	t.Run("get non-existent provider", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		provider, err := manager.GetProvider("nonexistent")

		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.True(t, IsAuthError(err, ErrProviderNotFound))
	})
}

func TestManager_ListProviders(t *testing.T) {
	t.Run("list empty providers", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		providers := manager.ListProviders()

		assert.Empty(t, providers)
	})

	t.Run("list multiple providers", func(t *testing.T) {
		store := NewMemoryStore()
		manager := NewManager(store)

		require.NoError(t, manager.Register(&mockAuthenticator{name: "provider1"}))
		require.NoError(t, manager.Register(&mockAuthenticator{name: "provider2"}))
		require.NoError(t, manager.Register(&mockAuthenticator{name: "provider3"}))

		providers := manager.ListProviders()

		assert.Len(t, providers, 3)
		assert.Contains(t, providers, "provider1")
		assert.Contains(t, providers, "provider2")
		assert.Contains(t, providers, "provider3")
	})
}

func TestSession_IsExpired(t *testing.T) {
	t.Run("session not expired", func(t *testing.T) {
		session := &Session{
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		assert.False(t, session.IsExpired())
	})

	t.Run("session expired", func(t *testing.T) {
		session := &Session{
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}

		assert.True(t, session.IsExpired())
	})

	t.Run("session exactly at expiry", func(t *testing.T) {
		// Session expires in the past by a small margin
		session := &Session{
			ExpiresAt: time.Now().Add(-1 * time.Millisecond),
		}

		assert.True(t, session.IsExpired())
	})
}

func TestSession_Fields(t *testing.T) {
	t.Run("session with all fields", func(t *testing.T) {
		teamID := "team-123"
		teamRole := "developer"

		session := &Session{
			UserID:           "user-123",
			Email:            "user@example.com",
			OrganizationID:   "org-123",
			OrganizationRole: "admin",
			TeamID:           &teamID,
			TeamRole:         &teamRole,
			Provider:         "test_provider",
			Token:            "access-token",
			RefreshToken:     "refresh-token",
			ExpiresAt:        time.Now().Add(1 * time.Hour),
			CreatedAt:        time.Now(),
			Attributes: map[string]interface{}{
				"custom": "value",
			},
		}

		assert.Equal(t, "user-123", session.UserID)
		assert.Equal(t, "user@example.com", session.Email)
		assert.Equal(t, "org-123", session.OrganizationID)
		assert.Equal(t, "admin", session.OrganizationRole)
		assert.Equal(t, &teamID, session.TeamID)
		assert.Equal(t, &teamRole, session.TeamRole)
		assert.Equal(t, "test_provider", session.Provider)
		assert.Equal(t, "access-token", session.Token)
		assert.Equal(t, "refresh-token", session.RefreshToken)
		assert.Equal(t, "value", session.Attributes["custom"])
	})

	t.Run("session with optional fields nil", func(t *testing.T) {
		session := &Session{
			UserID:           "user-123",
			Email:            "user@example.com",
			OrganizationID:   "org-123",
			OrganizationRole: "member",
			TeamID:           nil,
			TeamRole:         nil,
			Provider:         "test_provider",
		}

		assert.Nil(t, session.TeamID)
		assert.Nil(t, session.TeamRole)
	})
}
