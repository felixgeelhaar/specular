package apikey

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// contextKeyAPIKey is the context key for storing the validated API key.
	contextKeyAPIKey contextKey = "api_key"
)

// Middleware provides HTTP middleware for API key authentication.
type Middleware struct {
	manager *Manager
}

// NewMiddleware creates a new API key middleware.
func NewMiddleware(manager *Manager) *Middleware {
	return &Middleware{
		manager: manager,
	}
}

// RequireAPIKey is HTTP middleware that validates API key authentication.
//
// It extracts the API key from the Authorization header (format: "Bearer <key>")
// and validates it against Vault. If valid, the key metadata is stored in the
// request context for downstream handlers.
//
// Usage:
//
//	http.Handle("/api/resource", middleware.RequireAPIKey(handler))
func (m *Middleware) RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract API key from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.unauthorizedResponse(w, "missing authorization header")
			return
		}

		// Parse "Bearer <key>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			m.unauthorizedResponse(w, "invalid authorization format")
			return
		}

		apiKey := parts[1]
		if apiKey == "" {
			m.unauthorizedResponse(w, "missing API key")
			return
		}

		// Validate API key
		// For efficiency, we'll need to implement a better lookup mechanism
		// For now, we'll assume the organization ID is provided via query param or header
		orgID := r.Header.Get("X-Organization-ID")
		if orgID == "" {
			// Try to extract from query param
			orgID = r.URL.Query().Get("organization_id")
		}

		if orgID == "" {
			m.unauthorizedResponse(w, "organization ID required")
			return
		}

		key, err := m.manager.GetKeyBySecret(ctx, orgID, apiKey)
		if err != nil {
			m.unauthorizedResponse(w, "invalid API key")
			return
		}

		// Check key status
		if key.Status != StatusActive {
			m.unauthorizedResponse(w, "API key is not active")
			return
		}

		// Check expiration
		if !key.ExpiresAt.IsZero() && time.Now().UTC().After(key.ExpiresAt) {
			m.unauthorizedResponse(w, "API key has expired")
			return
		}

		// Store key in context for downstream handlers
		ctx = SetAPIKeyInContext(ctx, key)
		r = r.WithContext(ctx)

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// RequireScopes is HTTP middleware that requires specific scopes on the API key.
//
// Usage:
//
//	http.Handle("/api/resource",
//	    middleware.RequireAPIKey(
//	        middleware.RequireScopes([]string{"read", "write"}, handler)))
func (m *Middleware) RequireScopes(scopes []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := GetAPIKeyFromContext(r.Context())
		if key == nil {
			m.forbiddenResponse(w, "API key required")
			return
		}

		// Check if key has all required scopes
		keyScopes := make(map[string]bool)
		for _, scope := range key.Scopes {
			keyScopes[scope] = true
		}

		for _, requiredScope := range scopes {
			if !keyScopes[requiredScope] {
				m.forbiddenResponse(w, "insufficient scopes")
				return
			}
		}

		// All required scopes present
		next.ServeHTTP(w, r)
	})
}

// RequireAnyScope is HTTP middleware that requires at least one of the specified scopes.
//
// Usage:
//
//	http.Handle("/api/resource",
//	    middleware.RequireAPIKey(
//	        middleware.RequireAnyScope([]string{"admin", "write"}, handler)))
func (m *Middleware) RequireAnyScope(scopes []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := GetAPIKeyFromContext(r.Context())
		if key == nil {
			m.forbiddenResponse(w, "API key required")
			return
		}

		// Check if key has any of the required scopes
		keyScopes := make(map[string]bool)
		for _, scope := range key.Scopes {
			keyScopes[scope] = true
		}

		for _, requiredScope := range scopes {
			if keyScopes[requiredScope] {
				// Has at least one required scope
				next.ServeHTTP(w, r)
				return
			}
		}

		m.forbiddenResponse(w, "insufficient scopes")
	})
}

// SetAPIKeyInContext stores an API key in the request context.
func SetAPIKeyInContext(ctx context.Context, key *APIKey) context.Context {
	return context.WithValue(ctx, contextKeyAPIKey, key)
}

// GetAPIKeyFromContext retrieves the API key from the request context.
func GetAPIKeyFromContext(ctx context.Context) *APIKey {
	if key, ok := ctx.Value(contextKeyAPIKey).(*APIKey); ok {
		return key
	}
	return nil
}

// unauthorizedResponse sends a 401 Unauthorized response.
func (m *Middleware) unauthorizedResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error": "` + message + `"}`))
}

// forbiddenResponse sends a 403 Forbidden response.
func (m *Middleware) forbiddenResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error": "` + message + `"}`))
}

// RateLimitConfig holds configuration for API key rate limiting.
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
}

// WithRateLimit adds rate limiting to the API key middleware.
// This is a placeholder for future implementation.
func (m *Middleware) WithRateLimit(config RateLimitConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement rate limiting based on API key
		// This could use Redis or an in-memory cache
		next.ServeHTTP(w, r)
	})
}
