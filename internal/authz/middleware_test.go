package authz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/specular/internal/auth"
)

func TestRequirePermission_Success(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)

	// Create admin policy
	policy := &Policy{
		ID:             "test-policy",
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
	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	// Create admin session
	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	// Setup middleware
	cfg := MiddlewareConfig{
		Engine:              engine,
		ResourceIDExtractor: URLParamExtractor("planID"),
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	middleware := RequirePermission("plan:approve", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request with session in context
	req := httptest.NewRequest("POST", "/api/plans/123/approve", nil)
	req.SetPathValue("planID", "123")
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled, "handler should have been called")
	assert.Equal(t, "success", rr.Body.String())
}

func TestRequirePermission_NoSession(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	cfg := MiddlewareConfig{
		Engine: engine,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	middleware := RequirePermission("plan:approve", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request without session
	req := httptest.NewRequest("POST", "/api/plans/123/approve", nil)
	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var errResp ErrorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&errResp))
	assert.Equal(t, "unauthorized", errResp.Error)
	assert.Contains(t, errResp.Message, "no authenticated session")
}

func TestRequirePermission_AccessDenied(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)

	// Create viewer policy (read-only)
	policy := &Policy{
		ID:             "viewer-policy",
		OrganizationID: "org-1",
		Name:           "Viewer Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "viewer", Scope: "organization"},
		},
		Actions:   []string{"*:read"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	// Create viewer session
	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "viewer",
	}

	cfg := MiddlewareConfig{
		Engine: engine,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	middleware := RequirePermission("plan:approve", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request
	req := httptest.NewRequest("POST", "/api/plans/123/approve", nil)
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, rr.Code)

	var errResp ErrorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&errResp))
	assert.Equal(t, "forbidden", errResp.Error)
	assert.Contains(t, errResp.Message, "no matching policy found")
}

