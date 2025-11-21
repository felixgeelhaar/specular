package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTokenFromRequest(t *testing.T) {
	tests := []struct {
		name          string
		setupRequest  func(*http.Request)
		expectedToken string
		description   string
	}{
		{
			name: "extract from Authorization header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer test-token-123")
			},
			expectedToken: "test-token-123",
			description:   "should extract token from Bearer header",
		},
		{
			name: "extract from Authorization header - case insensitive",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "bearer test-token-456")
			},
			expectedToken: "test-token-456",
			description:   "should handle lowercase bearer",
		},
		{
			name: "extract from cookie",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  "session_token",
					Value: "cookie-token-789",
				})
			},
			expectedToken: "cookie-token-789",
			description:   "should extract token from cookie",
		},
		{
			name: "extract from query parameter",
			setupRequest: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("token", "query-token-101")
				r.URL.RawQuery = q.Encode()
			},
			expectedToken: "query-token-101",
			description:   "should extract token from query parameter",
		},
		{
			name: "priority: header over cookie",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer header-token")
				r.AddCookie(&http.Cookie{
					Name:  "session_token",
					Value: "cookie-token",
				})
			},
			expectedToken: "header-token",
			description:   "should prioritize Authorization header",
		},
		{
			name: "priority: cookie over query",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  "session_token",
					Value: "cookie-token",
				})
				q := r.URL.Query()
				q.Add("token", "query-token")
				r.URL.RawQuery = q.Encode()
			},
			expectedToken: "cookie-token",
			description:   "should prioritize cookie over query param",
		},
		{
			name: "no token provided",
			setupRequest: func(r *http.Request) {
				// No token in request
			},
			expectedToken: "",
			description:   "should return empty string when no token",
		},
		{
			name: "malformed Authorization header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "InvalidFormat")
			},
			expectedToken: "",
			description:   "should return empty for malformed header",
		},
		{
			name: "Authorization header without Bearer",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
			},
			expectedToken: "",
			description:   "should ignore non-Bearer auth schemes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)

			token := ExtractTokenFromRequest(req)
			assert.Equal(t, tt.expectedToken, token, tt.description)
		})
	}
}

// mockProvider is a test authentication provider
type mockProvider struct{}

func (m *mockProvider) Name() string { return "test_provider" }

func (m *mockProvider) Authenticate(ctx context.Context, req *http.Request) (*Session, error) {
	return nil, nil
}

func (m *mockProvider) ValidateSession(ctx context.Context, session *Session) error {
	return nil // Always valid for tests
}

func (m *mockProvider) RefreshSession(ctx context.Context, session *Session) (*Session, error) {
	return session, nil
}

func (m *mockProvider) Logout(ctx context.Context, session *Session) error {
	return nil
}

func TestMiddleware_RequireAuth(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer)
	store := NewMemoryStore()
	manager := NewManager(store)

	// Register mock provider
	err := manager.Register(&mockProvider{})
	require.NoError(t, err)

	middleware := NewMiddleware(manager, sm)

	// Create a valid session and token
	session := &Session{
		UserID:    "user-123",
		Email:     "user@example.com",
		Provider:  "test_provider",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	validToken, err := sm.CreateSession(context.Background(), session)
	require.NoError(t, err)

	// Store session in session store
	err = store.Store(context.Background(), session.UserID, session)
	require.NoError(t, err)

	tests := []struct {
		name             string
		setupRequest     func(*http.Request)
		expectedStatus   int
		expectSessionCtx bool
		description      string
	}{
		{
			name: "valid token in header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validToken)
			},
			expectedStatus:   http.StatusOK,
			expectSessionCtx: true,
			description:      "should allow request with valid token",
		},
		{
			name: "valid token in cookie",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  "session_token",
					Value: validToken,
				})
			},
			expectedStatus:   http.StatusOK,
			expectSessionCtx: true,
			description:      "should allow request with valid cookie",
		},
		{
			name: "no token provided",
			setupRequest: func(r *http.Request) {
				// No token
			},
			expectedStatus:   http.StatusForbidden,
			expectSessionCtx: false,
			description:      "should reject request without token",
		},
		{
			name: "invalid token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid-token")
			},
			expectedStatus:   http.StatusForbidden, // ErrSessionInvalid
			expectSessionCtx: false,
			description:      "should reject request with invalid token",
		},
		{
			name: "malformed token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer not-a-jwt")
			},
			expectedStatus:   http.StatusForbidden, // ErrSessionInvalid
			expectSessionCtx: false,
			description:      "should reject request with malformed token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			var sessionFound bool
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sess := GetSession(r.Context())
				sessionFound = (sess != nil)
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			wrapped := middleware.RequireAuth(handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			tt.setupRequest(req)

			// Execute request
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			// Verify status
			assert.Equal(t, tt.expectedStatus, rec.Code, tt.description)

			// Verify session in context
			if tt.expectSessionCtx {
				assert.True(t, sessionFound, "session should be in context")
			} else {
				assert.False(t, sessionFound, "session should not be in context")
			}
		})
	}
}

