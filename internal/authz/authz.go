package authz

import (
	"context"
	"time"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// Effect represents the effect of a policy decision (allow or deny).
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// ConditionOperator represents operators for policy conditions.
type ConditionOperator string

const (
	OperatorEquals              ConditionOperator = "equals"
	OperatorNotEquals           ConditionOperator = "not_equals"
	OperatorIn                  ConditionOperator = "in"
	OperatorNotIn               ConditionOperator = "not_in"
	OperatorGreaterThan         ConditionOperator = "greater_than"
	OperatorLessThan            ConditionOperator = "less_than"
	OperatorGreaterThanOrEquals ConditionOperator = "greater_than_or_equals"
	OperatorLessThanOrEquals    ConditionOperator = "less_than_or_equals"
	OperatorStringLike          ConditionOperator = "string_like"
	OperatorExists              ConditionOperator = "exists"
	OperatorNotExists           ConditionOperator = "not_exists"
)

// Role represents standard built-in roles (RBAC abstraction over ABAC).
type Role string

const (
	RoleOwner  Role = "owner"  // Full control, can manage members
	RoleAdmin  Role = "admin"  // Can approve, create, update, delete
	RoleMember Role = "member" // Can create, update (not approve or delete)
	RoleViewer Role = "viewer" // Read-only access
)

// Policy represents an ABAC authorization policy.
type Policy struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	Version        int            `json:"version"`
	Effect         Effect         `json:"effect"` // allow or deny
	Principals     []Principal    `json:"principals"`
	Actions        []string       `json:"actions"`     // e.g., ["plan:approve", "build:run"]
	Resources      []string       `json:"resources"`   // e.g., ["plan:*", "build:123"]
	Conditions     []Condition    `json:"conditions"`  // Optional conditions
	Enabled        bool           `json:"enabled"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Principal identifies who the policy applies to.
type Principal struct {
	// For role-based principals
	Role  string `json:"role,omitempty"`  // e.g., "admin", "member"
	Scope string `json:"scope,omitempty"` // e.g., "organization", "team"

	// For attribute-based principals
	Attribute string      `json:"attribute,omitempty"` // e.g., "subject.department"
	Operator  ConditionOperator `json:"operator,omitempty"`
	Value     interface{} `json:"value,omitempty"`
}

// Condition represents a policy condition that must be satisfied.
type Condition struct {
	Attribute string            `json:"attribute"` // e.g., "resource.status", "subject.role"
	Operator  ConditionOperator `json:"operator"`
	Value     interface{}       `json:"value"` // Can be string, number, array, or attribute reference ($subject.tenant_id)
}

// Resource represents a resource being accessed.
type Resource struct {
	Type string `json:"type"` // e.g., "plan", "build", "policy"
	ID   string `json:"id"`   // e.g., "plan-123", "build-456"
}

// Attributes represents a map of attribute key-value pairs.
type Attributes map[string]interface{}

// AuthorizationRequest represents a request to evaluate authorization.
type AuthorizationRequest struct {
	Subject     *auth.Session          // Authenticated user session
	Action      string                 // Action being performed (e.g., "plan:approve")
	Resource    Resource               // Resource being accessed
	Environment map[string]interface{} // Contextual attributes (IP, time, etc.)
}

// Decision represents the result of an authorization evaluation.
type Decision struct {
	Allowed   bool     `json:"allowed"`            // Whether access is granted
	Reason    string   `json:"reason"`             // Human-readable explanation
	PolicyIDs []string `json:"policy_ids"`         // Policies that contributed to decision
	Timestamp time.Time `json:"timestamp"`
}

// PolicyStore manages authorization policies.
type PolicyStore interface {
	// LoadPolicies loads all enabled policies for a tenant.
	LoadPolicies(ctx context.Context, organizationID string) ([]*Policy, error)

	// CreatePolicy creates a new policy.
	CreatePolicy(ctx context.Context, policy *Policy) error

	// UpdatePolicy updates an existing policy.
	UpdatePolicy(ctx context.Context, policy *Policy) error

	// DeletePolicy deletes a policy.
	DeletePolicy(ctx context.Context, policyID string) error

	// GetPolicy retrieves a specific policy.
	GetPolicy(ctx context.Context, policyID string) (*Policy, error)
}

// AttributeResolver resolves attributes for authorization decisions.
type AttributeResolver interface {
	// GetSubjectAttributes extracts attributes from the authenticated session.
	GetSubjectAttributes(ctx context.Context, subject *auth.Session) (Attributes, error)

	// GetResourceAttributes fetches attributes for a specific resource.
	GetResourceAttributes(ctx context.Context, resourceType, resourceID string) (Attributes, error)
}

// Evaluator evaluates authorization requests against policies.
type Evaluator interface {
	// Evaluate determines if the request should be allowed or denied.
	Evaluate(ctx context.Context, req *AuthorizationRequest) (*Decision, error)
}

// Engine is the main authorization engine that coordinates policy evaluation.
type Engine struct {
	policyStore  PolicyStore
	attrResolver AttributeResolver
}

// NewEngine creates a new authorization engine.
func NewEngine(policyStore PolicyStore, attrResolver AttributeResolver) *Engine {
	return &Engine{
		policyStore:  policyStore,
		attrResolver: attrResolver,
	}
}

// Evaluate evaluates an authorization request.
//
// Algorithm (AWS IAM-style):
// 1. Default decision: DENY
// 2. Load all policies for the organization
// 3. Filter policies that match principal, action, and resource
// 4. If any policy has effect: deny → DENY (explicit deny wins)
// 5. If any policy has effect: allow and all conditions pass → ALLOW
// 6. Otherwise → DENY
func (e *Engine) Evaluate(ctx context.Context, req *AuthorizationRequest) (*Decision, error) {
	// Default deny
	decision := &Decision{
		Allowed:   false,
		Reason:    "no matching policy found (default deny)",
		PolicyIDs: []string{},
		Timestamp: time.Now(),
	}

	// Get organization ID from session
	if req.Subject == nil {
		decision.Reason = "no authenticated subject"
		return decision, nil
	}

	organizationID := req.Subject.OrganizationID
	if organizationID == "" {
		decision.Reason = "subject not associated with organization"
		return decision, nil
	}

	// Load policies for organization
	policies, err := e.policyStore.LoadPolicies(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	// Get subject and resource attributes
	subjectAttrs, err := e.attrResolver.GetSubjectAttributes(ctx, req.Subject)
	if err != nil {
		return nil, err
	}

	var resourceAttrs Attributes
	if req.Resource.ID != "" {
		resourceAttrs, err = e.attrResolver.GetResourceAttributes(ctx, req.Resource.Type, req.Resource.ID)
		if err != nil {
			return nil, err
		}
	} else {
		resourceAttrs = make(Attributes)
	}

	// Evaluate policies
	var denyPolicies []string
	var allowPolicies []string

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		// Check if policy matches
		if !e.policyMatches(policy, req, subjectAttrs, resourceAttrs) {
			continue
		}

		// Evaluate conditions
		conditionsPassed, err := e.evaluateConditions(policy.Conditions, subjectAttrs, resourceAttrs, req.Environment)
		if err != nil {
			return nil, err
		}

		if !conditionsPassed {
			continue
		}

		// Track matching policy
		if policy.Effect == EffectDeny {
			denyPolicies = append(denyPolicies, policy.ID)
		} else if policy.Effect == EffectAllow {
			allowPolicies = append(allowPolicies, policy.ID)
		}
	}

	// Explicit deny wins
	if len(denyPolicies) > 0 {
		decision.Allowed = false
		decision.Reason = "access explicitly denied by policy"
		decision.PolicyIDs = denyPolicies
		return decision, nil
	}

	// Allow if at least one allow policy matched
	if len(allowPolicies) > 0 {
		decision.Allowed = true
		decision.Reason = "access granted by policy"
		decision.PolicyIDs = allowPolicies
		return decision, nil
	}

	// Default deny
	return decision, nil
}

// policyMatches checks if a policy matches the request.
func (e *Engine) policyMatches(policy *Policy, req *AuthorizationRequest, subjectAttrs, resourceAttrs Attributes) bool {
	// Check principal match
	if !e.principalMatches(policy.Principals, subjectAttrs) {
		return false
	}

	// Check action match
	if !e.actionMatches(policy.Actions, req.Action) {
		return false
	}

	// Check resource match
	if !e.resourceMatches(policy.Resources, req.Resource) {
		return false
	}

	return true
}

// principalMatches checks if any principal in the policy matches the subject.
func (e *Engine) principalMatches(principals []Principal, subjectAttrs Attributes) bool {
	if len(principals) == 0 {
		// Empty principals means match all
		return true
	}

	for _, principal := range principals {
		if principal.Role != "" {
			// Role-based principal
			if role, ok := subjectAttrs["role"].(string); ok && role == principal.Role {
				return true
			}
			if orgRole, ok := subjectAttrs["organization_role"].(string); ok && orgRole == principal.Role {
				return true
			}
		} else if principal.Attribute != "" {
			// Attribute-based principal
			attrValue := e.resolveAttribute(principal.Attribute, subjectAttrs, nil, nil)
			if e.evaluateOperator(principal.Operator, attrValue, principal.Value) {
				return true
			}
		}
	}

	return false
}

// actionMatches checks if the policy action matches the request action.
func (e *Engine) actionMatches(policyActions []string, requestAction string) bool {
	for _, action := range policyActions {
		if action == "*" || action == requestAction {
			return true
		}
		// Support wildcards like "plan:*"
		if len(action) > 0 && action[len(action)-1] == '*' {
			prefix := action[:len(action)-1]
			if len(requestAction) >= len(prefix) && requestAction[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// resourceMatches checks if the policy resource matches the request resource.
func (e *Engine) resourceMatches(policyResources []string, requestResource Resource) bool {
	resourcePattern := requestResource.Type + ":" + requestResource.ID

	for _, resource := range policyResources {
		if resource == "*" || resource == resourcePattern {
			return true
		}
		// Support wildcards like "plan:*"
		if len(resource) > 0 && resource[len(resource)-1] == '*' {
			prefix := resource[:len(resource)-1]
			if len(resourcePattern) >= len(prefix) && resourcePattern[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// evaluateConditions evaluates all conditions in a policy.
func (e *Engine) evaluateConditions(conditions []Condition, subjectAttrs, resourceAttrs Attributes, env map[string]interface{}) (bool, error) {
	// No conditions means always pass
	if len(conditions) == 0 {
		return true, nil
	}

	// All conditions must pass (AND logic)
	for _, condition := range conditions {
		attrValue := e.resolveAttribute(condition.Attribute, subjectAttrs, resourceAttrs, env)

		// Resolve value if it's an attribute reference (e.g., $subject.organization_id)
		conditionValue := condition.Value
		if strValue, ok := conditionValue.(string); ok && len(strValue) > 0 && strValue[0] == '$' {
			conditionValue = e.resolveAttribute(strValue, subjectAttrs, resourceAttrs, env)
		}

		if !e.evaluateOperator(condition.Operator, attrValue, conditionValue) {
			return false, nil
		}
	}

	return true, nil
}

// resolveAttribute resolves an attribute value from subject, resource, or environment.
func (e *Engine) resolveAttribute(attrPath string, subjectAttrs, resourceAttrs Attributes, env map[string]interface{}) interface{} {
	// Handle attribute references like $subject.tenant_id, $resource.status
	if len(attrPath) > 0 && attrPath[0] == '$' {
		// Parse attribute path
		if len(attrPath) > 9 && attrPath[:9] == "$subject." {
			key := attrPath[9:]
			if val, ok := subjectAttrs[key]; ok {
				return val
			}
		} else if len(attrPath) > 10 && attrPath[:10] == "$resource." {
			key := attrPath[10:]
			if val, ok := resourceAttrs[key]; ok {
				return val
			}
		} else if len(attrPath) > 13 && attrPath[:13] == "$environment." {
			key := attrPath[13:]
			if val, ok := env[key]; ok {
				return val
			}
		}
		return nil
	}

	// Direct attribute lookup (fallback)
	if val, ok := resourceAttrs[attrPath]; ok {
		return val
	}
	if val, ok := subjectAttrs[attrPath]; ok {
		return val
	}
	if val, ok := env[attrPath]; ok {
		return val
	}

	return nil
}

// evaluateOperator evaluates a condition operator.
func (e *Engine) evaluateOperator(op ConditionOperator, left, right interface{}) bool {
	switch op {
	case OperatorEquals:
		return left == right

	case OperatorNotEquals:
		return left != right

	case OperatorIn:
		if rightArray, ok := right.([]interface{}); ok {
			for _, item := range rightArray {
				if left == item {
					return true
				}
			}
		}
		return false

	case OperatorNotIn:
		if rightArray, ok := right.([]interface{}); ok {
			for _, item := range rightArray {
				if left == item {
					return false
				}
			}
			return true
		}
		return false

	case OperatorGreaterThan:
		return compareNumbers(left, right) > 0

	case OperatorLessThan:
		return compareNumbers(left, right) < 0

	case OperatorGreaterThanOrEquals:
		return compareNumbers(left, right) >= 0

	case OperatorLessThanOrEquals:
		return compareNumbers(left, right) <= 0

	case OperatorStringLike:
		leftStr, lok := left.(string)
		rightStr, rok := right.(string)
		if lok && rok {
			return matchGlob(leftStr, rightStr)
		}
		return false

	case OperatorExists:
		return left != nil

	case OperatorNotExists:
		return left == nil

	default:
		return false
	}
}

// compareNumbers compares two values as numbers.
func compareNumbers(left, right interface{}) int {
	leftNum, lok := toFloat64(left)
	rightNum, rok := toFloat64(right)
	if !lok || !rok {
		return 0
	}
	if leftNum < rightNum {
		return -1
	} else if leftNum > rightNum {
		return 1
	}
	return 0
}

// toFloat64 converts a value to float64 if possible.
func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	default:
		return 0, false
	}
}

// matchGlob performs simple glob pattern matching (* wildcard).
func matchGlob(text, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == "" {
		return text == ""
	}

	// Simple implementation - find * and match parts
	if pattern[0] == '*' {
		// Prefix wildcard
		suffix := pattern[1:]
		return len(text) >= len(suffix) && text[len(text)-len(suffix):] == suffix
	} else if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		// Suffix wildcard
		prefix := pattern[:len(pattern)-1]
		return len(text) >= len(prefix) && text[:len(prefix)] == prefix
	}

	// No wildcard - exact match
	return text == pattern
}
