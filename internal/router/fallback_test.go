package router

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

type fallbackTestProvider struct {
	name      string
	genErr    error
	streamErr error
}

func (p *fallbackTestProvider) Generate(_ context.Context, _ *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	if p.genErr != nil {
		return nil, p.genErr
	}
	return &provider.GenerateResponse{
		Content:      "ok",
		Provider:     p.name,
		Model:        "test-model",
		TokensUsed:   12,
		InputTokens:  4,
		OutputTokens: 8,
		Latency:      10 * time.Millisecond,
		FinishReason: "stop",
	}, nil
}

func (p *fallbackTestProvider) Stream(_ context.Context, _ *provider.GenerateRequest) (<-chan provider.StreamChunk, error) {
	if p.streamErr != nil {
		return nil, p.streamErr
	}
	out := make(chan provider.StreamChunk, 2)
	out <- provider.StreamChunk{Content: "ok", Delta: "ok", Done: false}
	out <- provider.StreamChunk{Done: true, TokensUsed: 7}
	close(out)
	return out, nil
}

func (p *fallbackTestProvider) GetCapabilities() *provider.ProviderCapabilities {
	return &provider.ProviderCapabilities{SupportsStreaming: true}
}

func (p *fallbackTestProvider) GetInfo() *provider.ProviderInfo {
	return &provider.ProviderInfo{Name: p.name}
}

func (p *fallbackTestProvider) IsAvailable() bool            { return true }
func (p *fallbackTestProvider) Health(context.Context) error { return nil }
func (p *fallbackTestProvider) Close() error                 { return nil }

func makeFallbackRouter(t *testing.T, local provider.ProviderClient, openai provider.ProviderClient) *Router {
	t.Helper()

	reg := provider.NewRegistry()
	if err := reg.Register("ollama", local, &provider.ProviderConfig{Name: "ollama", Enabled: true}); err != nil {
		t.Fatalf("register ollama: %v", err)
	}
	if err := reg.Register("openai", openai, &provider.ProviderConfig{Name: "openai", Enabled: true}); err != nil {
		t.Fatalf("register openai: %v", err)
	}

	r, err := NewRouterWithProviders(&RouterConfig{
		BudgetUSD:         100,
		MaxLatencyMs:      60000,
		EnableFallback:    true,
		MaxRetries:        0,
		RetryBackoffMs:    1,
		RetryMaxBackoffMs: 2,
	}, reg)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	// Force deterministic model set for fallback tests.
	r.models = []Model{
		{ID: "local-primary", Name: "local", Provider: ProviderLocal, Type: ModelTypeFast, CostPerMToken: 0, Available: true, CapabilityScore: 80, ContextWindow: 32000, MaxLatencyMs: 1000},
		{ID: "openai-fallback", Name: "openai", Provider: ProviderOpenAI, Type: ModelTypeFast, CostPerMToken: 1, Available: true, CapabilityScore: 70, ContextWindow: 128000, MaxLatencyMs: 2000},
	}

	return r
}

func TestGenerateWithFallback_SucceedsOnSecondaryProvider(t *testing.T) {
	r := makeFallbackRouter(t,
		&fallbackTestProvider{name: "ollama", genErr: errors.New("primary failed")},
		&fallbackTestProvider{name: "openai"},
	)

	req := GenerateRequest{Prompt: "hello", ModelHint: "fast", Priority: "P1", Complexity: 3, TaskID: types.TaskID("task-1")}
	primary := &RoutingResult{Model: &r.models[0], Reason: "primary"}

	resp, err := r.generateWithFallback(context.Background(), req, primary, time.Now())
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if resp.Provider != "openai" {
		t.Fatalf("provider = %q, want %q", resp.Provider, "openai")
	}
	if resp.SelectionReason == "" {
		t.Fatal("expected fallback selection reason")
	}
}

func TestStreamWithFallback_SucceedsOnSecondaryProvider(t *testing.T) {
	r := makeFallbackRouter(t,
		&fallbackTestProvider{name: "ollama", streamErr: errors.New("primary stream failed")},
		&fallbackTestProvider{name: "openai"},
	)

	req := GenerateRequest{Prompt: "hello", ModelHint: "fast", Priority: "P1", Complexity: 3, TaskID: types.TaskID("task-2")}
	primary := &RoutingResult{Model: &r.models[0], Reason: "primary"}

	stream, err := r.streamWithFallback(context.Background(), req, primary, time.Now())
	if err != nil {
		t.Fatalf("expected stream fallback success, got error: %v", err)
	}

	seenDone := false
	for chunk := range stream {
		if chunk.Done {
			seenDone = true
		}
	}
	if !seenDone {
		t.Fatal("expected done chunk from fallback stream")
	}
}

func TestGenerateWithFallback_AllProvidersFail(t *testing.T) {
	r := makeFallbackRouter(t,
		&fallbackTestProvider{name: "ollama", genErr: errors.New("primary failed")},
		&fallbackTestProvider{name: "openai", genErr: errors.New("fallback failed")},
	)

	req := GenerateRequest{Prompt: "hello", ModelHint: "fast", Priority: "P1", Complexity: 3}
	primary := &RoutingResult{Model: &r.models[0], Reason: "primary"}

	_, err := r.generateWithFallback(context.Background(), req, primary, time.Now())
	if err == nil {
		t.Fatal("expected error when all fallback providers fail")
	}
}