func TestRequirePermission_URLParamExtractor(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)

	// Set resource attributes
	resourceStore.SetResourceAttributes("plan", "plan-123", Attributes{
		"organization_id": "org-1",
	})

	// Create policy with condition
	policy := &Policy{
		ID:             "test-policy",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "member", Scope: "organization"},
		},
		Actions:   []string{"plan:read"},
		Resources: []string{"plan:*"},
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
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "member",
	}

	cfg := MiddlewareConfig{
		Engine:              engine,
		ResourceIDExtractor: URLParamExtractor("planID"),
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission("plan:read", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request
	req := httptest.NewRequest("GET", "/api/plans/plan-123", nil)
	req.SetPathValue("planID", "plan-123")
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestRequirePermission_HeaderExtractor(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	policy := &Policy{
		ID:             "test-policy",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "member", Scope: "organization"},
		},
		Actions:   []string{"plan:read"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "member",
	}

	cfg := MiddlewareConfig{
		Engine:              engine,
		ResourceIDExtractor: HeaderExtractor("X-Resource-ID"),
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission("plan:read", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request
	req := httptest.NewRequest("GET", "/api/plans", nil)
	req.Header.Set("X-Resource-ID", "plan-123")
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestRequirePermission_QueryParamExtractor(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	policy := &Policy{
		ID:             "test-policy",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "member", Scope: "organization"},
		},
		Actions:   []string{"plan:read"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "member",
	}

	cfg := MiddlewareConfig{
		Engine:              engine,
		ResourceIDExtractor: QueryParamExtractor("plan_id"),
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission("plan:read", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request
	req := httptest.NewRequest("GET", "/api/plans?plan_id=plan-123", nil)
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestRequirePermission_NoResourceIDExtractor(t *testing.T) {
	// Test type-level authorization (no specific resource ID)
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	policy := &Policy{
		ID:             "test-policy",
		OrganizationID: "org-1",
		Name:           "Test Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "member", Scope: "organization"},
		},
		Actions:   []string{"plan:list"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "member",
	}

	cfg := MiddlewareConfig{
		Engine: engine,
		// No ResourceIDExtractor - type-level authorization
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission("plan:list", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request
	req := httptest.NewRequest("GET", "/api/plans", nil)
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestRequirePermission_CustomUnauthorizedHandler(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	customUnauthorizedCalled := false
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customUnauthorizedCalled = true
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("custom unauthorized"))
	})

	cfg := MiddlewareConfig{
		Engine:              engine,
		UnauthorizedHandler: customHandler,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	middleware := RequirePermission("plan:approve", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request without session
	req := httptest.NewRequest("POST", "/api/plans/123/approve", nil)
	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.True(t, customUnauthorizedCalled)
	assert.Equal(t, "custom unauthorized", rr.Body.String())
}

func TestRequirePermission_CustomForbiddenHandler(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resolver := NewDefaultAttributeResolver(NewInMemoryResourceStore())
	engine := NewEngine(store, resolver)

	// No policies - default deny

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "member",
	}

	customForbiddenCalled := false
	var capturedDecision *Decision

	cfg := MiddlewareConfig{
		Engine: engine,
		ForbiddenHandler: func(w http.ResponseWriter, r *http.Request, decision *Decision) {
			customForbiddenCalled = true
			capturedDecision = decision
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("custom forbidden"))
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	middleware := RequirePermission("plan:approve", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request
	req := httptest.NewRequest("POST", "/api/plans/123/approve", nil)
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.True(t, customForbiddenCalled)
	assert.Equal(t, "custom forbidden", rr.Body.String())
	assert.NotNil(t, capturedDecision)
	assert.False(t, capturedDecision.Allowed)
}

func TestRequirePermission_PanicOnNilEngine(t *testing.T) {
	cfg := MiddlewareConfig{
		Engine: nil, // This should panic
	}

	assert.Panics(t, func() {
		RequirePermission("plan:approve", "plan", cfg)
	})
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		setupReq   func() *http.Request
		expectedIP string
	}{
		{
			name: "X-Forwarded-For with single IP",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "192.168.1.100")
				return req
			},
			expectedIP: "192.168.1.100",
		},
		{
			name: "X-Forwarded-For with multiple IPs",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1, 172.16.0.1")
				return req
			},
			expectedIP: "192.168.1.100",
		},
		{
			name: "X-Real-IP",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "192.168.1.200")
				return req
			},
			expectedIP: "192.168.1.200",
		},
		{
			name: "RemoteAddr",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.168.1.50:12345"
				return req
			},
			expectedIP: "192.168.1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			ip := getClientIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}

func TestSetAndGetSessionFromContext(t *testing.T) {
	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	// Set session in context
	ctx := context.Background()
	ctx = SetSessionInContext(ctx, session)

	// Get session from context
	retrieved := GetSessionFromContext(ctx)

	assert.NotNil(t, retrieved)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.OrganizationID, retrieved.OrganizationID)
	assert.Equal(t, session.OrganizationRole, retrieved.OrganizationRole)
}

func TestGetSessionFromContext_NoSession(t *testing.T) {
	ctx := context.Background()
	session := GetSessionFromContext(ctx)
	assert.Nil(t, session)
}

func TestExtractEnvironmentAttributes(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/plans/123/approve", nil)
	req.Header.Set("User-Agent", "test-agent/1.0")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.RemoteAddr = "10.0.0.1:12345"

	env := extractEnvironmentAttributes(req)

	assert.Equal(t, "192.168.1.100", env["client_ip"])
	assert.Equal(t, "POST", env["method"])
	assert.Equal(t, "/api/plans/123/approve", env["path"])
	assert.Equal(t, "test-agent/1.0", env["user_agent"])
}

// Integration test: Full authorization flow with resource attributes and conditions
func TestRequirePermission_IntegrationWithConditions(t *testing.T) {
	// Setup
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)

	// Set resource attributes
	resourceStore.SetResourceAttributes("plan", "plan-123", Attributes{
		"organization_id": "org-1",
		"status":          "pending",
	})

	// Create policy with multiple conditions
	policy := &Policy{
		ID:             "test-policy",
		OrganizationID: "org-1",
		Name:           "Conditional Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "admin", Scope: "organization"},
		},
		Actions:   []string{"plan:approve"},
		Resources: []string{"plan:*"},
		Conditions: []Condition{
			{
				Attribute: "$resource.organization_id",
				Operator:  OperatorEquals,
				Value:     "$subject.organization_id",
			},
			{
				Attribute: "$resource.status",
				Operator:  OperatorEquals,
				Value:     "pending",
			},
		},
		Enabled: true,
	}
	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	cfg := MiddlewareConfig{
		Engine:              engine,
		ResourceIDExtractor: URLParamExtractor("planID"),
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission("plan:approve", "plan", cfg)
	wrappedHandler := middleware(handler)

	// Create request
	req := httptest.NewRequest("POST", "/api/plans/plan-123/approve", nil)
	req.SetPathValue("planID", "plan-123")
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

// Test that environment attributes are properly passed to the engine
func TestRequirePermission_EnvironmentAttributes(t *testing.T) {
	// Setup with a policy that checks environment attributes
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)

	// Policy that only allows GET requests
	policy := &Policy{
		ID:             "test-policy",
		OrganizationID: "org-1",
		Name:           "GET Only Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "member", Scope: "organization"},
		},
		Actions:   []string{"plan:read"},
		Resources: []string{"*"},
		Conditions: []Condition{
			{
				Attribute: "$environment.method",
				Operator:  OperatorEquals,
				Value:     "GET",
			},
		},
		Enabled: true,
	}
	require.NoError(t, store.CreatePolicy(context.Background(), policy))

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "member",
	}

	cfg := MiddlewareConfig{
		Engine: engine,
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission("plan:read", "plan", cfg)
	wrappedHandler := middleware(handler)

	t.Run("GET request allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/plans", nil)
		ctx := SetSessionInContext(req.Context(), session)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlerCalled = false

		wrappedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.True(t, handlerCalled)
	})

	t.Run("POST request denied", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/plans", nil)
		ctx := SetSessionInContext(req.Context(), session)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handlerCalled = false

		wrappedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.False(t, handlerCalled)
	})
}

// Benchmark the middleware performance
func BenchmarkRequirePermission(b *testing.B) {
	// Setup
	store := NewInMemoryPolicyStore()
	resourceStore := NewInMemoryResourceStore()
	resolver := NewDefaultAttributeResolver(resourceStore)
	engine := NewEngine(store, resolver)

	policy := &Policy{
		ID:             "bench-policy",
		OrganizationID: "org-1",
		Name:           "Benchmark Policy",
		Effect:         EffectAllow,
		Principals: []Principal{
			{Role: "admin", Scope: "organization"},
		},
		Actions:   []string{"*"},
		Resources: []string{"*"},
		Enabled:   true,
	}
	store.CreatePolicy(context.Background(), policy)

	session := &auth.Session{
		UserID:           "user-1",
		OrganizationID:   "org-1",
		OrganizationRole: "admin",
	}

	cfg := MiddlewareConfig{
		Engine:              engine,
		ResourceIDExtractor: URLParamExtractor("id"),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequirePermission("plan:approve", "plan", cfg)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("POST", "/api/plans/123/approve", nil)
	req.SetPathValue("id", "123")
	ctx := SetSessionInContext(req.Context(), session)
	req = req.WithContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}
