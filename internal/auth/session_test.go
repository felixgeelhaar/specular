package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionManager_CreateSession(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer)

	tests := []struct {
		name    string
		session *Session
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid session",
			session: &Session{
				UserID:   "user-123",
				Email:    "user@example.com",
				Provider: "test_provider",
			},
			wantErr: false,
		},
		{
			name: "missing user ID",
			session: &Session{
				Email:    "user@example.com",
				Provider: "test_provider",
			},
			wantErr: true,
			errMsg:  "user ID cannot be empty",
		},
		{
			name: "missing email",
			session: &Session{
				UserID:   "user-123",
				Provider: "test_provider",
			},
			wantErr: true,
			errMsg:  "email cannot be empty",
		},
		{
			name: "missing provider",
			session: &Session{
				UserID: "user-123",
				Email:  "user@example.com",
			},
			wantErr: true,
			errMsg:  "provider cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			token, err := sm.CreateSession(ctx, tt.session)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
				assert.NotEmpty(t, tt.session.Token)

				// Verify timestamps were set
				assert.False(t, tt.session.CreatedAt.IsZero())
				assert.False(t, tt.session.ExpiresAt.IsZero())
				assert.True(t, tt.session.ExpiresAt.After(tt.session.CreatedAt))
			}
		})
	}
}

func TestSessionManager_ValidateSessionToken(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer)

	ctx := context.Background()
	session := &Session{
		UserID:   "user-123",
		Email:    "user@example.com",
		Provider: "test_provider",
		Attributes: map[string]interface{}{
			"name": "Test User",
		},
	}

	// Create valid token
	validToken, err := sm.CreateSession(ctx, session)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
		errCode string
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
			errCode: ErrTokenInvalid,
		},
		{
			name:    "malformed token",
			token:   "not-a-valid-jwt",
			wantErr: true,
			errCode: ErrTokenMalformed,
		},
		{
			name:    "token with wrong signature",
			token:   validToken[:len(validToken)-10] + "corrupted",
			wantErr: true,
			errCode: ErrTokenMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := sm.ValidateSessionToken(tt.token)

			if tt.wantErr {
				require.Error(t, err)
				authErr, ok := err.(*AuthError)
				if ok {
					assert.Equal(t, tt.errCode, authErr.Code)
				}
				assert.Nil(t, claims)
			} else {
				require.NoError(t, err)
				require.NotNil(t, claims)

				// Verify claims
				assert.Equal(t, session.UserID, claims.UserID)
				assert.Equal(t, session.Email, claims.Email)
				assert.Equal(t, session.Provider, claims.Provider)
				assert.Equal(t, issuer, claims.Issuer)
				assert.Equal(t, session.UserID, claims.Subject)
				assert.NotNil(t, claims.Attributes)
				assert.Equal(t, "Test User", claims.Attributes["name"])
			}
		})
	}
}

func TestSessionManager_ValidateSessionToken_Expired(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer).WithTokenDuration(1*time.Millisecond, 1*time.Hour)

	ctx := context.Background()
	session := &Session{
		UserID:   "user-123",
		Email:    "user@example.com",
		Provider: "test_provider",
	}

	// Create token with very short expiration
	token, err := sm.CreateSession(ctx, session)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validate expired token
	claims, err := sm.ValidateSessionToken(token)
	require.Error(t, err)
	assert.Nil(t, claims)

	// JWT library may return either malformed or expired depending on timing
	authErr, ok := err.(*AuthError)
	require.True(t, ok)
	assert.Contains(t, []string{ErrTokenExpired, ErrTokenMalformed}, authErr.Code)
}

