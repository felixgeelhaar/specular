package router

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
)

func TestNewRouter(t *testing.T) {
	tests := []struct {
		name     string
		config   *RouterConfig
		wantErr  bool
		validate func(*testing.T, *Router)
	}{
		{
			name: "valid config",
			config: &RouterConfig{
				BudgetUSD:    20.0,
				MaxLatencyMs: 60000,
			},
			wantErr: false,
			validate: func(t *testing.T, r *Router) {
				if r.budget.LimitUSD != 20.0 {
					t.Errorf("Budget limit = %v, want 20.0", r.budget.LimitUSD)
				}
			},
		},
		{
			name: "config with context validation enabled",
			config: &RouterConfig{
				BudgetUSD:               20.0,
				MaxLatencyMs:            60000,
				EnableContextValidation: true,
				TruncationStrategy:      "oldest",
			},
			wantErr: false,
			validate: func(t *testing.T, r *Router) {
				if r.contextValidator == nil {
					t.Error("contextValidator should be initialized when EnableContextValidation is true")
				}
				if r.contextTruncator == nil {
					t.Error("contextTruncator should be initialized when EnableContextValidation is true")
				}
			},
		},
		{
			name: "config with context validation and default truncation strategy",
			config: &RouterConfig{
				BudgetUSD:               20.0,
				MaxLatencyMs:            60000,
				EnableContextValidation: true,
				TruncationStrategy:      "", // Empty should default to TruncateOldest
			},
			wantErr: false,
			validate: func(t *testing.T, r *Router) {
				if r.contextValidator == nil {
					t.Error("contextValidator should be initialized")
				}
				if r.contextTruncator == nil {
					t.Error("contextTruncator should be initialized")
				}
			},
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, err := NewRouter(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRouter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if router == nil {
					t.Error("NewRouter() returned nil router")
					return
				}
				if tt.validate != nil {
					tt.validate(t, router)
				}
			}
		})
	}
}

func TestNewRouterWithProviders(t *testing.T) {
	tests := []struct {
		name     string
		config   *RouterConfig
		registry *provider.Registry
		wantErr  bool
		validate func(*testing.T, *Router)
	}{
		{
			name: "with existing registry",
			config: &RouterConfig{
				BudgetUSD:    20.0,
				MaxLatencyMs: 60000,
			},
			registry: provider.NewRegistry(),
			wantErr:  false,
			validate: func(t *testing.T, r *Router) {
				if r.registry == nil {
					t.Error("registry should not be nil")
				}
			},
		},
		{
			name: "with nil registry - should create new",
			config: &RouterConfig{
				BudgetUSD:    20.0,
				MaxLatencyMs: 60000,
			},
			registry: nil,
			wantErr:  false,
			validate: func(t *testing.T, r *Router) {
				if r.registry == nil {
					t.Error("registry should be created when nil is passed")
				}
			},
		},
		{
			name: "with context validation enabled",
			config: &RouterConfig{
				BudgetUSD:               20.0,
				MaxLatencyMs:            60000,
				EnableContextValidation: true,
				TruncationStrategy:      "newest",
			},
			registry: provider.NewRegistry(),
			wantErr:  false,
			validate: func(t *testing.T, r *Router) {
				if r.contextValidator == nil {
					t.Error("contextValidator should be initialized")
				}
				if r.contextTruncator == nil {
					t.Error("contextTruncator should be initialized")
				}
			},
		},
		{
			name:     "nil config should error",
			config:   nil,
			registry: provider.NewRegistry(),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, err := NewRouterWithProviders(tt.config, tt.registry)

			if tt.wantErr {
				if err == nil {
					t.Error("NewRouterWithProviders() expected error, got nil")
				}
				if router != nil {
					t.Error("NewRouterWithProviders() expected nil router on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewRouterWithProviders() unexpected error = %v", err)
				}
				if router == nil {
					t.Error("NewRouterWithProviders() returned nil router")
					return
				}
				if tt.validate != nil {
					tt.validate(t, router)
				}
			}
		})
	}
}

