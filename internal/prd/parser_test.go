package prd

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/router"
)

// MockProvider implements a simple provider for testing
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &provider.GenerateResponse{
		Content:      m.response,
		TokensUsed:   100,
		InputTokens:  50,
		OutputTokens: 50,
		Latency:      100 * time.Millisecond,
		FinishReason: "stop",
		Model:        "test-model",
		Provider:     "test-provider",
	}, nil
}

func (m *mockProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (<-chan provider.StreamChunk, error) {
	return nil, nil
}

func (m *mockProvider) GetCapabilities() *provider.ProviderCapabilities {
	return &provider.ProviderCapabilities{
		SupportsStreaming: false,
		SupportsTools:     false,
		SupportsMultiTurn: true,
		MaxContextTokens:  4096,
	}
}

func (m *mockProvider) GetInfo() *provider.ProviderInfo {
	return &provider.ProviderInfo{
		Name:    "test-provider",
		Version: "1.0.0",
		Type:    provider.ProviderTypeCLI,
	}
}

func (m *mockProvider) IsAvailable() bool {
	return true
}

func (m *mockProvider) Health(ctx context.Context) error {
	return nil
}

func (m *mockProvider) Close() error {
	return nil
}

func setupTestRouter(response string, err error) (*router.Router, error) {
	// Create a mock provider
	mockProv := &mockProvider{
		response: response,
		err:      err,
	}

	// Create provider registry
	registry := provider.NewRegistry()

	// Create provider config
	providerConfig := &provider.ProviderConfig{
		Name:    "ollama",
		Type:    provider.ProviderTypeCLI,
		Enabled: true,
		Config:  map[string]interface{}{},
	}

	if err := registry.Register("ollama", mockProv, providerConfig); err != nil {
		return nil, err
	}

	// Create router config
	config := &router.RouterConfig{
		BudgetUSD:    10.0,
		MaxLatencyMs: 60000,
		PreferCheap:  true,
	}

	// Create router with the mock provider
	return router.NewRouterWithProviders(config, registry)
}

func TestParsePRD_Success(t *testing.T) {
	validSpecJSON := `{
  "product": "Test Product",
  "goals": ["Goal 1", "Goal 2"],
  "features": [
    {
      "id": "feat-001",
      "title": "Feature One",
      "desc": "Description of feature one",
      "priority": "P0",
      "success": ["Success criterion 1"],
      "trace": ["PRD-section-1"]
    }
  ],
  "non_functional": {
    "performance": ["Performance req 1"],
    "security": ["Security req 1"],
    "scalability": [],
    "availability": []
  },
  "acceptance": ["Acceptance criterion 1"],
  "milestones": []
}`

	r, err := setupTestRouter(validSpecJSON, nil)
	if err != nil {
		t.Fatalf("Failed to setup test router: %v", err)
	}

	parser := NewParser(r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prdContent := "# Test PRD\n\nThis is a test PRD document."
	spec, err := parser.ParsePRD(ctx, prdContent)
	if err != nil {
		t.Fatalf("ParsePRD failed: %v", err)
	}

	if spec.Product != "Test Product" {
		t.Errorf("Expected product 'Test Product', got '%s'", spec.Product)
	}

	if len(spec.Features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(spec.Features))
	}

	if spec.Features[0].ID != "feat-001" {
		t.Errorf("Expected feature ID 'feat-001', got '%s'", spec.Features[0].ID)
	}

	if spec.Features[0].Priority != "P0" {
		t.Errorf("Expected priority 'P0', got '%s'", spec.Features[0].Priority)
	}
}

func TestParsePRD_WithMarkdownCodeBlock(t *testing.T) {
	specInMarkdown := "```json\n" + `{
  "product": "Test Product",
  "goals": ["Goal 1"],
  "features": [
    {
      "id": "feat-001",
      "title": "Feature",
      "desc": "Description",
      "priority": "P1",
      "success": [],
      "trace": []
    }
  ],
  "non_functional": {
    "performance": [],
    "security": [],
    "scalability": [],
    "availability": []
  },
  "acceptance": [],
  "milestones": []
}` + "\n```"

	r, err := setupTestRouter(specInMarkdown, nil)
	if err != nil {
		t.Fatalf("Failed to setup test router: %v", err)
	}

	parser := NewParser(r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prdContent := "# Test PRD\n\nThis is a test PRD."
	spec, err := parser.ParsePRD(ctx, prdContent)
	if err != nil {
		t.Fatalf("ParsePRD failed with markdown code block: %v", err)
	}

	if spec.Product != "Test Product" {
		t.Errorf("Expected product 'Test Product', got '%s'", spec.Product)
	}
}

func TestParsePRD_MissingProduct(t *testing.T) {
	invalidSpecJSON := `{
  "product": "",
  "goals": ["Goal 1"],
  "features": [
    {
      "id": "feat-001",
      "title": "Feature",
      "desc": "Description",
      "priority": "P1",
      "success": [],
      "trace": []
    }
  ],
  "non_functional": {
    "performance": [],
    "security": [],
    "scalability": [],
    "availability": []
  },
  "acceptance": [],
  "milestones": []
}`

	r, err := setupTestRouter(invalidSpecJSON, nil)
	if err != nil {
		t.Fatalf("Failed to setup test router: %v", err)
	}

	parser := NewParser(r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prdContent := "# Test PRD"
	_, err = parser.ParsePRD(ctx, prdContent)
	if err == nil {
		t.Error("Expected error for missing product name, got nil")
	}
	if err != nil && err.Error() != "invalid spec: product name is required" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestParsePRD_MissingFeatures(t *testing.T) {
	invalidSpecJSON := `{
  "product": "Test Product",
  "goals": ["Goal 1"],
  "features": [],
  "non_functional": {
    "performance": [],
    "security": [],
    "scalability": [],
    "availability": []
  },
  "acceptance": [],
  "milestones": []
}`

	r, err := setupTestRouter(invalidSpecJSON, nil)
	if err != nil {
		t.Fatalf("Failed to setup test router: %v", err)
	}

	parser := NewParser(r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prdContent := "# Test PRD"
	_, err = parser.ParsePRD(ctx, prdContent)
	if err == nil {
		t.Error("Expected error for missing features, got nil")
	}
	if err != nil && err.Error() != "invalid spec: at least one feature is required" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestExtractJSONFromMarkdown_CodeBlock(t *testing.T) {
	markdown := "Some text before\n```json\n{\"key\": \"value\"}\n```\nSome text after"
	result := extractJSONFromMarkdown(markdown)
	expected := `{"key": "value"}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestExtractJSONFromMarkdown_NoLanguageTag(t *testing.T) {
	markdown := "```\n{\"key\": \"value\"}\n```"
	result := extractJSONFromMarkdown(markdown)
	expected := `{"key": "value"}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestExtractJSONFromMarkdown_NoBraces(t *testing.T) {
	markdown := "Some text {\"nested\": {\"key\": \"value\"}} more text"
	result := extractJSONFromMarkdown(markdown)
	expected := `{"nested": {"key": "value"}}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestExtractJSONFromMarkdown_NoJSON(t *testing.T) {
	markdown := "Some text without JSON"
	result := extractJSONFromMarkdown(markdown)
	if result != "" {
		t.Errorf("Expected empty string for non-JSON content, got '%s'", result)
	}
}
