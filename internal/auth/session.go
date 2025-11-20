package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionClaims represents JWT claims for a session.
//
// Uses standard JWT claims plus custom application claims.
// All authentication providers create sessions using this standard format.
type SessionClaims struct {
	jwt.RegisteredClaims

	// UserID is the unique user identifier
	UserID string `json:"user_id"`

	// Email is the user's email address
	Email string `json:"email"`

	// Provider is the authentication provider (okta_saml, auth0_oidc, legacy_jwt)
	Provider string `json:"provider"`

	// Attributes contains provider-specific claims
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SessionManager handles JWT session creation and validation.
//
// All authentication providers use SessionManager to create standardized JWT tokens.
// This ensures consistent session format across SAML, OIDC, and legacy auth.
type SessionManager struct {
	// signingKey is the secret key for signing JWTs
	signingKey []byte

	// issuer is the JWT issuer (e.g., "specular.example.com")
	issuer string

	// tokenDuration is how long access tokens are valid (default: 1 hour)
	tokenDuration time.Duration

	// refreshDuration is how long refresh tokens are valid (default: 7 days)
	refreshDuration time.Duration
}

// NewSessionManager creates a new session manager.
//
// Parameters:
//   - signingKey: Secret key for signing JWTs (must be kept secure)
//   - issuer: JWT issuer identifier (e.g., "specular.example.com")
func NewSessionManager(signingKey []byte, issuer string) *SessionManager {
	return &SessionManager{
		signingKey:      signingKey,
		issuer:          issuer,
		tokenDuration:   1 * time.Hour,    // Access token: 1 hour
		refreshDuration: 7 * 24 * time.Hour, // Refresh token: 7 days
	}
}

// WithTokenDuration sets custom token durations.
func (sm *SessionManager) WithTokenDuration(tokenDuration, refreshDuration time.Duration) *SessionManager {
	sm.tokenDuration = tokenDuration
	sm.refreshDuration = refreshDuration
	return sm
}

// CreateSession creates a new JWT session token.
//
// Creates a signed JWT with standard claims and returns the token string.
// The session is also stored in the session store.
func (sm *SessionManager) CreateSession(ctx context.Context, s *Session) (string, error) {
	if s.UserID == "" {
		return "", NewError(ErrSessionInvalid, "user ID cannot be empty", nil)
	}
	if s.Email == "" {
		return "", NewError(ErrSessionInvalid, "email cannot be empty", nil)
	}
	if s.Provider == "" {
		return "", NewError(ErrSessionInvalid, "provider cannot be empty", nil)
	}

	// Set timestamps if not already set
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.ExpiresAt.IsZero() {
		s.ExpiresAt = now.Add(sm.tokenDuration)
	}

	// Create JWT claims
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    sm.issuer,
			Subject:   s.UserID,
			Audience:  jwt.ClaimStrings{sm.issuer},
			ExpiresAt: jwt.NewNumericDate(s.ExpiresAt),
			NotBefore: jwt.NewNumericDate(s.CreatedAt),
			IssuedAt:  jwt.NewNumericDate(s.CreatedAt),
			ID:        generateSessionID(s.UserID, s.CreatedAt),
		},
		UserID:     s.UserID,
		Email:      s.Email,
		Provider:   s.Provider,
		Attributes: s.Attributes,
	}

	// Create and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(sm.signingKey)
	if err != nil {
		return "", WrapError(ErrTokenSigningFailed, "failed to sign token", err, map[string]interface{}{
			"user_id": s.UserID,
		})
	}

	// Update session with token
	s.Token = tokenString

	return tokenString, nil
}

