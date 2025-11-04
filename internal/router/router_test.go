package router

import (
	"testing"
	"time"
)

func TestNewRouter(t *testing.T) {
	tests := []struct {
		name    string
		config  *RouterConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &RouterConfig{
				BudgetUSD:    20.0,
				MaxLatencyMs: 60000,
			},
			wantErr: false,
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
			if !tt.wantErr && router == nil {
				t.Error("NewRouter() returned nil router")
			}
			if !tt.wantErr && router.budget.LimitUSD != tt.config.BudgetUSD {
				t.Errorf("Budget limit = %v, want %v", router.budget.LimitUSD, tt.config.BudgetUSD)
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
	router.RecordUsage(Usage{
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
	router.RecordUsage(Usage{
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
	// Test GetModelByID
	model := GetModelByID("claude-sonnet-4")
	if model == nil {
		t.Error("GetModelByID() returned nil for known model")
	}
	if model != nil && model.ID != "claude-sonnet-4" {
		t.Errorf("GetModelByID() returned wrong model: %v", model.ID)
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

	// Record some usage
	router.RecordUsage(Usage{
		Model:     "claude-sonnet-4",
		Provider:  ProviderAnthropic,
		Tokens:    5000,
		CostUSD:   0.15,
		LatencyMs: 3000,
		Timestamp: time.Now(),
		TaskID:    "task-001",
		Success:   true,
	})

	router.RecordUsage(Usage{
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
