package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// sessionContextKey is the context key for storing session information.
	sessionContextKey contextKey = "auth_session"
)

// Middleware provides HTTP middleware for authentication.
type Middleware struct {
	manager        *Manager
	sessionManager *SessionManager
}

// NewMiddleware creates a new authentication middleware.
func NewMiddleware(manager *Manager, sessionManager *SessionManager) *Middleware {
	return &Middleware{
		manager:        manager,
		sessionManager: sessionManager,
	}
}

// RequireAuth is middleware that requires valid authentication.
//
// Returns 401 Unauthorized if token is invalid or expired.
// Returns 403 Forbidden if no authentication token is provided.
// Attaches session to request context on success.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.authenticate(r)
		if err != nil {
			m.writeAuthError(w, err)
			return
		}

		// Attach session to context
		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth is middleware that attempts authentication but allows unauthenticated requests.
//
// If authentication succeeds, attaches session to request context.
// If authentication fails or no token provided, continues without session.
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.authenticate(r)
		if err == nil && session != nil {
			// Attach session to context
			ctx := context.WithValue(r.Context(), sessionContextKey, session)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// authenticate extracts and validates the authentication token from the request.
func (m *Middleware) authenticate(r *http.Request) (*Session, error) {
	// Extract token from request
	token := ExtractTokenFromRequest(r)
	if token == "" {
		return nil, NewError(ErrSessionInvalid, "no authentication token provided", nil)
	}

	// Validate token using Manager (handles JWT validation and provider checks)
	session, err := m.manager.ValidateSession(r.Context(), token)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// writeAuthError writes an authentication error response.
func (m *Middleware) writeAuthError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	authErr, ok := err.(*AuthError)
	if !ok {
		// Generic error
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "authentication_failed",
			"message": err.Error(),
		})
		return
	}

	// Determine HTTP status code based on error code
	statusCode := http.StatusUnauthorized
	switch authErr.Code {
	case ErrSessionInvalid:
		statusCode = http.StatusForbidden
	case ErrSessionExpired, ErrTokenExpired:
		statusCode = http.StatusUnauthorized
	case ErrTokenInvalid, ErrTokenMalformed:
		statusCode = http.StatusUnauthorized
	case ErrInvalidCredentials:
		statusCode = http.StatusUnauthorized
	default:
		statusCode = http.StatusUnauthorized
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   authErr.Code,
		"message": authErr.Message,
		"context": authErr.Context,
	})
}

// ExtractTokenFromRequest extracts authentication token from HTTP request.
//
// Checks in order:
//  1. Authorization header (Bearer token)
//  2. Cookie (session cookie named "session_token")
//  3. Query parameter (token)
//
// Returns empty string if no token found.
func ExtractTokenFromRequest(r *http.Request) string {
	// 1. Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// 2. Check session cookie
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 3. Check query parameter
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	return ""
}

// GetSession retrieves the session from the request context.
//
// Returns nil if no session is attached to the context.
// Use this in handlers that have RequireAuth or OptionalAuth middleware.
func GetSession(ctx context.Context) *Session {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	if !ok {
		return nil
	}
	return session
}

// MustGetSession retrieves the session from the request context.
//
// Panics if no session is attached to the context.
// Use this in handlers that have RequireAuth middleware (session is guaranteed).
func MustGetSession(ctx context.Context) *Session {
	session := GetSession(ctx)
	if session == nil {
		panic("no session in context (did you forget RequireAuth middleware?)")
	}
	return session
}

// SetSessionCookie sets the session cookie on the response.
//
// Cookie attributes:
//   - HttpOnly: Prevents JavaScript access (XSS protection)
//   - Secure: HTTPS only (in production)
//   - SameSite: Lax (CSRF protection)
//   - Path: / (available to all routes)
//   - MaxAge: Set based on session expiration
func SetSessionCookie(w http.ResponseWriter, token string, expiresAt int64, secure bool) {
	maxAge := int(expiresAt - nowUnix())
	if maxAge < 0 {
		maxAge = 0
	}

	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure, // Set to true in production (HTTPS)
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}

// ClearSessionCookie clears the session cookie.
//
// Sets MaxAge to -1 to delete the cookie immediately.
func ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}

// nowUnix returns the current Unix timestamp (can be mocked in tests).
var nowUnix = func() int64 {
	return time.Now().Unix()
}