func TestSelectModel(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:    100.0,
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	router.SetModelsAvailable(true) // Enable models for testing

	tests := []struct {
		name    string
		request RoutingRequest
		wantErr bool
	}{
		{
			name: "codegen hint",
			request: RoutingRequest{
				ModelHint:   "codegen",
				Complexity:  7,
				Priority:    "P0",
				ContextSize: 10000,
			},
			wantErr: false,
		},
		{
			name: "agentic hint",
			request: RoutingRequest{
				ModelHint:   "agentic",
				Complexity:  8,
				Priority:    "P0",
				ContextSize: 50000,
			},
			wantErr: false,
		},
		{
			name: "long-context hint",
			request: RoutingRequest{
				ModelHint:   "long-context",
				Complexity:  6,
				Priority:    "P1",
				ContextSize: 100000,
			},
			wantErr: false,
		},
		{
			name: "fast hint",
			request: RoutingRequest{
				ModelHint:   "fast",
				Complexity:  3,
				Priority:    "P2",
				ContextSize: 5000,
			},
			wantErr: false,
		},
		{
			name: "no hint",
			request: RoutingRequest{
				Complexity:  5,
				Priority:    "P1",
				ContextSize: 10000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := router.SelectModel(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == nil {
					t.Error("SelectModel() returned nil result")
					return
				}
				if result.Model == nil {
					t.Error("SelectModel() returned nil model")
					return
				}
				if result.Reason == "" {
					t.Error("SelectModel() returned empty reason")
				}
				if result.EstimatedCost < 0 {
					t.Error("SelectModel() returned negative cost")
				}
			}
		})
	}
}

func TestBudgetManagement(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:    1.0, // Low budget to test exhaustion
		MaxLatencyMs: 60000,
	})
	router.SetModelsAvailable(true) // Enable models for testing

	// First request should succeed
	result1, err := router.SelectModel(RoutingRequest{
		ModelHint:   "cheap",
		Complexity:  3,
		Priority:    "P2",
		ContextSize: 5000,
	})
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	// Record usage that consumes budget
	_ = router.RecordUsage(Usage{
		Model:     result1.Model.ID,
		Provider:  result1.Model.Provider,
		Tokens:    10000,
		CostUSD:   0.5,
		LatencyMs: 2000,
		Timestamp: time.Now(),
		Success:   true,
	})

	if router.budget.SpentUSD != 0.5 {
		t.Errorf("Spent = %v, want 0.5", router.budget.SpentUSD)
	}
	if router.budget.RemainingUSD != 0.5 {
		t.Errorf("Remaining = %v, want 0.5", router.budget.RemainingUSD)
	}

	// Consume more budget
	_ = router.RecordUsage(Usage{
		Model:     result1.Model.ID,
		Provider:  result1.Model.Provider,
		Tokens:    10000,
		CostUSD:   0.6,
		LatencyMs: 2000,
		Timestamp: time.Now(),
		Success:   true,
	})

	// Budget should be exhausted
	_, err = router.SelectModel(RoutingRequest{
		ModelHint:  "codegen",
		Complexity: 7,
		Priority:   "P0",
	})

	if err == nil {
		t.Error("Expected error when budget exhausted, got nil")
	}
}

func TestModelScoring(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:    100.0,
		MaxLatencyMs: 60000,
		PreferCheap:  true,
	})
	router.SetModelsAvailable(true) // Enable models for testing

	// High complexity P0 task should get capable model
	result1, err := router.SelectModel(RoutingRequest{
		ModelHint:   "codegen",
		Complexity:  9,
		Priority:    "P0",
		ContextSize: 50000,
	})
	if err != nil {
		t.Fatalf("SelectModel() failed: %v", err)
	}
	if result1.Model.CapabilityScore < 80 {
		t.Errorf("High complexity task got low capability model: %v", result1.Model.ID)
	}

	// Low complexity P2 task with cheap preference should get cheaper model
	router2, _ := NewRouter(&RouterConfig{
		BudgetUSD:    100.0,
		MaxLatencyMs: 60000,
		PreferCheap:  true,
	})
	router2.SetModelsAvailable(true) // Enable models for testing

	result2, err := router2.SelectModel(RoutingRequest{
		ModelHint:   "fast",
		Complexity:  2,
		Priority:    "P2",
		ContextSize: 5000,
	})
	if err != nil {
		t.Fatalf("SelectModel() failed: %v", err)
	}
	if result2.Model.CostPerMToken > 1.0 {
		t.Errorf("Low complexity task with cheap preference got expensive model: %v (cost: %.2f)", result2.Model.ID, result2.Model.CostPerMToken)
	}
}

