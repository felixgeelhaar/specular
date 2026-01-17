package codegen

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// mockRouter implements RouterAdapter for testing
type mockRouter struct {
	generateFunc func(ctx context.Context, req router.GenerateRequest) (*router.GenerateResponse, error)
	budget       *router.Budget
}

func (m *mockRouter) Generate(ctx context.Context, req router.GenerateRequest) (*router.GenerateResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &router.GenerateResponse{
		Content:      "```go:cmd/main.go\npackage main\nfunc main() {}\n```",
		Model:        "test-model",
		Provider:     "test",
		TokensUsed:   100,
		CostUSD:      0.001,
		Latency:      100 * time.Millisecond,
		FinishReason: "stop",
	}, nil
}

func (m *mockRouter) GetBudget() *router.Budget {
	if m.budget != nil {
		return m.budget
	}
	return &router.Budget{
		LimitUSD:     10.0,
		SpentUSD:     0.0,
		RemainingUSD: 10.0,
		UsageCount:   0,
	}
}

func newTestSpec() *spec.ProductSpec {
	return &spec.ProductSpec{
		Product: "TestProduct",
		Goals:   []string{"Build a test app"},
		Features: []spec.Feature{
			{
				ID:       types.FeatureID("feat-001"),
				Title:    "Test Feature",
				Desc:     "A test feature description",
				Priority: types.Priority("P0"),
				Success:  []string{"Feature works correctly"},
				API: []spec.API{
					{Method: "GET", Path: "/api/test", Response: "TestResponse"},
				},
			},
		},
	}
}

func newTestTask() plan.Task {
	return plan.Task{
		ID:        types.TaskID("task-001"),
		FeatureID: types.FeatureID("feat-001"),
		Skill:     "go-backend",
		Priority:  types.Priority("P0"),
		ModelHint: "codegen",
		Estimate:  5,
	}
}

func TestNewGenerator(t *testing.T) {
	r := &mockRouter{}
	s := newTestSpec()
	cfg := Config{
		OutputDir:   "/tmp/test",
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	g := NewGenerator(r, s, cfg)

	if g == nil {
		t.Fatal("NewGenerator() returned nil")
	}
	if g.config.OutputDir != "/tmp/test" {
		t.Errorf("Config.OutputDir = %q, want %q", g.config.OutputDir, "/tmp/test")
	}
}

func TestNewGenerator_DefaultConfig(t *testing.T) {
	r := &mockRouter{}
	s := newTestSpec()
	cfg := Config{} // Empty config

	g := NewGenerator(r, s, cfg)

	// Should apply defaults
	if g.config.OutputDir != "." {
		t.Errorf("Default OutputDir = %q, want %q", g.config.OutputDir, ".")
	}
	if g.config.MaxTokens != 4096 {
		t.Errorf("Default MaxTokens = %d, want %d", g.config.MaxTokens, 4096)
	}
	if g.config.Temperature != 0.7 {
		t.Errorf("Default Temperature = %f, want %f", g.config.Temperature, 0.7)
	}
}

func TestGenerator_Generate_Success(t *testing.T) {
	tmpDir := t.TempDir()

	r := &mockRouter{
		generateFunc: func(ctx context.Context, req router.GenerateRequest) (*router.GenerateResponse, error) {
			return &router.GenerateResponse{
				Content:      "```go:cmd/main.go\npackage main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n```",
				Model:        "claude-3",
				Provider:     "anthropic",
				TokensUsed:   150,
				InputTokens:  50,
				OutputTokens: 100,
				CostUSD:      0.002,
				Latency:      200 * time.Millisecond,
				FinishReason: "stop",
			}, nil
		},
	}

	s := newTestSpec()
	cfg := Config{
		OutputDir: tmpDir,
		Verbose:   false,
		DryRun:    false,
	}

	g := NewGenerator(r, s, cfg)
	task := newTestTask()

	result, err := g.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !result.Success {
		t.Errorf("Generate() Success = %v, want true", result.Success)
	}
	if result.TaskID != task.ID {
		t.Errorf("Generate() TaskID = %v, want %v", result.TaskID, task.ID)
	}
	if result.FeatureID != task.FeatureID {
		t.Errorf("Generate() FeatureID = %v, want %v", result.FeatureID, task.FeatureID)
	}
	if len(result.Files) != 1 {
		t.Errorf("Generate() got %d files, want 1", len(result.Files))
	}
	if result.Files[0].Path != "cmd/main.go" {
		t.Errorf("Generate() file path = %q, want %q", result.Files[0].Path, "cmd/main.go")
	}

	// Check AI metadata
	if result.AIMetadata.Model != "claude-3" {
		t.Errorf("AIMetadata.Model = %q, want %q", result.AIMetadata.Model, "claude-3")
	}
	if result.AIMetadata.TokensUsed != 150 {
		t.Errorf("AIMetadata.TokensUsed = %d, want %d", result.AIMetadata.TokensUsed, 150)
	}
}

func TestGenerator_Generate_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	r := &mockRouter{}
	s := newTestSpec()
	cfg := Config{
		OutputDir: tmpDir,
		DryRun:    true, // Dry run mode
	}

	g := NewGenerator(r, s, cfg)
	task := newTestTask()

	result, err := g.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !result.Success {
		t.Errorf("Generate() Success = %v, want true", result.Success)
	}

	// Files should not be written in dry-run mode
	for _, f := range result.Files {
		if f.Written {
			t.Errorf("File %s should not be written in dry-run mode", f.Path)
		}
	}
}

