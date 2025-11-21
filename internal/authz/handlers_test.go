package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// TestPolicyHandlers_CreatePolicy tests creating a new policy.
func TestPolicyHandlers_CreatePolicy(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create test session
	session := &auth.Session{
		UserID:           "user-1",
		Email:            "admin@example.com",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	// Prepare request
	reqBody := map[string]interface{}{
		"name":        "Test Policy",
		"description": "A test policy",
		"effect":      "allow",
		"principals": []map[string]interface{}{
			{"role": "admin", "scope": "organization"},
		},
		"actions":   []string{"plan:approve"},
		"resources": []string{"plan:*"},
		"conditions": []map[string]interface{}{
			{"attribute": "$resource.status", "operator": "equals", "value": "pending"},
		},
		"enabled": true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/policies", bytes.NewReader(body))
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.createPolicy(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var response Policy
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Name != "Test Policy" {
		t.Errorf("expected name 'Test Policy', got %s", response.Name)
	}
	if response.OrganizationID != "org-1" {
		t.Errorf("expected organization_id 'org-1', got %s", response.OrganizationID)
	}
	if response.Effect != EffectAllow {
		t.Errorf("expected effect 'allow', got %s", response.Effect)
	}
	if !response.Enabled {
		t.Error("expected policy to be enabled")
	}

	// Verify policy was stored
	policies, err := store.LoadPolicies(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("failed to load policies: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
}

// TestPolicyHandlers_CreatePolicy_NoSession tests creating a policy without authentication.
func TestPolicyHandlers_CreatePolicy_NoSession(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	reqBody := map[string]interface{}{
		"name":      "Test Policy",
		"effect":    "allow",
		"actions":   []string{"plan:read"},
		"resources": []string{"*"},
		"enabled":   true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/policies", bytes.NewReader(body))
	// No session in context

	w := httptest.NewRecorder()
	handlers.createPolicy(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestPolicyHandlers_CreatePolicy_ValidationErrors tests validation errors.
func TestPolicyHandlers_CreatePolicy_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		reqBody    map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "missing name",
			reqBody: map[string]interface{}{
				"effect":    "allow",
				"actions":   []string{"plan:read"},
				"resources": []string{"*"},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "name is required",
		},
		{
			name: "invalid effect",
			reqBody: map[string]interface{}{
				"name":      "Test",
				"effect":    "invalid",
				"actions":   []string{"plan:read"},
				"resources": []string{"*"},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "effect must be 'allow' or 'deny'",
		},
		{
			name: "missing actions",
			reqBody: map[string]interface{}{
				"name":      "Test",
				"effect":    "allow",
				"resources": []string{"*"},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "actions are required",
		},
		{
			name: "missing resources",
			reqBody: map[string]interface{}{
				"name":    "Test",
				"effect":  "allow",
				"actions": []string{"plan:read"},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "resources are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewInMemoryPolicyStore()
			resourceStore := NewInMemoryResourceStore()
			resolver := NewDefaultAttributeResolver(resourceStore)
			engine := NewEngine(store, resolver)
			handlers := NewPolicyHandlers(store, engine)

			session := &auth.Session{
				UserID:           "user-1",
				OrganizationID:   "org-1",
				OrganizationRole: "admin",
			}

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/policies", bytes.NewReader(body))
			req = req.WithContext(SetSessionInContext(context.Background(), session))

			w := httptest.NewRecorder()
			handlers.createPolicy(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var response map[string]string
			json.Unmarshal(w.Body.Bytes(), &response)
			if !strings.Contains(response["error"], tt.wantError) {
				t.Errorf("expected error to contain '%s', got '%s'", tt.wantError, response["error"])
			}
		})
	}
}

// TestPolicyHandlers_GetPolicy tests retrieving a policy.
func TestPolicyHandlers_GetPolicy(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create a test policy
	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Actions:        []string{"plan:read"},
		Resources:      []string{"*"},
		Enabled:        true,
	}
	store.CreatePolicy(context.Background(), policy)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/policies/policy-1", nil)
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.getPolicy(w, req, "policy-1")

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Policy
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.ID != "policy-1" {
		t.Errorf("expected ID 'policy-1', got %s", response.ID)
	}
	if response.Name != "Test Policy" {
		t.Errorf("expected name 'Test Policy', got %s", response.Name)
	}
}

// TestPolicyHandlers_GetPolicy_NotFound tests retrieving a non-existent policy.
func TestPolicyHandlers_GetPolicy_NotFound(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/policies/nonexistent", nil)
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.getPolicy(w, req, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestPolicyHandlers_GetPolicy_Forbidden tests accessing another organization's policy.
func TestPolicyHandlers_GetPolicy_Forbidden(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create a policy for org-1
	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Actions:        []string{"plan:read"},
		Resources:      []string{"*"},
		Enabled:        true,
	}
	store.CreatePolicy(context.Background(), policy)

	// Try to access with org-2 session
	session := &auth.Session{
		UserID:           "user-2",
		OrganizationID:   "org-2",
		OrganizationRole: "admin",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/policies/policy-1", nil)
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.getPolicy(w, req, "policy-1")

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

// TestPolicyHandlers_UpdatePolicy tests updating a policy.
func TestPolicyHandlers_UpdatePolicy(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create a test policy
	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Original Name",
		Description:    "Original Description",
		Effect:         EffectAllow,
		Actions:        []string{"plan:read"},
		Resources:      []string{"*"},
		Enabled:        true,
	}
	store.CreatePolicy(context.Background(), policy)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	// Update name and description
	reqBody := map[string]interface{}{
		"name":        "Updated Name",
		"description": "Updated Description",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/policies/policy-1", bytes.NewReader(body))
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.updatePolicy(w, req, "policy-1")

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Policy
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", response.Name)
	}
	if response.Description != "Updated Description" {
		t.Errorf("expected description 'Updated Description', got %s", response.Description)
	}

	// Verify other fields unchanged
	if response.Effect != EffectAllow {
		t.Error("effect should not have changed")
	}
	if len(response.Actions) != 1 || response.Actions[0] != "plan:read" {
		t.Error("actions should not have changed")
	}
}

// TestPolicyHandlers_UpdatePolicy_PartialUpdate tests partial policy updates.
func TestPolicyHandlers_UpdatePolicy_PartialUpdate(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create a test policy
	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Original",
		Effect:         EffectAllow,
		Actions:        []string{"plan:read"},
		Resources:      []string{"*"},
		Enabled:        true,
	}
	store.CreatePolicy(context.Background(), policy)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	// Update only enabled flag
	enabled := false
	reqBody := map[string]interface{}{
		"enabled": enabled,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/policies/policy-1", bytes.NewReader(body))
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.updatePolicy(w, req, "policy-1")

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Policy
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Enabled {
		t.Error("expected policy to be disabled")
	}
	if response.Name != "Original" {
		t.Error("name should not have changed")
	}
}

// TestPolicyHandlers_UpdatePolicy_Validation tests validation during updates.
func TestPolicyHandlers_UpdatePolicy_Validation(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Test",
		Effect:         EffectAllow,
		Actions:        []string{"plan:read"},
		Resources:      []string{"*"},
		Enabled:        true,
	}
	store.CreatePolicy(context.Background(), policy)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	tests := []struct {
		name       string
		reqBody    map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "invalid effect",
			reqBody: map[string]interface{}{
				"effect": "invalid",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "effect must be 'allow' or 'deny'",
		},
		{
			name: "empty actions",
			reqBody: map[string]interface{}{
				"actions": []string{},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "actions cannot be empty",
		},
		{
			name: "empty resources",
			reqBody: map[string]interface{}{
				"resources": []string{},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "resources cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPut, "/api/policies/policy-1", bytes.NewReader(body))
			req = req.WithContext(SetSessionInContext(context.Background(), session))

			w := httptest.NewRecorder()
			handlers.updatePolicy(w, req, "policy-1")

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var response map[string]string
			json.Unmarshal(w.Body.Bytes(), &response)
			if !strings.Contains(response["error"], tt.wantError) {
				t.Errorf("expected error to contain '%s', got '%s'", tt.wantError, response["error"])
			}
		})
	}
}

// TestPolicyHandlers_DeletePolicy tests deleting a policy.
func TestPolicyHandlers_DeletePolicy(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create a test policy
	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Actions:        []string{"plan:read"},
		Resources:      []string{"*"},
		Enabled:        true,
	}
	store.CreatePolicy(context.Background(), policy)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/policies/policy-1", nil)
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.deletePolicy(w, req, "policy-1")

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify policy was deleted
	_, err := store.GetPolicy(context.Background(), "policy-1")
	if err == nil {
		t.Error("expected policy to be deleted")
	}
}

// TestPolicyHandlers_DeletePolicy_NotFound tests deleting a non-existent policy.
func TestPolicyHandlers_DeletePolicy_NotFound(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/policies/nonexistent", nil)
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.deletePolicy(w, req, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestPolicyHandlers_DeletePolicy_Forbidden tests deleting another organization's policy.
func TestPolicyHandlers_DeletePolicy_Forbidden(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create a policy for org-1
	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Actions:        []string{"plan:read"},
		Resources:      []string{"*"},
		Enabled:        true,
	}
	store.CreatePolicy(context.Background(), policy)

	// Try to delete with org-2 session
	session := &auth.Session{
		UserID:           "user-2",
		OrganizationID:   "org-2",
		OrganizationRole: "admin",
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/policies/policy-1", nil)
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.deletePolicy(w, req, "policy-1")

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}

	// Verify policy still exists
	_, err := store.GetPolicy(context.Background(), "policy-1")
	if err != nil {
		t.Error("policy should not have been deleted")
	}
}

// TestPolicyHandlers_Simulate tests policy simulation endpoint.
func TestPolicyHandlers_Simulate(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create a test policy
	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Admin Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "admin", Scope: "organization"},
		},
		Actions:   []string{"plan:approve"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	store.CreatePolicy(context.Background(), policy)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	// Simulate authorization request
	reqBody := map[string]interface{}{
		"subject": map[string]interface{}{
			"UserID":           "test-user",
			"OrganizationID":   "org-1",
			"OrganizationRole": "admin",
		},
		"action": "plan:approve",
		"resource": map[string]interface{}{
			"type": "plan",
			"id":   "plan-123",
		},
		"environment": map[string]interface{}{
			"client_ip": "192.168.1.1",
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/policies/simulate", bytes.NewReader(body))
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.handleSimulate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Decision
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Allowed {
		t.Error("expected access to be allowed")
	}
	if len(response.PolicyIDs) != 1 || response.PolicyIDs[0] != "policy-1" {
		t.Errorf("expected policy-1 to match, got %v", response.PolicyIDs)
	}
}

// TestPolicyHandlers_Simulate_Denied tests denied simulation.
func TestPolicyHandlers_Simulate_Denied(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	// Create a policy that won't match
	policy := &Policy{
		ID:             "policy-1",
		OrganizationID: "org-1",
		Name:           "Admin Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "admin", Scope: "organization"},
		},
		Actions:   []string{"plan:approve"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	store.CreatePolicy(context.Background(), policy)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	// Simulate with member role (should be denied)
	reqBody := map[string]interface{}{
		"subject": map[string]interface{}{
			"UserID":           "test-user",
			"OrganizationID":   "org-1",
			"OrganizationRole": "member", // Not admin
		},
		"action": "plan:approve",
		"resource": map[string]interface{}{
			"type": "plan",
			"id":   "plan-123",
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/policies/simulate", bytes.NewReader(body))
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	handlers.handleSimulate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Decision
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Allowed {
		t.Error("expected access to be denied")
	}
}

// TestPolicyHandlers_Simulate_Validation tests simulation validation.
func TestPolicyHandlers_Simulate_Validation(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	tests := []struct {
		name       string
		reqBody    map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "missing subject",
			reqBody: map[string]interface{}{
				"action": "plan:read",
				"resource": map[string]interface{}{
					"type": "plan",
				},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "subject is required",
		},
		{
			name: "missing action",
			reqBody: map[string]interface{}{
				"subject": map[string]interface{}{
					"UserID":         "test-user",
					"OrganizationID": "org-1",
				},
				"resource": map[string]interface{}{
					"type": "plan",
				},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "action is required",
		},
		{
			name: "missing resource type",
			reqBody: map[string]interface{}{
				"subject": map[string]interface{}{
					"UserID":         "test-user",
					"OrganizationID": "org-1",
				},
				"action": "plan:read",
				"resource": map[string]interface{}{
					"id": "123",
				},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "resource type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/policies/simulate", bytes.NewReader(body))
			req = req.WithContext(SetSessionInContext(context.Background(), session))

			w := httptest.NewRecorder()
			handlers.handleSimulate(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var response map[string]string
			json.Unmarshal(w.Body.Bytes(), &response)
			if !strings.Contains(response["error"], tt.wantError) {
				t.Errorf("expected error to contain '%s', got '%s'", tt.wantError, response["error"])
			}
		})
	}
}

// TestPolicyHandlers_RegisterRoutes tests route registration.
func TestPolicyHandlers_RegisterRoutes(t *testing.T) {
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)
	handlers := NewPolicyHandlers(store, engine)

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	// Test that routes are registered by making requests
	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	// Test POST /api/policies
	reqBody := map[string]interface{}{
		"name":      "Test",
		"effect":    "allow",
		"actions":   []string{"plan:read"},
		"resources": []string{"*"},
		"enabled":   true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/policies", bytes.NewReader(body))
	req = req.WithContext(SetSessionInContext(context.Background(), session))

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("POST /api/policies failed with status %d", w.Code)
	}
}
