package auth

import "time"

// Session represents an authenticated user session.
// This is a stub for authz development - will be replaced when M9.2.1 merges.
type Session struct {
	UserID    string
	Email     string
	Provider  string
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time

	// Authorization fields (will be merged with M9.2.1 Session)
	OrganizationID   string   // Tenant/organization user belongs to
	OrganizationRole string   // owner, admin, member, viewer
	TeamID           *string  // Optional team membership
	TeamRole         *string  // Team-specific role override
	Permissions      []string // Cached permissions for performance

	Attributes map[string]interface{}
}