func TestGetModelHelpers(t *testing.T) {
	// Test GetModelByID - found case
	model := GetModelByID("claude-sonnet-4")
	if model == nil {
		t.Error("GetModelByID() returned nil for known model")
	}
	if model != nil && model.ID != "claude-sonnet-4" {
		t.Errorf("GetModelByID() returned wrong model: %v", model.ID)
	}

	// Test GetModelByID - not found case
	notFound := GetModelByID("non-existent-model-xyz")
	if notFound != nil {
		t.Errorf("GetModelByID() returned non-nil for unknown model: %v", notFound.ID)
	}

	// Test GetCheapestModel
	cheapest := GetCheapestModel()
	if cheapest == nil {
		t.Error("GetCheapestModel() returned nil")
	}
	if cheapest != nil && cheapest.CostPerMToken > 0.5 {
		t.Errorf("GetCheapestModel() returned expensive model: cost = %.2f", cheapest.CostPerMToken)
	}

	// Test GetFastestModel
	fastest := GetFastestModel()
	if fastest == nil {
		t.Error("GetFastestModel() returned nil")
	}
	if fastest != nil && fastest.MaxLatencyMs > 2000 {
		t.Errorf("GetFastestModel() returned slow model: latency = %d", fastest.MaxLatencyMs)
	}

	// Test GetBestModel
	best := GetBestModel()
	if best == nil {
		t.Error("GetBestModel() returned nil")
	}
	if best != nil && best.CapabilityScore < 90 {
		t.Errorf("GetBestModel() returned low capability model: score = %.1f", best.CapabilityScore)
	}
}

func TestContextSizeFiltering(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:    100.0,
		MaxLatencyMs: 60000,
	})
	router.SetModelsAvailable(true) // Enable models for testing

	// Request with large context (100k tokens)
	result, err := router.SelectModel(RoutingRequest{
		ModelHint:   "long-context",
		Complexity:  6,
		Priority:    "P1",
		ContextSize: 100000,
	})

	if err != nil {
		t.Fatalf("SelectModel() failed: %v", err)
	}

	if result.Model.ContextWindow < 100000 {
		t.Errorf("Selected model context window (%d) is smaller than required (%d)", result.Model.ContextWindow, 100000)
	}
}

func TestUsageStats(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:    100.0,
		MaxLatencyMs: 60000,
	})
	router.SetModelsAvailable(true) // Enable models for testing

	// Record some usage
	_ = router.RecordUsage(Usage{
		Model:     "claude-sonnet-4",
		Provider:  ProviderAnthropic,
		Tokens:    5000,
		CostUSD:   0.15,
		LatencyMs: 3000,
		Timestamp: time.Now(),
		TaskID:    "task-001",
		Success:   true,
	})

	_ = router.RecordUsage(Usage{
		Model:     "gpt-4o",
		Provider:  ProviderOpenAI,
		Tokens:    3000,
		CostUSD:   0.08,
		LatencyMs: 2500,
		Timestamp: time.Now(),
		TaskID:    "task-002",
		Success:   true,
	})

	stats := router.GetUsageStats()

	if stats["total_requests"].(int) != 2 {
		t.Errorf("Total requests = %v, want 2", stats["total_requests"])
	}

	budgetSpent := stats["budget_spent"].(float64)
	expectedSpent := 0.23
	tolerance := 0.001
	if budgetSpent < expectedSpent-tolerance || budgetSpent > expectedSpent+tolerance {
		t.Errorf("Budget spent = %v, want approximately %v", budgetSpent, expectedSpent)
	}
}