func TestGenerator_Generate_BudgetExhausted(t *testing.T) {
	r := &mockRouter{
		budget: &router.Budget{
			LimitUSD:     10.0,
			SpentUSD:     10.0,
			RemainingUSD: 0.0, // No budget remaining
		},
	}

	s := newTestSpec()
	cfg := Config{OutputDir: t.TempDir()}

	g := NewGenerator(r, s, cfg)
	task := newTestTask()

	result, err := g.Generate(context.Background(), task)

	if err == nil {
		t.Error("Generate() should error when budget is exhausted")
	}
	if result.Success {
		t.Error("Generate() Success should be false when budget is exhausted")
	}
}

func TestGenerator_Generate_FeatureNotFound(t *testing.T) {
	r := &mockRouter{}
	s := newTestSpec()
	cfg := Config{OutputDir: t.TempDir()}

	g := NewGenerator(r, s, cfg)

	// Task with non-existent feature
	task := plan.Task{
		ID:        types.TaskID("task-001"),
		FeatureID: types.FeatureID("non-existent-feature"),
		Skill:     "go-backend",
		ModelHint: "codegen",
	}

	result, err := g.Generate(context.Background(), task)

	if err == nil {
		t.Error("Generate() should error when feature is not found")
	}
	if result.Success {
		t.Error("Generate() Success should be false when feature is not found")
	}
}

func TestGenerator_Generate_RouterError(t *testing.T) {
	r := &mockRouter{
		generateFunc: func(ctx context.Context, req router.GenerateRequest) (*router.GenerateResponse, error) {
			return nil, fmt.Errorf("API error")
		},
	}

	s := newTestSpec()
	cfg := Config{OutputDir: t.TempDir()}

	g := NewGenerator(r, s, cfg)
	task := newTestTask()

	result, err := g.Generate(context.Background(), task)

	if err == nil {
		t.Error("Generate() should error when router fails")
	}
	if result.Success {
		t.Error("Generate() Success should be false when router fails")
	}
}

func TestGenerator_Generate_ParseError(t *testing.T) {
	r := &mockRouter{
		generateFunc: func(ctx context.Context, req router.GenerateRequest) (*router.GenerateResponse, error) {
			return &router.GenerateResponse{
				Content:      "No code blocks here, just plain text",
				Model:        "test",
				FinishReason: "stop",
			}, nil
		},
	}

	s := newTestSpec()
	cfg := Config{OutputDir: t.TempDir()}

	g := NewGenerator(r, s, cfg)
	task := newTestTask()

	result, err := g.Generate(context.Background(), task)

	if err == nil {
		t.Error("Generate() should error when response has no code blocks")
	}
	if result.Success {
		t.Error("Generate() Success should be false when parsing fails")
	}
}

