package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// sessionContextKey is the context key for storing authenticated sessions.
	sessionContextKey contextKey = "authz:session"
)

// ResourceIDExtractor is a function that extracts the resource ID from an HTTP request.
// It returns the resource ID and whether extraction was successful.
//
// Common extractors:
//   - URLParamExtractor("id") - Extract from URL path parameter
//   - HeaderExtractor("X-Resource-ID") - Extract from HTTP header
//   - Custom extractor - Implement your own extraction logic
type ResourceIDExtractor func(*http.Request) (string, bool)

// MiddlewareConfig holds configuration for authorization middleware.
type MiddlewareConfig struct {
	// Engine is the authorization engine for policy evaluation.
	Engine *Engine

	// ResourceIDExtractor extracts the resource ID from the HTTP request.
	// If nil, the resource ID will be empty (type-level authorization only).
	ResourceIDExtractor ResourceIDExtractor

	// UnauthorizedHandler is called when no authenticated session is found.
	// If nil, returns 401 Unauthorized with a default message.
	UnauthorizedHandler http.Handler

	// ForbiddenHandler is called when authorization is denied.
	// If nil, returns 403 Forbidden with the decision reason.
	ForbiddenHandler func(w http.ResponseWriter, r *http.Request, decision *Decision)
}

// RequirePermission creates middleware that enforces authorization for the specified action and resource type.
//
// The middleware:
//  1. Extracts the authenticated session from the request context
//  2. Extracts the resource ID using the configured extractor (if provided)
//  3. Evaluates the authorization request using the ABAC engine
//  4. Returns 401 Unauthorized if no session is found
//  5. Returns 403 Forbidden if access is denied
//  6. Calls the next handler if access is granted
//
// Example usage:
//
//	engine := authz.NewEngine(policyStore, attrResolver)
//	cfg := authz.MiddlewareConfig{
//	    Engine: engine,
//	    ResourceIDExtractor: authz.URLParamExtractor("planID"),
//	}
//
//	// Protect plan approval endpoint
//	http.Handle("/api/plans/{planID}/approve",
//	    authz.RequirePermission("plan:approve", "plan", cfg)(
//	        http.HandlerFunc(handlePlanApproval),
//	    ),
//	)
func RequirePermission(action, resourceType string, cfg MiddlewareConfig) func(http.Handler) http.Handler {
	if cfg.Engine == nil {
		panic("authz: MiddlewareConfig.Engine cannot be nil")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1. Get authenticated session from context
			session := GetSessionFromContext(ctx)
			if session == nil {
				if cfg.UnauthorizedHandler != nil {
					cfg.UnauthorizedHandler.ServeHTTP(w, r)
					return
				}
				writeUnauthorizedError(w, "no authenticated session")
				return
			}

			// 2. Extract resource ID (if extractor is configured)
			resourceID := ""
			if cfg.ResourceIDExtractor != nil {
				if id, ok := cfg.ResourceIDExtractor(r); ok {
					resourceID = id
				}
				// Note: Missing resource ID is not an error - allows type-level authorization
			}

			// 3. Evaluate authorization
			decision, err := cfg.Engine.Evaluate(ctx, &AuthorizationRequest{
				Subject: session,
				Action:  action,
				Resource: Resource{
					Type: resourceType,
					ID:   resourceID,
				},
				Environment: extractEnvironmentAttributes(r),
			})

			if err != nil {
				// Internal error during policy evaluation
				writeInternalError(w, fmt.Sprintf("authorization evaluation failed: %v", err))
				return
			}

			// 4. Check decision
			if !decision.Allowed {
				if cfg.ForbiddenHandler != nil {
					cfg.ForbiddenHandler(w, r, decision)
					return
				}
				writeForbiddenError(w, decision)
				return
			}

			// 5. Access granted - proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// SetSessionInContext stores an authenticated session in the request context.
// This should be called by authentication middleware before authorization middleware.
func SetSessionInContext(ctx context.Context, session *auth.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// GetSessionFromContext retrieves the authenticated session from the request context.
// Returns nil if no session is found.
func GetSessionFromContext(ctx context.Context) *auth.Session {
	session, ok := ctx.Value(sessionContextKey).(*auth.Session)
	if !ok {
		return nil
	}
	return session
}

// extractEnvironmentAttributes extracts contextual attributes from the HTTP request.
func extractEnvironmentAttributes(r *http.Request) map[string]interface{} {
	env := make(map[string]interface{})

	// Client IP address
	if ip := getClientIP(r); ip != "" {
		env["client_ip"] = ip
	}

	// Request method
	env["method"] = r.Method

	// Request path
	env["path"] = r.URL.Path

	// User agent
	if ua := r.Header.Get("User-Agent"); ua != "" {
		env["user_agent"] = ua
	}

	// Request time
	env["request_time"] = r.Context().Value(http.ServerContextKey)

	return env
}

// getClientIP extracts the client IP address from the request.
// Checks X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header (nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// Common resource ID extractors

// URLParamExtractor creates a ResourceIDExtractor that extracts the resource ID from a URL path parameter.
//
// Example:
//
//	extractor := URLParamExtractor("planID")
//	// For request to /api/plans/123/approve, returns "123"
func URLParamExtractor(paramName string) ResourceIDExtractor {
	return func(r *http.Request) (string, bool) {
		// For Go 1.22+ http.ServeMux pattern matching
		id := r.PathValue(paramName)
		if id != "" {
			return id, true
		}
		return "", false
	}
}

// HeaderExtractor creates a ResourceIDExtractor that extracts the resource ID from an HTTP header.
//
// Example:
//
//	extractor := HeaderExtractor("X-Resource-ID")
func HeaderExtractor(headerName string) ResourceIDExtractor {
	return func(r *http.Request) (string, bool) {
		id := r.Header.Get(headerName)
		if id != "" {
			return id, true
		}
		return "", false
	}
}

// QueryParamExtractor creates a ResourceIDExtractor that extracts the resource ID from a URL query parameter.
//
// Example:
//
//	extractor := QueryParamExtractor("resource_id")
//	// For request to /api/plans?resource_id=123, returns "123"
func QueryParamExtractor(paramName string) ResourceIDExtractor {
	return func(r *http.Request) (string, bool) {
		id := r.URL.Query().Get(paramName)
		if id != "" {
			return id, true
		}
		return "", false
	}
}

// Error response helpers

// ErrorResponse represents a JSON error response.
type ErrorResponse struct {
	Error   string   `json:"error"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

// writeUnauthorizedError writes a 401 Unauthorized response.
func writeUnauthorizedError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   "unauthorized",
		Message: message,
	})
}

// writeForbiddenError writes a 403 Forbidden response with authorization decision details.
func writeForbiddenError(w http.ResponseWriter, decision *Decision) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   "forbidden",
		Message: decision.Reason,
		Details: decision.PolicyIDs,
	})
}

// writeInternalError writes a 500 Internal Server Error response.
func writeInternalError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   "internal_server_error",
		Message: message,
	})
}