func TestSessionManager_ValidateSessionToken_WrongIssuer(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	sm1 := NewSessionManager(signingKey, "issuer-1")
	sm2 := NewSessionManager(signingKey, "issuer-2")

	ctx := context.Background()
	session := &Session{
		UserID:   "user-123",
		Email:    "user@example.com",
		Provider: "test_provider",
	}

	// Create token with issuer-1
	token, err := sm1.CreateSession(ctx, session)
	require.NoError(t, err)

	// Validate with issuer-2 (should fail)
	claims, err := sm2.ValidateSessionToken(token)
	require.Error(t, err)
	assert.Nil(t, claims)

	authErr, ok := err.(*AuthError)
	require.True(t, ok)
	assert.Equal(t, ErrTokenInvalid, authErr.Code)
	assert.Contains(t, authErr.Message, "invalid issuer")
}

func TestSessionManager_CreateRefreshToken(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer)

	ctx := context.Background()
	session := &Session{
		UserID:   "user-123",
		Email:    "user@example.com",
		Provider: "test_provider",
	}

	refreshToken, err := sm.CreateRefreshToken(ctx, session)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshToken)

	// Validate refresh token
	claims, err := sm.ValidateRefreshToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, claims.UserID)
	assert.Equal(t, session.Email, claims.Email)
	assert.Equal(t, session.Provider, claims.Provider)

	// Refresh token should have longer expiration
	assert.True(t, claims.ExpiresAt.Time.After(time.Now().Add(6*24*time.Hour)))
}

func TestSessionManager_CreateRefreshToken_MissingUserID(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer)

	ctx := context.Background()
	session := &Session{
		Email:    "user@example.com",
		Provider: "test_provider",
	}

	refreshToken, err := sm.CreateRefreshToken(ctx, session)
	require.Error(t, err)
	assert.Empty(t, refreshToken)

	authErr, ok := err.(*AuthError)
	require.True(t, ok)
	assert.Equal(t, ErrSessionInvalid, authErr.Code)
}

func TestSessionManager_WithTokenDuration(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer).
		WithTokenDuration(30*time.Minute, 30*24*time.Hour)

	ctx := context.Background()
	session := &Session{
		UserID:   "user-123",
		Email:    "user@example.com",
		Provider: "test_provider",
	}

	// Create access token
	_, err := sm.CreateSession(ctx, session)
	require.NoError(t, err)

	// Verify expiration is approximately 30 minutes
	expectedExpiry := time.Now().Add(30 * time.Minute)
	assert.WithinDuration(t, expectedExpiry, session.ExpiresAt, 1*time.Second)

	// Create refresh token
	refreshToken, err := sm.CreateRefreshToken(ctx, session)
	require.NoError(t, err)

	claims, err := sm.ValidateRefreshToken(refreshToken)
	require.NoError(t, err)

	// Verify refresh token expiration is approximately 30 days
	expectedRefreshExpiry := time.Now().Add(30 * 24 * time.Hour)
	assert.WithinDuration(t, expectedRefreshExpiry, claims.ExpiresAt.Time, 1*time.Second)
}

func TestParseSessionToken(t *testing.T) {
	signingKey := []byte("test-secret-key-at-least-32-bytes-long")
	issuer := "specular-test"
	sm := NewSessionManager(signingKey, issuer)

	ctx := context.Background()
	session := &Session{
		UserID:   "user-123",
		Email:    "user@example.com",
		Provider: "test_provider",
		Attributes: map[string]interface{}{
			"role": "admin",
		},
	}

	token, err := sm.CreateSession(ctx, session)
	require.NoError(t, err)

	// Parse without verification
	claims, err := ParseSessionToken(token)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, claims.UserID)
	assert.Equal(t, session.Email, claims.Email)
	assert.Equal(t, session.Provider, claims.Provider)
	assert.Equal(t, "admin", claims.Attributes["role"])
}

func TestParseSessionToken_Malformed(t *testing.T) {
	claims, err := ParseSessionToken("not-a-jwt")
	require.Error(t, err)
	assert.Nil(t, claims)

	authErr, ok := err.(*AuthError)
	require.True(t, ok)
	assert.Equal(t, ErrTokenMalformed, authErr.Code)
}
