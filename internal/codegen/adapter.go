package codegen

import (
	"context"
	"time"

	"github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/plan"
)

// ExecutorAdapter wraps a Generator to implement exec.CodeGenerator
type ExecutorAdapter struct {
	generator *Generator
}

// NewExecutorAdapter creates a new adapter for the exec package
func NewExecutorAdapter(g *Generator) *ExecutorAdapter {
	return &ExecutorAdapter{generator: g}
}

// Generate implements exec.CodeGenerator interface
func (a *ExecutorAdapter) Generate(ctx context.Context, task plan.Task) (*exec.CodeGenerationResult, error) {
	result, err := a.generator.Generate(ctx, task)
	if err != nil {
		return nil, err
	}

	// Convert codegen.GenerationResult to exec.CodeGenerationResult
	execResult := &exec.CodeGenerationResult{
		TaskID:     result.TaskID.String(),
		FeatureID:  result.FeatureID.String(),
		Success:    result.Success,
		Error:      result.Error,
		DurationMs: result.Duration.Milliseconds(),
	}

	// Convert files
	for _, f := range result.Files {
		execResult.Files = append(execResult.Files, exec.GeneratedFileRecord{
			Path:     f.Path,
			Hash:     f.Hash,
			Language: f.Language,
			Size:     f.Size,
		})
	}

	// Convert AI metadata
	execResult.AIGeneration = &exec.AIGenerationRecord{
		Model:        result.AIMetadata.Model,
		Provider:     result.AIMetadata.Provider,
		TokensUsed:   result.AIMetadata.TokensUsed,
		InputTokens:  result.AIMetadata.InputTokens,
		OutputTokens: result.AIMetadata.OutputTokens,
		CostUSD:      result.AIMetadata.CostUSD,
		Latency:      time.Duration(result.AIMetadata.Latency),
	}

	return execResult, nil
}

// Ensure ExecutorAdapter implements exec.CodeGenerator
var _ exec.CodeGenerator = (*ExecutorAdapter)(nil)
