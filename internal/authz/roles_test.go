package authz

import (
	"testing"
)

// TestPolicyBuilder tests the fluent policy builder API.
func TestPolicyBuilder(t *testing.T) {
	policy := NewPolicyBuilder("org-1", "Test Policy").
		WithID("policy-1").
		WithDescription("Test description").
		AllowRole("admin").
		OnActions("plan:approve", "plan:read").
		OnResourceType("plan").
		WithCondition("$resource.status", OperatorEquals, "pending").
		Build()

	if policy.ID != "policy-1" {
		t.Errorf("expected ID 'policy-1', got %s", policy.ID)
	}
	if policy.OrganizationID != "org-1" {
		t.Errorf("expected OrganizationID 'org-1', got %s", policy.OrganizationID)
	}
	if policy.Name != "Test Policy" {
		t.Errorf("expected Name 'Test Policy', got %s", policy.Name)
	}
	if policy.Description != "Test description" {
		t.Errorf("expected Description 'Test description', got %s", policy.Description)
	}
	if policy.Effect != EffectAllow {
		t.Errorf("expected Effect 'allow', got %s", policy.Effect)
	}
	if len(policy.Principals) != 1 || policy.Principals[0].Role != "admin" {
		t.Error("expected admin principal")
	}
	if len(policy.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(policy.Actions))
	}
	if len(policy.Resources) != 1 || policy.Resources[0] != "plan:*" {
		t.Errorf("expected 'plan:*' resource, got %v", policy.Resources)
	}
	if len(policy.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(policy.Conditions))
	}
	if !policy.Enabled {
		t.Error("expected policy to be enabled")
	}
}

// TestPolicyBuilder_MultipleRoles tests adding multiple roles.
func TestPolicyBuilder_MultipleRoles(t *testing.T) {
	policy := NewPolicyBuilder("org-1", "Multi-Role Policy").
		AllowRole("admin").
		AllowRole("member").
		AllowTeamRole("lead").
		OnActions("*:read").
		OnAllResources().
		Build()

	if len(policy.Principals) != 3 {
		t.Fatalf("expected 3 principals, got %d", len(policy.Principals))
	}

	// Check organization-scoped roles
	if policy.Principals[0].Role != "admin" || policy.Principals[0].Scope != "organization" {
		t.Error("expected admin organization principal")
	}
	if policy.Principals[1].Role != "member" || policy.Principals[1].Scope != "organization" {
		t.Error("expected member organization principal")
	}

	// Check team-scoped role
	if policy.Principals[2].Role != "lead" || policy.Principals[2].Scope != "team" {
		t.Error("expected lead team principal")
	}
}

// TestPolicyBuilder_AttributeBased tests attribute-based principals.
func TestPolicyBuilder_AttributeBased(t *testing.T) {
	policy := NewPolicyBuilder("org-1", "Attribute Policy").
		AllowAttribute("$subject.department", OperatorEquals, "engineering").
		OnActions("*:read").
		OnAllResources().
		Build()

	if len(policy.Principals) != 1 {
		t.Fatalf("expected 1 principal, got %d", len(policy.Principals))
	}

	principal := policy.Principals[0]
	if principal.Attribute != "$subject.department" {
		t.Errorf("expected attribute '$subject.department', got %s", principal.Attribute)
	}
	if principal.Operator != OperatorEquals {
		t.Errorf("expected operator 'equals', got %s", principal.Operator)
	}
	if principal.Value != "engineering" {
		t.Errorf("expected value 'engineering', got %v", principal.Value)
	}
}

// TestPolicyBuilder_Disabled tests creating a disabled policy.
func TestPolicyBuilder_Disabled(t *testing.T) {
	policy := NewPolicyBuilder("org-1", "Disabled Policy").
		AllowRole("admin").
		OnActions("*:delete").
		OnAllResources().
		Disabled().
		Build()

	if policy.Enabled {
		t.Error("expected policy to be disabled")
	}
}

// TestPolicyBuilder_DenyEffect tests creating a deny policy.
func TestPolicyBuilder_DenyEffect(t *testing.T) {
	policy := NewPolicyBuilder("org-1", "Deny Policy").
		WithEffect(EffectDeny).
		AllowRole("member").
		OnActions("*:delete").
		OnAllResources().
		Build()

	if policy.Effect != EffectDeny {
		t.Errorf("expected Effect 'deny', got %s", policy.Effect)
	}
}