func TestMiddleware_OptionalAuth(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer)
	store := NewMemoryStore()
	manager := NewManager(store)

	// Register mock provider
	err := manager.Register(&mockProvider{})
	require.NoError(t, err)

	middleware := NewMiddleware(manager, sm)

	// Create a valid session and token
	session := &Session{
		UserID:    "user-123",
		Email:     "user@example.com",
		Provider:  "test_provider",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	validToken, err := sm.CreateSession(context.Background(), session)
	require.NoError(t, err)

	// Store session in session store
	err = store.Store(context.Background(), session.UserID, session)
	require.NoError(t, err)

	tests := []struct {
		name             string
		setupRequest     func(*http.Request)
		expectSessionCtx bool
		description      string
	}{
		{
			name: "valid token provided",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validToken)
			},
			expectSessionCtx: true,
			description:      "should attach session with valid token",
		},
		{
			name: "no token provided",
			setupRequest: func(r *http.Request) {
				// No token
			},
			expectSessionCtx: false,
			description:      "should allow request without token",
		},
		{
			name: "invalid token provided",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid-token")
			},
			expectSessionCtx: false,
			description:      "should allow request with invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			var sessionFound bool
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sess := GetSession(r.Context())
				sessionFound = (sess != nil)
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			wrapped := middleware.OptionalAuth(handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/optional", nil)
			tt.setupRequest(req)

			// Execute request
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			// OptionalAuth always allows request
			assert.Equal(t, http.StatusOK, rec.Code, "should always allow request")

			// Verify session in context
			assert.Equal(t, tt.expectSessionCtx, sessionFound, tt.description)
		})
	}
}

func TestGetSession(t *testing.T) {
	session := &Session{
		UserID: "user-123",
		Email:  "user@example.com",
	}

	t.Run("session in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), sessionContextKey, session)
		retrieved := GetSession(ctx)
		assert.NotNil(t, retrieved)
		assert.Equal(t, session.UserID, retrieved.UserID)
		assert.Equal(t, session.Email, retrieved.Email)
	})

	t.Run("no session in context", func(t *testing.T) {
		ctx := context.Background()
		retrieved := GetSession(ctx)
		assert.Nil(t, retrieved)
	})

	t.Run("wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), sessionContextKey, "not-a-session")
		retrieved := GetSession(ctx)
		assert.Nil(t, retrieved)
	})
}

func TestMustGetSession(t *testing.T) {
	session := &Session{
		UserID: "user-123",
		Email:  "user@example.com",
	}

	t.Run("session in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), sessionContextKey, session)
		retrieved := MustGetSession(ctx)
		assert.NotNil(t, retrieved)
		assert.Equal(t, session.UserID, retrieved.UserID)
		assert.Equal(t, session.Email, retrieved.Email)
	})

	t.Run("panic when no session", func(t *testing.T) {
		ctx := context.Background()
		assert.Panics(t, func() {
			MustGetSession(ctx)
		}, "should panic when no session in context")
	})
}

func TestSetSessionCookie(t *testing.T) {
	token := "test-token-123"
	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	t.Run("set cookie with secure=true", func(t *testing.T) {
		rec := httptest.NewRecorder()
		SetSessionCookie(rec, token, expiresAt, true)

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)

		cookie := cookies[0]
		assert.Equal(t, "session_token", cookie.Name)
		assert.Equal(t, token, cookie.Value)
		assert.Equal(t, "/", cookie.Path)
		assert.True(t, cookie.HttpOnly, "should be HttpOnly")
		assert.True(t, cookie.Secure, "should be Secure in production")
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.Greater(t, cookie.MaxAge, 0, "should have positive MaxAge")
	})

	t.Run("set cookie with secure=false", func(t *testing.T) {
		rec := httptest.NewRecorder()
		SetSessionCookie(rec, token, expiresAt, false)

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)

		cookie := cookies[0]
		assert.False(t, cookie.Secure, "should not be Secure in development")
	})

	t.Run("set cookie with expired timestamp", func(t *testing.T) {
		rec := httptest.NewRecorder()
		expiredTime := time.Now().Add(-1 * time.Hour).Unix()
		SetSessionCookie(rec, token, expiredTime, true)

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)

		cookie := cookies[0]
		assert.Equal(t, 0, cookie.MaxAge, "should have MaxAge=0 for expired")
	})
}

func TestClearSessionCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	ClearSessionCookie(rec)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, "session_token", cookie.Name)
	assert.Equal(t, "", cookie.Value, "should have empty value")
	assert.Equal(t, "/", cookie.Path)
	assert.Equal(t, -1, cookie.MaxAge, "should have MaxAge=-1 to delete")
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}
