package router

import (
	"context"
	"testing"
)

// BenchmarkSelectModel_Codegen benchmarks model selection with codegen hint
func BenchmarkSelectModel_Codegen(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    1000.0,
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true) // Enable models for benchmarking

	req := RoutingRequest{
		ModelHint:   "codegen",
		Complexity:  7,
		Priority:    "P0",
		ContextSize: 10000,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.SelectModel(ctx, req)
		if err != nil {
			b.Fatalf("SelectModel failed: %v", err)
		}
	}
}

// BenchmarkSelectModel_Agentic benchmarks model selection with agentic hint
func BenchmarkSelectModel_Agentic(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    1000.0,
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true)

	req := RoutingRequest{
		ModelHint:   "agentic",
		Complexity:  8,
		Priority:    "P0",
		ContextSize: 50000,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.SelectModel(ctx, req)
		if err != nil {
			b.Fatalf("SelectModel failed: %v", err)
		}
	}
}

// BenchmarkSelectModel_LongContext benchmarks model selection with long-context hint
func BenchmarkSelectModel_LongContext(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    1000.0,
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true)

	req := RoutingRequest{
		ModelHint:   "long-context",
		Complexity:  6,
		Priority:    "P1",
		ContextSize: 100000,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.SelectModel(ctx, req)
		if err != nil {
			b.Fatalf("SelectModel failed: %v", err)
		}
	}
}

// BenchmarkSelectModel_Fast benchmarks model selection with fast hint
func BenchmarkSelectModel_Fast(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    1000.0,
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true)

	req := RoutingRequest{
		ModelHint:   "fast",
		Complexity:  3,
		Priority:    "P2",
		ContextSize: 5000,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.SelectModel(ctx, req)
		if err != nil {
			b.Fatalf("SelectModel failed: %v", err)
		}
	}
}

// BenchmarkSelectModel_Cheap benchmarks model selection with cheap hint
func BenchmarkSelectModel_Cheap(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    1000.0,
		MaxLatencyMs: 60000,
		PreferCheap:  true,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true)

	req := RoutingRequest{
		ModelHint:   "cheap",
		Complexity:  4,
		Priority:    "P3",
		ContextSize: 8000,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.SelectModel(ctx, req)
		if err != nil {
			b.Fatalf("SelectModel failed: %v", err)
		}
	}
}

// BenchmarkSelectModel_NoHint benchmarks model selection without hint
func BenchmarkSelectModel_NoHint(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    1000.0,
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true)

	req := RoutingRequest{
		ModelHint:   "",
		Complexity:  5,
		Priority:    "P1",
		ContextSize: 15000,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.SelectModel(ctx, req)
		if err != nil {
			b.Fatalf("SelectModel failed: %v", err)
		}
	}
}

// BenchmarkSelectModel_LowBudget benchmarks model selection with low budget pressure
func BenchmarkSelectModel_LowBudget(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    1.0, // Low budget to force cheaper model selection
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true)

	req := RoutingRequest{
		ModelHint:   "codegen",
		Complexity:  7,
		Priority:    "P0",
		ContextSize: 10000,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset budget for each iteration to test budget checking logic
		router.budget.RemainingUSD = 1.0
		router.budget.SpentUSD = 0.0

		_, err := router.SelectModel(ctx, req)
		if err != nil {
			b.Fatalf("SelectModel failed: %v", err)
		}
	}
}

// BenchmarkSelectModel_HighComplexity benchmarks model selection with high complexity
func BenchmarkSelectModel_HighComplexity(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    1000.0,
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true)

	req := RoutingRequest{
		ModelHint:   "agentic",
		Complexity:  10, // Maximum complexity
		Priority:    "P0",
		ContextSize: 200000, // Large context
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.SelectModel(ctx, req)
		if err != nil {
			b.Fatalf("SelectModel failed: %v", err)
		}
	}
}

// BenchmarkSelectModel_Parallel benchmarks parallel model selection
func BenchmarkSelectModel_Parallel(b *testing.B) {
	router, err := NewRouter(&RouterConfig{
		BudgetUSD:    10000.0, // High budget to avoid budget exhaustion
		MaxLatencyMs: 60000,
		PreferCheap:  false,
	})
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}
	router.SetModelsAvailable(true)

	req := RoutingRequest{
		ModelHint:   "codegen",
		Complexity:  7,
		Priority:    "P0",
		ContextSize: 10000,
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := router.SelectModel(ctx, req)
			if err != nil {
				b.Fatalf("SelectModel failed: %v", err)
			}
		}
	})
}
