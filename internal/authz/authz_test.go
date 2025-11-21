package authz

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/specular/internal/auth"
)

func TestEngine_Evaluate_DefaultDeny(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	session := &auth.Session{
		UserID:           "user-123",
		Email:            "user@example.com",
		OrganizationID:   "org-1",
		OrganizationRole: string(RoleAdmin),
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, decision.Allowed, "should default to deny with no policies")
	assert.Contains(t, decision.Reason, "no matching policy")
}

func TestEngine_Evaluate_ExplicitDenyWins(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	// Create allow policy
	allowPolicy := &Policy{
		ID:             "allow-policy",
		OrganizationID: "org-1",
		Name:           "Allow All",
		Effect:         EffectAllow,
		Principals:     []Principal{{Role: string(RoleAdmin)}},
		Actions:        []string{"*"},
		Resources:      []string{"*"},
		Enabled:        true,
	}

	// Create deny policy (should override allow)
	denyPolicy := &Policy{
		ID:             "deny-policy",
		OrganizationID: "org-1",
		Name:           "Deny Plan Approval",
		Effect:         EffectDeny,
		Principals:     []Principal{{Role: string(RoleAdmin)}},
		Actions:        []string{"plan:approve"},
		Resources:      []string{"plan:*"},
		Enabled:        true,
	}

	require.NoError(t, store.CreatePolicy(context.Background(), allowPolicy))
	require.NoError(t, store.CreatePolicy(context.Background(), denyPolicy))

	session := &auth.Session{
		UserID:           "user-123",
		OrganizationID:   "org-1",
		OrganizationRole: string(RoleAdmin),
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, decision.Allowed, "explicit deny should win over allow")
	assert.Contains(t, decision.Reason, "explicitly denied")
	assert.Contains(t, decision.PolicyIDs, "deny-policy")
}

func TestEngine_Evaluate_AllowWithMatchingPolicy(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	policy := &Policy{
		ID:             "admin-policy",
		OrganizationID: "org-1",
		Name:           "Admin Policy",
		Effect:         EffectAllow,
		Principals:     []Principal{{Role: string(RoleAdmin)}},
		Actions:        []string{"plan:approve", "plan:create"},
		Resources:      []string{"plan:*"},
		Enabled:        true,
	}

	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-123",
		OrganizationID:   "org-1",
		OrganizationRole: string(RoleAdmin),
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, decision.Allowed, "should allow with matching policy")
	assert.Contains(t, decision.Reason, "granted")
	assert.Contains(t, decision.PolicyIDs, "admin-policy")
}

func TestEngine_Evaluate_ConditionFails(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)

	// Set resource attributes
	resourceStore.SetResourceAttributes("plan", "plan-123", Attributes{
		"organization_id": "org-2", // Different org
		"status":          "pending",
	})

	policy := &Policy{
		ID:             "org-isolation-policy",
		OrganizationID: "org-1",
		Name:           "Organization Isolation",
		Effect:         EffectAllow,
		Principals:     []Principal{{Role: string(RoleAdmin)}},
		Actions:        []string{"plan:*"},
		Resources:      []string{"plan:*"},
		Conditions: []Condition{
			{
				Attribute: "$resource.organization_id",
				Operator:  OperatorEquals,
				Value:     "$subject.organization_id",
			},
		},
		Enabled: true,
	}

	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-123",
		OrganizationID:   "org-1", // Different from resource org
		OrganizationRole: string(RoleAdmin),
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, decision.Allowed, "should deny when condition fails")
}

func TestEngine_Evaluate_ConditionPasses(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)

	// Set resource attributes
	resourceStore.SetResourceAttributes("plan", "plan-123", Attributes{
		"organization_id": "org-1", // Same org
		"status":          "pending",
	})

	policy := &Policy{
		ID:             "org-isolation-policy",
		OrganizationID: "org-1",
		Name:           "Organization Isolation",
		Effect:         EffectAllow,
		Principals:     []Principal{{Role: string(RoleAdmin)}},
		Actions:        []string{"plan:*"},
		Resources:      []string{"plan:*"},
		Conditions: []Condition{
			{
				Attribute: "$resource.organization_id",
				Operator:  OperatorEquals,
				Value:     "$subject.organization_id",
			},
		},
		Enabled: true,
	}

	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-123",
		OrganizationID:   "org-1", // Same as resource org
		OrganizationRole: string(RoleAdmin),
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, decision.Allowed, "should allow when condition passes")
}