func TestGenerator_Generate_ContextCancellation(t *testing.T) {
	r := &mockRouter{
		generateFunc: func(ctx context.Context, req router.GenerateRequest) (*router.GenerateResponse, error) {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			return &router.GenerateResponse{
				Content: "```go:main.go\npackage main\n```",
			}, nil
		},
	}

	s := newTestSpec()
	cfg := Config{OutputDir: t.TempDir()}

	g := NewGenerator(r, s, cfg)
	task := newTestTask()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := g.Generate(ctx, task)

	if err == nil {
		t.Error("Generate() should error when context is cancelled")
	}
}

func TestGenerator_GenerateMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	r := &mockRouter{}
	s := newTestSpec()
	cfg := Config{OutputDir: tmpDir}

	g := NewGenerator(r, s, cfg)

	tasks := []plan.Task{
		{
			ID:        types.TaskID("task-001"),
			FeatureID: types.FeatureID("feat-001"),
			Skill:     "go-backend",
			ModelHint: "codegen",
		},
		{
			ID:        types.TaskID("task-002"),
			FeatureID: types.FeatureID("feat-001"),
			Skill:     "testing",
			ModelHint: "fast", // Not codegen - should be skipped
		},
	}

	results, err := g.GenerateMultiple(context.Background(), tasks)
	if err != nil {
		t.Fatalf("GenerateMultiple() error = %v", err)
	}

	// Only one task has codegen hint
	if len(results) != 1 {
		t.Errorf("GenerateMultiple() got %d results, want 1", len(results))
	}
}

func TestShouldGenerate(t *testing.T) {
	tests := []struct {
		name      string
		modelHint string
		want      bool
	}{
		{"codegen task", "codegen", true},
		{"fast task", "fast", false},
		{"agentic task", "agentic", false},
		{"empty hint", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := plan.Task{ModelHint: tt.modelHint}
			got := ShouldGenerate(task)
			if got != tt.want {
				t.Errorf("ShouldGenerate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerator_GetConfig(t *testing.T) {
	r := &mockRouter{}
	s := newTestSpec()
	cfg := Config{
		OutputDir:   "/custom/dir",
		MaxTokens:   8192,
		Temperature: 0.5,
	}

	g := NewGenerator(r, s, cfg)

	gotCfg := g.GetConfig()
	if gotCfg.OutputDir != cfg.OutputDir {
		t.Errorf("GetConfig().OutputDir = %q, want %q", gotCfg.OutputDir, cfg.OutputDir)
	}
	if gotCfg.MaxTokens != cfg.MaxTokens {
		t.Errorf("GetConfig().MaxTokens = %d, want %d", gotCfg.MaxTokens, cfg.MaxTokens)
	}
}

func TestGenerator_SetConfig(t *testing.T) {
	r := &mockRouter{}
	s := newTestSpec()
	cfg := Config{OutputDir: "/initial"}

	g := NewGenerator(r, s, cfg)

	newCfg := Config{
		OutputDir: "/updated",
		Verbose:   true,
	}
	g.SetConfig(newCfg)

	gotCfg := g.GetConfig()
	if gotCfg.OutputDir != "/updated" {
		t.Errorf("After SetConfig, OutputDir = %q, want %q", gotCfg.OutputDir, "/updated")
	}
	if !gotCfg.Verbose {
		t.Error("After SetConfig, Verbose should be true")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.OutputDir != "." {
		t.Errorf("DefaultConfig().OutputDir = %q, want %q", cfg.OutputDir, ".")
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("DefaultConfig().MaxTokens = %d, want %d", cfg.MaxTokens, 4096)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("DefaultConfig().Temperature = %f, want %f", cfg.Temperature, 0.7)
	}
	if cfg.DryRun {
		t.Error("DefaultConfig().DryRun should be false")
	}
	if cfg.Verbose {
		t.Error("DefaultConfig().Verbose should be false")
	}
}