// TestNewOwnerPolicy tests the owner policy helper.
func TestNewOwnerPolicy(t *testing.T) {
	policy := NewOwnerPolicy("org-1", "owner-policy-1")

	if policy.ID != "owner-policy-1" {
		t.Errorf("expected ID 'owner-policy-1', got %s", policy.ID)
	}
	if policy.OrganizationID != "org-1" {
		t.Errorf("expected OrganizationID 'org-1', got %s", policy.OrganizationID)
	}
	if policy.Name != "Owner Full Access" {
		t.Errorf("expected Name 'Owner Full Access', got %s", policy.Name)
	}
	if policy.Effect != EffectAllow {
		t.Error("expected allow effect")
	}
	if len(policy.Principals) != 1 || policy.Principals[0].Role != string(RoleOwner) {
		t.Error("expected owner principal")
	}
	if len(policy.Actions) != 1 || policy.Actions[0] != "*" {
		t.Error("expected wildcard actions")
	}
	if len(policy.Resources) != 1 || policy.Resources[0] != "*" {
		t.Error("expected wildcard resources")
	}
	if !policy.Enabled {
		t.Error("expected policy to be enabled")
	}
}

// TestNewAdminPolicy tests the admin policy helper.
func TestNewAdminPolicy(t *testing.T) {
	policy := NewAdminPolicy("org-1", "admin-policy-1")

	if policy.ID != "admin-policy-1" {
		t.Errorf("expected ID 'admin-policy-1', got %s", policy.ID)
	}
	if policy.OrganizationID != "org-1" {
		t.Errorf("expected OrganizationID 'org-1', got %s", policy.OrganizationID)
	}
	if len(policy.Principals) != 1 || policy.Principals[0].Role != string(RoleAdmin) {
		t.Error("expected admin principal")
	}

	// Verify admin actions (approve, create, update, delete, read)
	expectedActions := []string{"*:approve", "*:create", "*:update", "*:delete", "*:read"}
	if len(policy.Actions) != len(expectedActions) {
		t.Errorf("expected %d actions, got %d", len(expectedActions), len(policy.Actions))
	}

	for _, expectedAction := range expectedActions {
		found := false
		for _, action := range policy.Actions {
			if action == expectedAction {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected action %s not found", expectedAction)
		}
	}
}

// TestNewMemberPolicy tests the member policy helper.
func TestNewMemberPolicy(t *testing.T) {
	policy := NewMemberPolicy("org-1", "member-policy-1")

	if policy.ID != "member-policy-1" {
		t.Errorf("expected ID 'member-policy-1', got %s", policy.ID)
	}
	if len(policy.Principals) != 1 || policy.Principals[0].Role != string(RoleMember) {
		t.Error("expected member principal")
	}

	// Verify member actions (create, update, read)
	expectedActions := []string{"*:create", "*:update", "*:read"}
	if len(policy.Actions) != len(expectedActions) {
		t.Errorf("expected %d actions, got %d", len(expectedActions), len(policy.Actions))
	}
}

// TestNewViewerPolicy tests the viewer policy helper.
func TestNewViewerPolicy(t *testing.T) {
	policy := NewViewerPolicy("org-1", "viewer-policy-1")

	if policy.ID != "viewer-policy-1" {
		t.Errorf("expected ID 'viewer-policy-1', got %s", policy.ID)
	}
	if len(policy.Principals) != 1 || policy.Principals[0].Role != string(RoleViewer) {
		t.Error("expected viewer principal")
	}

	// Verify viewer actions (read, list only)
	expectedActions := []string{"*:read", "*:list"}
	if len(policy.Actions) != len(expectedActions) {
		t.Errorf("expected %d actions, got %d", len(expectedActions), len(policy.Actions))
	}
}

// TestNewResourceSpecificPolicy tests the resource-specific policy helper.
func TestNewResourceSpecificPolicy(t *testing.T) {
	policy := NewResourceSpecificPolicy(
		"org-1",
		"plan-approve-policy",
		"Admins Approve Plans",
		"admin",
		"plan:approve",
		"plan",
	)

	if policy.ID != "plan-approve-policy" {
		t.Errorf("expected ID 'plan-approve-policy', got %s", policy.ID)
	}
	if policy.Name != "Admins Approve Plans" {
		t.Errorf("expected Name 'Admins Approve Plans', got %s", policy.Name)
	}
	if len(policy.Principals) != 1 || policy.Principals[0].Role != "admin" {
		t.Error("expected admin principal")
	}
	if len(policy.Actions) != 1 || policy.Actions[0] != "plan:approve" {
		t.Error("expected plan:approve action")
	}
	if len(policy.Resources) != 1 || policy.Resources[0] != "plan:*" {
		t.Errorf("expected 'plan:*' resource, got %v", policy.Resources)
	}
}

// TestNewConditionalPolicy tests the conditional policy helper.
func TestNewConditionalPolicy(t *testing.T) {
	condition := Condition{
		Attribute: "$resource.status",
		Operator:  OperatorEquals,
		Value:     "pending",
	}

	policy := NewConditionalPolicy(
		"org-1",
		"conditional-policy",
		"Approve Pending Plans",
		"admin",
		[]string{"plan:approve"},
		"plan",
		condition,
	)

	if policy.ID != "conditional-policy" {
		t.Errorf("expected ID 'conditional-policy', got %s", policy.ID)
	}
	if len(policy.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(policy.Conditions))
	}
	if policy.Conditions[0].Attribute != "$resource.status" {
		t.Errorf("expected condition attribute '$resource.status', got %s", policy.Conditions[0].Attribute)
	}
	if policy.Conditions[0].Operator != OperatorEquals {
		t.Errorf("expected condition operator 'equals', got %s", policy.Conditions[0].Operator)
	}
	if policy.Conditions[0].Value != "pending" {
		t.Errorf("expected condition value 'pending', got %v", policy.Conditions[0].Value)
	}
}