func TestGetModelsByType(t *testing.T) {
	tests := []struct {
		name      string
		modelType ModelType
		wantLen   int
	}{
		{
			name:      "codegen models",
			modelType: ModelTypeCodegen,
			wantLen:   3, // claude-sonnet-3.5, gpt-4o, codellama
		},
		{
			name:      "agentic models",
			modelType: ModelTypeAgentic,
			wantLen:   2, // claude-sonnet-4, llama3
		},
		{
			name:      "fast models",
			modelType: ModelTypeFast,
			wantLen:   3, // claude-haiku-3.5, gpt-3.5-turbo, llama3.2
		},
		{
			name:      "long-context models",
			modelType: ModelTypeLongContext,
			wantLen:   1, // gpt-4-turbo
		},
		{
			name:      "cheap models",
			modelType: ModelTypeCheap,
			wantLen:   1, // gpt-4o-mini
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models := GetModelsByType(tt.modelType)
			if len(models) != tt.wantLen {
				t.Errorf("GetModelsByType(%s) returned %d models, want %d", tt.modelType, len(models), tt.wantLen)
			}
			// Verify all returned models have the correct type
			for _, m := range models {
				if m.Type != tt.modelType {
					t.Errorf("GetModelsByType(%s) returned model %s with wrong type %s", tt.modelType, m.ID, m.Type)
				}
			}
		})
	}
}

func TestGetModelsByProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		wantLen  int
	}{
		{
			name:     "anthropic models",
			provider: ProviderAnthropic,
			wantLen:  3, // claude-sonnet-4, claude-sonnet-3.5, claude-haiku-3.5
		},
		{
			name:     "openai models",
			provider: ProviderOpenAI,
			wantLen:  4, // gpt-4-turbo, gpt-4o, gpt-4o-mini, gpt-3.5-turbo
		},
		{
			name:     "local models",
			provider: ProviderLocal,
			wantLen:  3, // llama3.2, codellama, llama3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models := GetModelsByProvider(tt.provider)
			if len(models) != tt.wantLen {
				t.Errorf("GetModelsByProvider(%s) returned %d models, want %d", tt.provider, len(models), tt.wantLen)
			}
			// Verify all returned models have the correct provider
			for _, m := range models {
				if m.Provider != tt.provider {
					t.Errorf("GetModelsByProvider(%s) returned model %s with wrong provider %s", tt.provider, m.ID, m.Provider)
				}
			}
		})
	}
}

func TestGetBudget(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:    50.0,
		MaxLatencyMs: 60000,
	})
	router.SetModelsAvailable(true) // Enable models for testing

	budget := router.GetBudget()
	if budget == nil {
		t.Fatal("GetBudget() returned nil")
	}

	if budget.LimitUSD != 50.0 {
		t.Errorf("Budget limit = %v, want 50.0", budget.LimitUSD)
	}

	if budget.SpentUSD != 0.0 {
		t.Errorf("Budget spent = %v, want 0.0", budget.SpentUSD)
	}

	if budget.RemainingUSD != 50.0 {
		t.Errorf("Budget remaining = %v, want 50.0", budget.RemainingUSD)
	}

	// Record some usage
	_ = router.RecordUsage(Usage{
		Model:     "claude-sonnet-4",
		Provider:  ProviderAnthropic,
		Tokens:    10000,
		CostUSD:   1.5,
		LatencyMs: 3000,
		Timestamp: time.Now(),
		Success:   true,
	})

	// Check budget updated
	budget = router.GetBudget()
	if budget.SpentUSD != 1.5 {
		t.Errorf("Budget spent after usage = %v, want 1.5", budget.SpentUSD)
	}

	if budget.RemainingUSD != 48.5 {
		t.Errorf("Budget remaining after usage = %v, want 48.5", budget.RemainingUSD)
	}

	if budget.UsageCount != 1 {
		t.Errorf("Usage count = %v, want 1", budget.UsageCount)
	}
}

