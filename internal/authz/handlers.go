package authz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// PolicyHandlers provides HTTP handlers for policy CRUD operations.
type PolicyHandlers struct {
	policyStore PolicyStore
	engine      *Engine
}

// NewPolicyHandlers creates new policy HTTP handlers.
func NewPolicyHandlers(policyStore PolicyStore, engine *Engine) *PolicyHandlers {
	return &PolicyHandlers{
		policyStore: policyStore,
		engine:      engine,
	}
}

// RegisterRoutes registers policy management routes on the provided mux.
//
// Routes:
//   - POST   /api/policies          - Create a new policy
//   - GET    /api/policies/:id      - Get a specific policy
//   - PUT    /api/policies/:id      - Update a policy
//   - DELETE /api/policies/:id      - Delete a policy
//   - POST   /api/policies/simulate - Simulate policy evaluation (dry-run)
func (h *PolicyHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/policies", h.handlePolicies)
	mux.HandleFunc("/api/policies/", h.handlePolicy)
	mux.HandleFunc("/api/policies/simulate", h.handleSimulate)
}

// handlePolicies handles listing and creating policies.
func (h *PolicyHandlers) handlePolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createPolicy(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handlePolicy handles operations on a specific policy.
func (h *PolicyHandlers) handlePolicy(w http.ResponseWriter, r *http.Request) {
	// Extract policy ID from path
	policyID := strings.TrimPrefix(r.URL.Path, "/api/policies/")
	if policyID == "" || policyID == "simulate" {
		writeError(w, http.StatusBadRequest, "policy ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getPolicy(w, r, policyID)
	case http.MethodPut:
		h.updatePolicy(w, r, policyID)
	case http.MethodDelete:
		h.deletePolicy(w, r, policyID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// createPolicy creates a new policy.
//
// Request body:
//
//	{
//	  "name": "Admin Policy",
//	  "description": "Admins can approve plans",
//	  "effect": "allow",
//	  "principals": [{"role": "admin", "scope": "organization"}],
//	  "actions": ["plan:approve"],
//	  "resources": ["*"],
//	  "conditions": [],
//	  "enabled": true
//	}
//
// Response: 201 Created with policy JSON
func (h *PolicyHandlers) createPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated session
	session := GetSessionFromContext(ctx)
	if session == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Parse request
	var req struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Effect      Effect      `json:"effect"`
		Principals  []Principal `json:"principals"`
		Actions     []string    `json:"actions"`
		Resources   []string    `json:"resources"`
		Conditions  []Condition `json:"conditions"`
		Enabled     bool        `json:"enabled"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", decodeErr))
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Effect != EffectAllow && req.Effect != EffectDeny {
		writeError(w, http.StatusBadRequest, "effect must be 'allow' or 'deny'")
		return
	}
	if len(req.Actions) == 0 {
		writeError(w, http.StatusBadRequest, "actions are required")
		return
	}
	if len(req.Resources) == 0 {
		writeError(w, http.StatusBadRequest, "resources are required")
		return
	}

	// Generate ID (in production, use UUID or similar)
	policyID := fmt.Sprintf("policy-%s-%d", session.OrganizationID, generateID())

	// Create policy
	policy := &Policy{
		ID:             policyID,
		OrganizationID: session.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		Effect:         req.Effect,
		Principals:     req.Principals,
		Actions:        req.Actions,
		Resources:      req.Resources,
		Conditions:     req.Conditions,
		Enabled:        req.Enabled,
	}

	if err := h.policyStore.CreatePolicy(ctx, policy); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create policy: %v", err))
		return
	}

	// Return created policy
	writeJSON(w, http.StatusCreated, policy)
}

// getPolicy retrieves a specific policy.
func (h *PolicyHandlers) getPolicy(w http.ResponseWriter, r *http.Request, policyID string) {
	ctx := r.Context()

	// Get authenticated session
	session := GetSessionFromContext(ctx)
	if session == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Get policy
	policy, err := h.policyStore.GetPolicy(ctx, policyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	// Verify policy belongs to user's organization
	if policy.OrganizationID != session.OrganizationID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

// updatePolicy updates an existing policy.
//
//nolint:gocyclo // Policy update requires validating multiple optional fields
func (h *PolicyHandlers) updatePolicy(w http.ResponseWriter, r *http.Request, policyID string) {
	ctx := r.Context()

	// Get authenticated session
	session := GetSessionFromContext(ctx)
	if session == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Get existing policy
	existing, err := h.policyStore.GetPolicy(ctx, policyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	// Verify policy belongs to user's organization
	if existing.OrganizationID != session.OrganizationID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	// Parse update request
	var req struct {
		Name        *string     `json:"name,omitempty"`
		Description *string     `json:"description,omitempty"`
		Effect      *Effect     `json:"effect,omitempty"`
		Principals  []Principal `json:"principals,omitempty"`
		Actions     []string    `json:"actions,omitempty"`
		Resources   []string    `json:"resources,omitempty"`
		Conditions  []Condition `json:"conditions,omitempty"`
		Enabled     *bool       `json:"enabled,omitempty"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", decodeErr))
		return
	}

	// Apply updates
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Effect != nil {
		if *req.Effect != EffectAllow && *req.Effect != EffectDeny {
			writeError(w, http.StatusBadRequest, "effect must be 'allow' or 'deny'")
			return
		}
		existing.Effect = *req.Effect
	}
	if req.Principals != nil {
		existing.Principals = req.Principals
	}
	if req.Actions != nil {
		if len(req.Actions) == 0 {
			writeError(w, http.StatusBadRequest, "actions cannot be empty")
			return
		}
		existing.Actions = req.Actions
	}
	if req.Resources != nil {
		if len(req.Resources) == 0 {
			writeError(w, http.StatusBadRequest, "resources cannot be empty")
			return
		}
		existing.Resources = req.Resources
	}
	if req.Conditions != nil {
		existing.Conditions = req.Conditions
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	// Update policy
	if updateErr := h.policyStore.UpdatePolicy(ctx, existing); updateErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update policy: %v", updateErr))
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// deletePolicy deletes a policy.
func (h *PolicyHandlers) deletePolicy(w http.ResponseWriter, r *http.Request, policyID string) {
	ctx := r.Context()

	// Get authenticated session
	session := GetSessionFromContext(ctx)
	if session == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Get policy to verify ownership
	policy, err := h.policyStore.GetPolicy(ctx, policyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	// Verify policy belongs to user's organization
	if policy.OrganizationID != session.OrganizationID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	// Delete policy
	if deleteErr := h.policyStore.DeletePolicy(ctx, policyID); deleteErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete policy: %v", deleteErr))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSimulate simulates policy evaluation without making actual decisions.
//
// Request body:
//
//	{
//	  "subject": {
//	    "user_id": "user-123",
//	    "organization_id": "org-1",
//	    "organization_role": "admin"
//	  },
//	  "action": "plan:approve",
//	  "resource": {
//	    "type": "plan",
//	    "id": "plan-123"
//	  },
//	  "environment": {
//	    "client_ip": "192.168.1.1"
//	  }
//	}
//
// Response:
//
//	{
//	  "allowed": true,
//	  "reason": "access granted by policy",
//	  "policy_ids": ["policy-1"],
//	  "timestamp": "2024-01-15T10:00:00Z"
//	}
func (h *PolicyHandlers) handleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Get authenticated session for authorization
	session := GetSessionFromContext(ctx)
	if session == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Parse simulation request
	var req struct {
		Subject     *auth.Session          `json:"subject"`
		Action      string                 `json:"action"`
		Resource    Resource               `json:"resource"`
		Environment map[string]interface{} `json:"environment"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", decodeErr))
		return
	}

	// Validate request
	if req.Subject == nil {
		writeError(w, http.StatusBadRequest, "subject is required")
		return
	}
	if req.Action == "" {
		writeError(w, http.StatusBadRequest, "action is required")
		return
	}
	if req.Resource.Type == "" {
		writeError(w, http.StatusBadRequest, "resource type is required")
		return
	}

	// Simulate evaluation
	decision, err := h.engine.Evaluate(ctx, &AuthorizationRequest{
		Subject:     req.Subject,
		Action:      req.Action,
		Resource:    req.Resource,
		Environment: req.Environment,
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("simulation failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, decision)
}

// HTTP helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck,gosec // Response headers already sent
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

// generateID generates a simple numeric ID (in production, use UUID).
var idCounter int64

func generateID() int64 {
	idCounter++
	return idCounter
}