func TestEngine_Evaluate_WildcardActions(t *testing.T) {
	tests := []struct {
		name          string
		policyAction  string
		requestAction string
		shouldMatch   bool
	}{
		{
			name:          "exact match",
			policyAction:  "plan:approve",
			requestAction: "plan:approve",
			shouldMatch:   true,
		},
		{
			name:          "wildcard all",
			policyAction:  "*",
			requestAction: "plan:approve",
			shouldMatch:   true,
		},
		{
			name:          "prefix wildcard",
			policyAction:  "plan:*",
			requestAction: "plan:approve",
			shouldMatch:   true,
		},
		{
			name:          "no match",
			policyAction:  "build:*",
			requestAction: "plan:approve",
			shouldMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewInMemoryPolicyStore()
			resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
			engine := NewEngine(store, resolver)

			policy := &Policy{
				ID:             "test-policy",
				OrganizationID: "org-1",
				Name:           "Test Policy",
				Effect:         EffectAllow,
				Principals:     []Principal{{Role: string(RoleAdmin)}},
				Actions:        []string{tt.policyAction},
				Resources:      []string{"*"},
				Enabled:        true,
			}

			require.NoError(t, store.CreatePolicy(context.Background(), policy))

			session := &auth.Session{
				UserID:           "user-123",
				OrganizationID:   "org-1",
				OrganizationRole: string(RoleAdmin),
			}

			req := &AuthorizationRequest{
				Subject: session,
				Action:  tt.requestAction,
				Resource: Resource{
					Type: "plan",
					ID:   "plan-123",
				},
			}

			decision, err := engine.Evaluate(context.Background(), req)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldMatch, decision.Allowed, tt.name)
		})
	}
}

func TestEngine_Evaluate_DisabledPolicy(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	policy := &Policy{
		ID:             "disabled-policy",
		OrganizationID: "org-1",
		Name:           "Disabled Policy",
		Effect:         EffectAllow,
		Principals:     []Principal{{Role: string(RoleAdmin)}},
		Actions:        []string{"*"},
		Resources:      []string{"*"},
		Enabled:        false, // Disabled
	}

	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-123",
		OrganizationID:   "org-1",
		OrganizationRole: string(RoleAdmin),
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, decision.Allowed, "disabled policies should not be evaluated")
}

func TestEngine_Evaluate_NoSubject(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	req := &AuthorizationRequest{
		Subject: nil, // No subject
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "no authenticated subject")
}

func TestEngine_Evaluate_NoOrganization(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	session := &auth.Session{
		UserID:         "user-123",
		OrganizationID: "", // No organization
	}

	req := &AuthorizationRequest{
		Subject: session,
		Action:  "plan:approve",
		Resource: Resource{
			Type: "plan",
			ID:   "plan-123",
		},
	}

	decision, err := engine.Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "not associated with organization")
}

func TestEvaluateOperator_Equals(t *testing.T) {
	engine := &Engine{}

	assert.True(t, engine.evaluateOperator(OperatorEquals, "foo", "foo"))
	assert.False(t, engine.evaluateOperator(OperatorEquals, "foo", "bar"))
	assert.True(t, engine.evaluateOperator(OperatorEquals, 123, 123))
	assert.False(t, engine.evaluateOperator(OperatorEquals, 123, 456))
}

func TestEvaluateOperator_In(t *testing.T) {
	engine := &Engine{}

	assert.True(t, engine.evaluateOperator(OperatorIn, "foo", []interface{}{"foo", "bar"}))
	assert.False(t, engine.evaluateOperator(OperatorIn, "baz", []interface{}{"foo", "bar"}))
	assert.True(t, engine.evaluateOperator(OperatorIn, 2, []interface{}{1, 2, 3}))
}

