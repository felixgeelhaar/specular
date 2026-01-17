package codegen

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// Generator handles AI-powered code generation for Specular tasks
type Generator struct {
	router        RouterAdapter
	spec          *spec.ProductSpec
	config        Config
	promptBuilder *PromptBuilder
	parser        *ResponseParser
	writer        *FileWriter
}

// NewGenerator creates a new code generator
func NewGenerator(r RouterAdapter, s *spec.ProductSpec, cfg Config) *Generator {
	// Apply defaults
	if cfg.OutputDir == "" {
		cfg.OutputDir = "."
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}

	return &Generator{
		router:        r,
		spec:          s,
		config:        cfg,
		promptBuilder: NewPromptBuilder(),
		parser:        NewResponseParser(),
		writer:        NewFileWriter(cfg.OutputDir, cfg.Verbose, cfg.DryRun),
	}
}

// Generate generates code for a task
func (g *Generator) Generate(ctx context.Context, task plan.Task) (*GenerationResult, error) {
	startTime := time.Now()

	result := &GenerationResult{
		TaskID:    task.ID,
		FeatureID: task.FeatureID,
		Success:   false,
	}

	// 1. Check budget before generation
	if g.router != nil {
		budget := g.router.GetBudget()
		if budget != nil && budget.RemainingUSD <= 0 {
			result.Error = fmt.Sprintf("budget exhausted (spent: $%.2f)", budget.SpentUSD)
			return result, fmt.Errorf("budget exhausted")
		}
	}

	// 2. Find feature in spec
	feature, err := g.findFeature(task.FeatureID)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// 3. Build generation request
	genReq := g.buildRequest(feature, task)

	// 4. Build prompts
	systemPrompt, userPrompt := g.promptBuilder.BuildPrompt(genReq)

	if g.config.Verbose {
		fmt.Printf("🤖 Generating code for feature: %s\n", feature.Title)
		fmt.Printf("   Skill: %s\n", task.Skill)
		fmt.Printf("   Model hint: %s\n", task.ModelHint)
	}

	// 5. Call router to generate
	routerReq := router.GenerateRequest{
		Prompt:       userPrompt,
		SystemPrompt: systemPrompt,
		ModelHint:    "codegen", // Always use codegen model
		Complexity:   task.Estimate,
		Priority:     string(task.Priority),
		MaxTokens:    g.config.MaxTokens,
		Temperature:  g.config.Temperature,
		TaskID:       task.ID,
	}

	resp, err := g.router.Generate(ctx, routerReq)
	if err != nil {
		result.Error = fmt.Sprintf("generation failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("AI generation failed: %w", err)
	}

	// 6. Record AI metadata
	result.AIMetadata = AIGenerationMetadata{
		Model:           resp.Model,
		Provider:        string(resp.Provider),
		TokensUsed:      resp.TokensUsed,
		InputTokens:     resp.InputTokens,
		OutputTokens:    resp.OutputTokens,
		CostUSD:         resp.CostUSD,
		Latency:         resp.Latency,
		SelectionReason: resp.SelectionReason,
	}

	if g.config.Verbose {
		fmt.Printf("   Model used: %s (%s)\n", resp.Model, resp.Provider)
		fmt.Printf("   Tokens: %d (cost: $%.4f)\n", resp.TokensUsed, resp.CostUSD)
	}

	// 7. Parse response into files
	defaultLang := SkillToLanguage(task.Skill)
	files, err := g.parser.ParseResponse(resp.Content, defaultLang)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse response: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("failed to parse AI response: %w", err)
	}

	if g.config.Verbose {
		fmt.Printf("   Parsed %d file(s) from response\n", len(files))
	}

	// 8. Write files to disk (unless dry run)
	if dirErr := g.writer.EnsureBaseDir(); dirErr != nil {
		result.Error = fmt.Sprintf("failed to create output directory: %v", dirErr)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("failed to create output directory: %w", dirErr)
	}

	writtenFiles, err := g.writer.WriteFiles(files)
	if err != nil {
		result.Error = fmt.Sprintf("failed to write files: %v", err)
		result.Duration = time.Since(startTime)
		// Return partial result with written files
		result.Files = writtenFiles
		return result, fmt.Errorf("failed to write generated files: %w", err)
	}

	result.Files = writtenFiles
	result.Duration = time.Since(startTime)
	result.Success = true

	if g.config.Verbose {
		for _, f := range writtenFiles {
			status := "written"
			if !f.Written {
				status = "dry-run"
			}
			fmt.Printf("   📄 %s (%s, %d bytes) [%s]\n", f.Path, f.Language, f.Size, status)
		}
		fmt.Printf("   ✅ Generation completed in %v\n", result.Duration)
	}

	return result, nil
}

// findFeature looks up a feature by ID in the spec
func (g *Generator) findFeature(featureID types.FeatureID) (*spec.Feature, error) {
	if g.spec == nil {
		return nil, fmt.Errorf("no spec loaded")
	}

	for i := range g.spec.Features {
		if g.spec.Features[i].ID == featureID {
			return &g.spec.Features[i], nil
		}
	}

	return nil, fmt.Errorf("feature %s not found in spec", featureID)
}

// buildRequest constructs a CodeGenerationRequest from feature and task
func (g *Generator) buildRequest(feature *spec.Feature, task plan.Task) CodeGenerationRequest {
	req := CodeGenerationRequest{
		FeatureID:       feature.ID,
		FeatureTitle:    feature.Title,
		FeatureDesc:     feature.Desc,
		SuccessCriteria: feature.Success,
		Skill:           task.Skill,
		Priority:        task.Priority,
	}

	// Add product context
	if g.spec != nil {
		req.ProductContext = fmt.Sprintf("Product: %s\nGoals: %v", g.spec.Product, g.spec.Goals)
	}

	// Convert API definitions
	for _, api := range feature.API {
		req.APIs = append(req.APIs, APIEndpoint{
			Method:   api.Method,
			Path:     api.Path,
			Request:  api.Request,
			Response: api.Response,
		})
	}

	return req
}

// GetConfig returns the current configuration
func (g *Generator) GetConfig() Config {
	return g.config
}

// SetConfig updates the configuration
func (g *Generator) SetConfig(cfg Config) {
	g.config = cfg
	g.writer = NewFileWriter(cfg.OutputDir, cfg.Verbose, cfg.DryRun)
}

// WrittenFiles returns the list of files written during generation
func (g *Generator) WrittenFiles() []string {
	return g.writer.WrittenFiles()
}

// CleanupGenerated removes all generated files
func (g *Generator) CleanupGenerated() error {
	return g.writer.CleanupWritten()
}

// GenerateMultiple generates code for multiple tasks
func (g *Generator) GenerateMultiple(ctx context.Context, tasks []plan.Task) ([]*GenerationResult, error) {
	var results []*GenerationResult

	for _, task := range tasks {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		// Only generate for tasks with codegen hint
		if task.ModelHint != "codegen" {
			continue
		}

		result, err := g.Generate(ctx, task)
		results = append(results, result)

		if err != nil {
			// Continue with other tasks on error, unless context is cancelled
			if ctx.Err() != nil {
				return results, ctx.Err()
			}
		}
	}

	return results, nil
}

// ShouldGenerate checks if a task should have code generated
func ShouldGenerate(task plan.Task) bool {
	return task.ModelHint == "codegen"
}

// CodeGenerator is the interface for code generation (for dependency injection)
type CodeGenerator interface {
	Generate(ctx context.Context, task plan.Task) (*GenerationResult, error)
	GetConfig() Config
}

// Ensure Generator implements CodeGenerator
var _ CodeGenerator = (*Generator)(nil)
