package authz

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryPolicyStore provides an in-memory policy store for testing and development.
type InMemoryPolicyStore struct {
	mu       sync.RWMutex
	policies map[string]*Policy // key: policy.ID
	byOrg    map[string][]string // organizationID -> []policyID
}

// NewInMemoryPolicyStore creates a new in-memory policy store.
func NewInMemoryPolicyStore() *InMemoryPolicyStore {
	return &InMemoryPolicyStore{
		policies: make(map[string]*Policy),
		byOrg:    make(map[string][]string),
	}
}

// LoadPolicies loads all enabled policies for an organization.
func (s *InMemoryPolicyStore) LoadPolicies(ctx context.Context, organizationID string) ([]*Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policyIDs, ok := s.byOrg[organizationID]
	if !ok {
		return []*Policy{}, nil
	}

	var policies []*Policy
	for _, policyID := range policyIDs {
		policy, ok := s.policies[policyID]
		if !ok {
			continue
		}

		// Only return enabled policies
		if policy.Enabled {
			// Return a copy to avoid external mutations
			policyCopy := *policy
			policies = append(policies, &policyCopy)
		}
	}

	return policies, nil
}

// CreatePolicy creates a new policy.
func (s *InMemoryPolicyStore) CreatePolicy(ctx context.Context, policy *Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID cannot be empty")
	}
	if policy.OrganizationID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if policy already exists
	if _, exists := s.policies[policy.ID]; exists {
		return fmt.Errorf("policy with ID %s already exists", policy.ID)
	}

	// Set timestamps
	now := time.Now()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	// Set default version
	if policy.Version == 0 {
		policy.Version = 1
	}

	// Set default enabled
	if !policy.Enabled {
		policy.Enabled = true
	}

	// Store policy
	s.policies[policy.ID] = policy

	// Index by organization
	s.byOrg[policy.OrganizationID] = append(s.byOrg[policy.OrganizationID], policy.ID)

	return nil
}

// UpdatePolicy updates an existing policy.
func (s *InMemoryPolicyStore) UpdatePolicy(ctx context.Context, policy *Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if policy exists
	existing, exists := s.policies[policy.ID]
	if !exists {
		return fmt.Errorf("policy with ID %s not found", policy.ID)
	}

	// Preserve created timestamp
	policy.CreatedAt = existing.CreatedAt

	// Update timestamp
	policy.UpdatedAt = time.Now()

	// Increment version
	policy.Version = existing.Version + 1

	// Update policy
	s.policies[policy.ID] = policy

	return nil
}

// DeletePolicy deletes a policy.
func (s *InMemoryPolicyStore) DeletePolicy(ctx context.Context, policyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy, exists := s.policies[policyID]
	if !exists {
		return fmt.Errorf("policy with ID %s not found", policyID)
	}

	// Remove from organization index
	orgPolicies := s.byOrg[policy.OrganizationID]
	for i, id := range orgPolicies {
		if id == policyID {
			s.byOrg[policy.OrganizationID] = append(orgPolicies[:i], orgPolicies[i+1:]...)
			break
		}
	}

	// Remove policy
	delete(s.policies, policyID)

	return nil
}

// GetPolicy retrieves a specific policy.
func (s *InMemoryPolicyStore) GetPolicy(ctx context.Context, policyID string) (*Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, exists := s.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy with ID %s not found", policyID)
	}

	// Return a copy
	policyCopy := *policy
	return &policyCopy, nil
}

// LoadBuiltInPolicies loads standard role-based policies into the store.
// These provide RBAC-style roles as ABAC abstractions.
func (s *InMemoryPolicyStore) LoadBuiltInPolicies(organizationID string) error {
	// Owner role: Full control
	ownerPolicy := &Policy{
		ID:             fmt.Sprintf("%s-owner-policy", organizationID),
		OrganizationID: organizationID,
		Name:           "Owner Full Control",
		Description:    "Organization owners have full access to all resources",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: string(RoleOwner), Scope: "organization"},
		},
		Actions:   []string{"*"}, // All actions
		Resources: []string{"*"}, // All resources
		Enabled:   true,
	}

	// Admin role: Can approve, create, update, delete (but not manage members)
	adminPolicy := &Policy{
		ID:             fmt.Sprintf("%s-admin-policy", organizationID),
		OrganizationID: organizationID,
		Name:           "Admin Policy",
		Description:    "Admins can perform most operations except member management",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: string(RoleAdmin), Scope: "organization"},
		},
		Actions: []string{
			"plan:*",
			"build:*",
			"drift:*",
			"policy:read",
			"policy:create",
			"policy:update",
		},
		Resources: []string{"*"},
		Conditions: []Condition{
			{
				Attribute: "resource.organization_id",
				Operator:  OperatorEquals,
				Value:     "$subject.organization_id",
			},
		},
		Enabled: true,
	}

	// Member role: Can create and update (not approve or delete)
	memberPolicy := &Policy{
		ID:             fmt.Sprintf("%s-member-policy", organizationID),
		OrganizationID: organizationID,
		Name:           "Member Policy",
		Description:    "Members can create and update resources they own",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: string(RoleMember), Scope: "organization"},
		},
		Actions: []string{
			"plan:create",
			"plan:read",
			"plan:update",
			"build:create",
			"build:read",
			"drift:read",
		},
		Resources: []string{"*"},
		Conditions: []Condition{
			{
				Attribute: "resource.organization_id",
				Operator:  OperatorEquals,
				Value:     "$subject.organization_id",
			},
		},
		Enabled: true,
	}

	// Viewer role: Read-only access
	viewerPolicy := &Policy{
		ID:             fmt.Sprintf("%s-viewer-policy", organizationID),
		OrganizationID: organizationID,
		Name:           "Viewer Policy",
		Description:    "Viewers have read-only access to resources",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: string(RoleViewer), Scope: "organization"},
		},
		Actions: []string{
			"*:read",
			"*:list",
		},
		Resources: []string{"*"},
		Conditions: []Condition{
			{
				Attribute: "resource.organization_id",
				Operator:  OperatorEquals,
				Value:     "$subject.organization_id",
			},
		},
		Enabled: true,
	}

	// Create all policies
	policies := []*Policy{ownerPolicy, adminPolicy, memberPolicy, viewerPolicy}
	for _, policy := range policies {
		if err := s.CreatePolicy(context.Background(), policy); err != nil {
			return err
		}
	}

	return nil
}