func TestEvaluateOperator_GreaterThan(t *testing.T) {
	engine := &Engine{}

	assert.True(t, engine.evaluateOperator(OperatorGreaterThan, 10, 5))
	assert.False(t, engine.evaluateOperator(OperatorGreaterThan, 5, 10))
	assert.False(t, engine.evaluateOperator(OperatorGreaterThan, 5, 5))
	assert.True(t, engine.evaluateOperator(OperatorGreaterThan, 10.5, 10.2))
}

func TestEvaluateOperator_StringLike(t *testing.T) {
	engine := &Engine{}

	assert.True(t, engine.evaluateOperator(OperatorStringLike, "foobar", "*bar"))
	assert.True(t, engine.evaluateOperator(OperatorStringLike, "foobar", "foo*"))
	assert.True(t, engine.evaluateOperator(OperatorStringLike, "foobar", "*"))
	assert.False(t, engine.evaluateOperator(OperatorStringLike, "foobar", "*baz"))
}

func TestEvaluateOperator_Exists(t *testing.T) {
	engine := &Engine{}

	assert.True(t, engine.evaluateOperator(OperatorExists, "value", nil))
	assert.False(t, engine.evaluateOperator(OperatorExists, nil, nil))
	assert.True(t, engine.evaluateOperator(OperatorNotExists, nil, nil))
	assert.False(t, engine.evaluateOperator(OperatorNotExists, "value", nil))
}

func TestResolveAttribute_SubjectAttributes(t *testing.T) {
	engine := &Engine{}

	subjectAttrs := Attributes{
		"user_id":         "user-123",
		"organization_id": "org-1",
		"role":            "admin",
	}

	assert.Equal(t, "user-123", engine.resolveAttribute("$subject.user_id", subjectAttrs, nil, nil))
	assert.Equal(t, "org-1", engine.resolveAttribute("$subject.organization_id", subjectAttrs, nil, nil))
	assert.Equal(t, "admin", engine.resolveAttribute("$subject.role", subjectAttrs, nil, nil))
	assert.Nil(t, engine.resolveAttribute("$subject.nonexistent", subjectAttrs, nil, nil))
}

func TestResolveAttribute_ResourceAttributes(t *testing.T) {
	engine := &Engine{}

	resourceAttrs := Attributes{
		"status":          "pending",
		"organization_id": "org-1",
		"owner_id":        "user-123",
	}

	assert.Equal(t, "pending", engine.resolveAttribute("$resource.status", nil, resourceAttrs, nil))
	assert.Equal(t, "org-1", engine.resolveAttribute("$resource.organization_id", nil, resourceAttrs, nil))
	assert.Equal(t, "user-123", engine.resolveAttribute("$resource.owner_id", nil, resourceAttrs, nil))
}

func TestResolveAttribute_EnvironmentAttributes(t *testing.T) {
	engine := &Engine{}

	env := map[string]interface{}{
		"time":       time.Now(),
		"ip_address": "192.168.1.1",
	}

	assert.Equal(t, "192.168.1.1", engine.resolveAttribute("$environment.ip_address", nil, nil, env))
	assert.NotNil(t, engine.resolveAttribute("$environment.time", nil, nil, env))
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		text    string
		pattern string
		match   bool
	}{
		{"foobar", "*", true},
		{"foobar", "foo*", true},
		{"foobar", "*bar", true},
		{"foobar", "foobar", true},
		{"foobar", "foo", false},
		{"foobar", "bar", false},
		{"foobar", "*baz", false},
		{"", "", true},
		{"foo", "", false},
	}

	for _, tt := range tests {
		result := matchGlob(tt.text, tt.pattern)
		assert.Equal(t, tt.match, result, "matchGlob(%q, %q)", tt.text, tt.pattern)
	}
}

func TestCompareNumbers(t *testing.T) {
	tests := []struct {
		left     interface{}
		right    interface{}
		expected int
	}{
		{10, 5, 1},
		{5, 10, -1},
		{5, 5, 0},
		{10.5, 10.2, 1},
		{10.2, 10.5, -1},
		{float64(5), 5, 0},
		{int64(5), float32(5), 0},
	}

	for _, tt := range tests {
		result := compareNumbers(tt.left, tt.right)
		assert.Equal(t, tt.expected, result, "compareNumbers(%v, %v)", tt.left, tt.right)
	}
}