// TestNewTeamPolicy tests the team policy helper.
func TestNewTeamPolicy(t *testing.T) {
	policy := NewTeamPolicy(
		"org-1",
		"team-policy",
		"Team Lead Policy",
		"lead",
		[]string{"build:execute", "build:read"},
		[]string{"build:*"},
	)

	if policy.ID != "team-policy" {
		t.Errorf("expected ID 'team-policy', got %s", policy.ID)
	}
	if policy.Name != "Team Lead Policy" {
		t.Errorf("expected Name 'Team Lead Policy', got %s", policy.Name)
	}
	if len(policy.Principals) != 1 || policy.Principals[0].Role != "lead" || policy.Principals[0].Scope != "team" {
		t.Error("expected team-scoped lead principal")
	}
	if len(policy.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(policy.Actions))
	}
	if len(policy.Resources) != 1 || policy.Resources[0] != "build:*" {
		t.Errorf("expected 'build:*' resource, got %v", policy.Resources)
	}
}

// TestNewDenyPolicy tests the deny policy helper.
func TestNewDenyPolicy(t *testing.T) {
	principals := []Principal{
		{Role: "member", Scope: "organization"},
	}

	policy := NewDenyPolicy(
		"org-1",
		"deny-policy",
		"Deny Delete",
		principals,
		[]string{"*:delete"},
		[]string{"*"},
	)

	if policy.ID != "deny-policy" {
		t.Errorf("expected ID 'deny-policy', got %s", policy.ID)
	}
	if policy.Effect != EffectDeny {
		t.Errorf("expected Effect 'deny', got %s", policy.Effect)
	}
	if len(policy.Principals) != 1 || policy.Principals[0].Role != "member" {
		t.Error("expected member principal")
	}
	if len(policy.Actions) != 1 || policy.Actions[0] != "*:delete" {
		t.Error("expected *:delete action")
	}
}

// TestFormatAction tests the action formatting helper.
func TestFormatAction(t *testing.T) {
	tests := []struct {
		resourceType string
		action       string
		expected     string
	}{
		{"plan", ActionApprove, "plan:approve"},
		{"build", ActionExecute, "build:execute"},
		{"policy", ActionRead, "policy:read"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatAction(tt.resourceType, tt.action)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestFormatResource tests the resource formatting helper.
func TestFormatResource(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		expected     string
	}{
		{"wildcard with asterisk", "plan", "*", "plan:*"},
		{"wildcard with empty", "plan", "", "plan:*"},
		{"specific resource", "plan", "123", "plan:123"},
		{"build resource", "build", "456", "build:456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatResource(tt.resourceType, tt.resourceID)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRoleConstants tests that role constants are defined correctly.
func TestRoleConstants(t *testing.T) {
	if RoleOwner != "owner" {
		t.Errorf("expected RoleOwner 'owner', got %s", RoleOwner)
	}
	if RoleAdmin != "admin" {
		t.Errorf("expected RoleAdmin 'admin', got %s", RoleAdmin)
	}
	if RoleMember != "member" {
		t.Errorf("expected RoleMember 'member', got %s", RoleMember)
	}
	if RoleViewer != "viewer" {
		t.Errorf("expected RoleViewer 'viewer', got %s", RoleViewer)
	}
}

// TestActionConstants tests that action constants are defined correctly.
func TestActionConstants(t *testing.T) {
	expected := map[string]string{
		"approve": ActionApprove,
		"create":  ActionCreate,
		"read":    ActionRead,
		"update":  ActionUpdate,
		"delete":  ActionDelete,
		"list":    ActionList,
		"execute": ActionExecute,
	}

	for expectedValue, constant := range expected {
		if constant != expectedValue {
			t.Errorf("expected action constant %s, got %s", expectedValue, constant)
		}
	}
}

// TestPolicyBuilderChaining tests fluent API chaining.
func TestPolicyBuilderChaining(t *testing.T) {
	// Test that all builder methods return the builder for chaining
	builder := NewPolicyBuilder("org-1", "Chain Test")

	// This should not panic if chaining works correctly
	policy := builder.
		WithID("test-1").
		WithDescription("test").
		WithEffect(EffectAllow).
		AllowRole("admin").
		AllowTeamRole("lead").
		AllowAttribute("$subject.department", OperatorEquals, "eng").
		OnActions("read", "write").
		OnResources("resource:1", "resource:2").
		OnAllResources().
		OnResourceType("plan").
		WithCondition("$resource.status", OperatorEquals, "active").
		Disabled().
		Build()

	if policy == nil {
		t.Fatal("expected policy to be built")
	}
}