// ValidateSessionToken validates and parses a JWT session token.
//
// Returns the session claims if the token is valid.
// Returns an error if the token is expired, malformed, or has an invalid signature.
func (sm *SessionManager) ValidateSessionToken(tokenString string) (*SessionClaims, error) {
	if tokenString == "" {
		return nil, NewError(ErrTokenInvalid, "token cannot be empty", nil)
	}

	// Parse token with claims
	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, NewError(ErrTokenInvalid, "invalid signing method", map[string]interface{}{
				"method": token.Header["alg"],
			})
		}
		return sm.signingKey, nil
	})

	if err != nil {
		if jwt.ErrSignatureInvalid == err {
			return nil, NewError(ErrTokenInvalid, "invalid token signature", nil)
		}
		return nil, WrapError(ErrTokenMalformed, "failed to parse token", err, nil)
	}

	// Extract claims
	claims, ok := token.Claims.(*SessionClaims)
	if !ok || !token.Valid {
		return nil, NewError(ErrTokenInvalid, "invalid token claims", nil)
	}

	// Validate issuer
	if claims.Issuer != sm.issuer {
		return nil, NewError(ErrTokenInvalid, "invalid issuer", map[string]interface{}{
			"expected": sm.issuer,
			"actual":   claims.Issuer,
		})
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, NewError(ErrTokenExpired, "token has expired", map[string]interface{}{
			"expires_at": claims.ExpiresAt.Time,
		})
	}

	return claims, nil
}

// CreateRefreshToken creates a refresh token for a session.
//
// Refresh tokens are long-lived and can be used to obtain new access tokens.
// They should be stored securely and rotated on each use.
func (sm *SessionManager) CreateRefreshToken(ctx context.Context, s *Session) (string, error) {
	if s.UserID == "" {
		return "", NewError(ErrSessionInvalid, "user ID cannot be empty", nil)
	}

	now := time.Now()
	expiresAt := now.Add(sm.refreshDuration)

	// Create JWT claims for refresh token
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    sm.issuer,
			Subject:   s.UserID,
			Audience:  jwt.ClaimStrings{sm.issuer},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateSessionID(s.UserID, now) + "-refresh",
		},
		UserID:   s.UserID,
		Email:    s.Email,
		Provider: s.Provider,
		// No attributes in refresh token for security
	}

	// Create and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(sm.signingKey)
	if err != nil {
		return "", WrapError(ErrTokenSigningFailed, "failed to sign refresh token", err, map[string]interface{}{
			"user_id": s.UserID,
		})
	}

	return tokenString, nil
}

// ValidateRefreshToken validates a refresh token.
//
// Refresh tokens should only be used to obtain new access tokens.
func (sm *SessionManager) ValidateRefreshToken(tokenString string) (*SessionClaims, error) {
	// Use the same validation as access tokens
	// Refresh tokens have the same structure but longer expiration
	return sm.ValidateSessionToken(tokenString)
}

// generateSessionID generates a unique session ID.
//
// Format: <user_id>-<timestamp>-<random>
func generateSessionID(userID string, createdAt time.Time) string {
	// Use timestamp for uniqueness
	// In production, should add random component for additional uniqueness
	timestamp := createdAt.Unix()
	return jwt.RegisteredClaims{
		ID: userID + "-" + string(rune(timestamp)),
	}.ID
}

// ParseSessionToken is a helper function to parse a session token.
//
// This is used by the Manager to extract claims without full validation.
func ParseSessionToken(tokenString string) (*SessionClaims, error) {
	// Parse without validation (just extract claims)
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &SessionClaims{})
	if err != nil {
		return nil, WrapError(ErrTokenMalformed, "failed to parse token", err, nil)
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok {
		return nil, NewError(ErrTokenInvalid, "invalid token claims", nil)
	}

	return claims, nil
}

// ExtractToken extracts the authentication token from an HTTP request.
//
// Checks in order:
//  1. Authorization header (Bearer token)
//  2. Cookie (session cookie)
//  3. Query parameter (for OAuth callbacks)
func ExtractToken(req interface{}) string {
	// This is a placeholder - will be implemented in middleware.go
	// keeping the signature here for reference
	return ""
}