func TestFindCheaperModel(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:    10.0,
		MaxLatencyMs: 60000,
	})
	router.SetModelsAvailable(true) // Enable models for testing

	// Create a set of candidate models with different costs
	candidates := []Model{
		{
			ID:            "expensive-model",
			Provider:      ProviderOpenAI,
			Type:          ModelTypeCodegen,
			CostPerMToken: 10.0, // Expensive
			ContextWindow: 100000,
		},
		{
			ID:            "moderate-model",
			Provider:      ProviderAnthropic,
			Type:          ModelTypeCodegen,
			CostPerMToken: 3.0, // Moderate
			ContextWindow: 100000,
		},
		{
			ID:            "cheap-model",
			Provider:      ProviderAnthropic,
			Type:          ModelTypeFast,
			CostPerMToken: 0.8, // Cheap
			ContextWindow: 100000,
		},
	}

	// Should find the cheapest model below max cost
	maxCost := 5.0
	cheaper := router.findCheaperModel(candidates, maxCost)

	if cheaper == nil {
		t.Fatal("findCheaperModel() returned nil")
	}

	if cheaper.ID != "cheap-model" {
		t.Errorf("findCheaperModel() returned %s, want cheap-model", cheaper.ID)
	}

	// Test with max cost that excludes all but expensive
	maxCost = 100.0
	cheaper = router.findCheaperModel(candidates, maxCost)

	if cheaper == nil {
		t.Fatal("findCheaperModel() with high max cost returned nil")
	}

	if cheaper.CostPerMToken >= maxCost {
		t.Errorf("findCheaperModel() returned model with cost %.2f >= maxCost %.2f", cheaper.CostPerMToken, maxCost)
	}
}

func TestGetProviderName(t *testing.T) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    10.0,
		MaxLatencyMs: 60000,
	})
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	tests := []struct {
		name     string
		provider Provider
		want     string
	}{
		{
			name:     "anthropic provider",
			provider: ProviderAnthropic,
			want:     "anthropic",
		},
		{
			name:     "openai provider",
			provider: ProviderOpenAI,
			want:     "openai",
		},
		{
			name:     "local provider",
			provider: ProviderLocal,
			want:     "local",
		},
		{
			name:     "unknown provider",
			provider: Provider("unknown"),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.getProviderName(tt.provider)
			if got != tt.want {
				t.Errorf("getProviderName(%s) = %s, want %s", tt.provider, got, tt.want)
			}
		})
	}
}

func TestSelectModel_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		budgetUSD   float64
		setupRouter func(*Router)
		request     RoutingRequest
		wantErr     bool
		errContains string
	}{
		{
			name:      "budget exhausted",
			budgetUSD: 1.0,
			setupRouter: func(r *Router) {
				// Exhaust the budget by recording some spend
				r.budget.SpentUSD = 1.0
				r.budget.RemainingUSD = 0
			},
			request: RoutingRequest{
				ModelHint:   "fast",
				Complexity:  5,
				Priority:    "P1",
				ContextSize: 1000,
			},
			wantErr:     true,
			errContains: "budget exhausted",
		},
		{
			name:      "no suitable models - all unavailable",
			budgetUSD: 100.0,
			setupRouter: func(r *Router) {
				r.SetModelsAvailable(false) // Make all models unavailable
			},
			request: RoutingRequest{
				ModelHint:   "codegen",
				Complexity:  7,
				Priority:    "P0",
				ContextSize: 10000,
			},
			wantErr:     true,
			errContains: "no suitable models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := NewRouter(&RouterConfig{
				BudgetUSD:    tt.budgetUSD,
				MaxLatencyMs: 60000,
			})

			if tt.setupRouter != nil {
				tt.setupRouter(router)
			}

			result, err := router.SelectModel(tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("SelectModel() expected error, got nil")
				} else if !contains(err.Error(), tt.errContains) {
					t.Errorf("SelectModel() error = %v, want error containing %q", err, tt.errContains)
				}
				if result != nil {
					t.Error("SelectModel() expected nil result on error")
				}
			} else {
				if err != nil {
					t.Errorf("SelectModel() unexpected error = %v", err)
				}
				if result == nil {
					t.Error("SelectModel() returned nil result")
				}
			}
		})
	}
}
