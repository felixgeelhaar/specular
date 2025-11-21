package authz

import (
	"fmt"
	"time"
)

// PolicyBuilder provides a fluent API for building authorization policies.
type PolicyBuilder struct {
	policy *Policy
}

// NewPolicyBuilder creates a new policy builder.
func NewPolicyBuilder(organizationID, name string) *PolicyBuilder {
	return &PolicyBuilder{
		policy: &Policy{
			OrganizationID: organizationID,
			Name:           name,
			Effect:         EffectAllow,
			Principals:     []Principal{},
			Actions:        []string{},
			Resources:      []string{},
			Conditions:     []Condition{},
			Enabled:        true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
}

// WithID sets the policy ID.
func (b *PolicyBuilder) WithID(id string) *PolicyBuilder {
	b.policy.ID = id
	return b
}

// WithDescription sets the policy description.
func (b *PolicyBuilder) WithDescription(description string) *PolicyBuilder {
	b.policy.Description = description
	return b
}

// WithEffect sets the policy effect (allow or deny).
func (b *PolicyBuilder) WithEffect(effect Effect) *PolicyBuilder {
	b.policy.Effect = effect
	return b
}

// AllowRole adds a role-based principal to the policy.
func (b *PolicyBuilder) AllowRole(role string) *PolicyBuilder {
	b.policy.Principals = append(b.policy.Principals, Principal{
		Role:  role,
		Scope: "organization",
	})
	return b
}

// AllowTeamRole adds a team-scoped role principal to the policy.
func (b *PolicyBuilder) AllowTeamRole(role string) *PolicyBuilder {
	b.policy.Principals = append(b.policy.Principals, Principal{
		Role:  role,
		Scope: "team",
	})
	return b
}

// AllowAttribute adds an attribute-based principal to the policy.
func (b *PolicyBuilder) AllowAttribute(attribute string, operator ConditionOperator, value interface{}) *PolicyBuilder {
	b.policy.Principals = append(b.policy.Principals, Principal{
		Attribute: attribute,
		Operator:  operator,
		Value:     value,
	})
	return b
}

// OnActions adds actions to the policy.
func (b *PolicyBuilder) OnActions(actions ...string) *PolicyBuilder {
	b.policy.Actions = append(b.policy.Actions, actions...)
	return b
}

// OnResources adds resources to the policy.
func (b *PolicyBuilder) OnResources(resources ...string) *PolicyBuilder {
	b.policy.Resources = append(b.policy.Resources, resources...)
	return b
}

// OnAllResources allows access to all resources.
func (b *PolicyBuilder) OnAllResources() *PolicyBuilder {
	b.policy.Resources = []string{"*"}
	return b
}

// OnResourceType allows access to all resources of a specific type.
func (b *PolicyBuilder) OnResourceType(resourceType string) *PolicyBuilder {
	b.policy.Resources = append(b.policy.Resources, resourceType+":*")
	return b
}

// WithCondition adds a condition to the policy.
func (b *PolicyBuilder) WithCondition(attribute string, operator ConditionOperator, value interface{}) *PolicyBuilder {
	b.policy.Conditions = append(b.policy.Conditions, Condition{
		Attribute: attribute,
		Operator:  operator,
		Value:     value,
	})
	return b
}

// Disabled marks the policy as disabled.
func (b *PolicyBuilder) Disabled() *PolicyBuilder {
	b.policy.Enabled = false
	return b
}

// Build returns the constructed policy.
func (b *PolicyBuilder) Build() *Policy {
	return b.policy
}

// Built-in role helper functions

// NewOwnerPolicy creates a policy allowing owners full access to all resources.
//
// Owners have unrestricted access to all actions and resources within the organization.
func NewOwnerPolicy(organizationID, policyID string) *Policy {
	return NewPolicyBuilder(organizationID, "Owner Full Access").
		WithID(policyID).
		WithDescription("Owners have full control over all resources").
		AllowRole(string(RoleOwner)).
		OnActions("*").
		OnAllResources().
		Build()
}

// NewAdminPolicy creates a policy allowing admins to approve, create, update, and delete resources.
//
// Admins can perform most operations but may be restricted from certain owner-only actions.
func NewAdminPolicy(organizationID, policyID string) *Policy {
	return NewPolicyBuilder(organizationID, "Admin Policy").
		WithID(policyID).
		WithDescription("Admins can approve, create, update, and delete resources").
		AllowRole(string(RoleAdmin)).
		OnActions(
			"*:approve",
			"*:create",
			"*:update",
			"*:delete",
			"*:read",
		).
		OnAllResources().
		Build()
}

// NewMemberPolicy creates a policy allowing members to create and update resources.
//
// Members have standard access for creating and modifying resources they own.
func NewMemberPolicy(organizationID, policyID string) *Policy {
	return NewPolicyBuilder(organizationID, "Member Policy").
		WithID(policyID).
		WithDescription("Members can create and update resources").
		AllowRole(string(RoleMember)).
		OnActions(
			"*:create",
			"*:update",
			"*:read",
		).
		OnAllResources().
		Build()
}

// NewViewerPolicy creates a policy allowing viewers read-only access.
//
// Viewers can only read resources, with no modification permissions.
func NewViewerPolicy(organizationID, policyID string) *Policy {
	return NewPolicyBuilder(organizationID, "Viewer Policy").
		WithID(policyID).
		WithDescription("Viewers have read-only access").
		AllowRole(string(RoleViewer)).
		OnActions("*:read", "*:list").
		OnAllResources().
		Build()
}

// NewResourceSpecificPolicy creates a policy for a specific resource type and action.
//
// Useful for creating targeted policies like "admins can approve plans" or
// "members can create builds".
func NewResourceSpecificPolicy(organizationID, policyID, name, role, action, resourceType string) *Policy {
	return NewPolicyBuilder(organizationID, name).
		WithID(policyID).
		WithDescription(fmt.Sprintf("%s can %s on %s", role, action, resourceType)).
		AllowRole(role).
		OnActions(action).
		OnResourceType(resourceType).
		Build()
}

// NewConditionalPolicy creates a policy with attribute-based conditions.
//
// Example: Only allow plan approval if plan status is "pending"
func NewConditionalPolicy(organizationID, policyID, name, role string, actions []string, resourceType string, condition Condition) *Policy {
	return NewPolicyBuilder(organizationID, name).
		WithID(policyID).
		WithDescription(fmt.Sprintf("%s can perform actions with conditions", role)).
		AllowRole(role).
		OnActions(actions...).
		OnResourceType(resourceType).
		WithCondition(condition.Attribute, condition.Operator, condition.Value).
		Build()
}

// NewTeamPolicy creates a team-scoped policy.
//
// Team policies apply to users with specific team roles, providing
// finer-grained access control within an organization.
func NewTeamPolicy(organizationID, policyID, name, teamRole string, actions []string, resources []string) *Policy {
	return NewPolicyBuilder(organizationID, name).
		WithID(policyID).
		WithDescription(fmt.Sprintf("Team policy for %s role", teamRole)).
		AllowTeamRole(teamRole).
		OnActions(actions...).
		OnResources(resources...).
		Build()
}

// NewDenyPolicy creates a deny policy that explicitly blocks access.
//
// Deny policies always take precedence over allow policies, following
// AWS IAM-style evaluation (explicit deny wins).
func NewDenyPolicy(organizationID, policyID, name string, principals []Principal, actions []string, resources []string) *Policy {
	policy := NewPolicyBuilder(organizationID, name).
		WithID(policyID).
		WithDescription("Explicit deny policy").
		WithEffect(EffectDeny).
		OnActions(actions...).
		OnResources(resources...).
		Build()

	policy.Principals = principals
	return policy
}

// Standard action constants for common operations
const (
	ActionApprove = "approve"
	ActionCreate  = "create"
	ActionRead    = "read"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionList    = "list"
	ActionExecute = "execute"
)

// FormatAction formats an action with a resource type prefix.
//
// Example: FormatAction("plan", ActionApprove) => "plan:approve"
func FormatAction(resourceType, action string) string {
	return fmt.Sprintf("%s:%s", resourceType, action)
}

// FormatResource formats a resource identifier.
//
// Example: FormatResource("plan", "123") => "plan:123"
func FormatResource(resourceType, resourceID string) string {
	if resourceID == "*" || resourceID == "" {
		return fmt.Sprintf("%s:*", resourceType)
	}
	return fmt.Sprintf("%s:%s", resourceType, resourceID)
}
