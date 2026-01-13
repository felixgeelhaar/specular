package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
)

func init() {
	provider.RegisterNativeProvider("ollama", newOllamaProvider)
}

type ollamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
	info    *provider.ProviderInfo
	caps    *provider.ProviderCapabilities
}

// newOllamaProvider creates a native Ollama provider using the given configuration.
func newOllamaProvider(config *provider.ProviderConfig) (provider.ProviderClient, error) {
	baseURL := "http://localhost:11434"
	if v, ok := config.Config["base_url"].(string); ok && v != "" {
		baseURL = strings.TrimSuffix(v, "/")
	}
	model := "llama3.2"
	if v, ok := config.Config["model"].(string); ok && v != "" {
		model = v
	}

	return &ollamaProvider{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 180 * time.Second,
		},
		info: &provider.ProviderInfo{
			Name:        config.Name,
			Version:     "native",
			Type:        provider.ProviderTypeNative,
			Description: "Native Ollama provider",
			TrustLevel:  provider.TrustLevelBuiltin,
		},
		caps: &provider.ProviderCapabilities{
			SupportsStreaming: false,
			SupportsTools:     false,
			SupportsMultiTurn: true,
			SupportsVision:    false,
			MaxContextTokens:  8192,
			CostPer1KTokens:   0.0,
		},
	}, nil
}

type ollamaRequest struct {
	Model   string   `json:"model"`
	Prompt  string   `json:"prompt"`
	System  string   `json:"system,omitempty"`
	Stream  bool     `json:"stream"`
	Options *options `json:"options,omitempty"`
}

type options struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Model           string `json:"model"`
	Done            bool   `json:"done"`
	Response        string `json:"response"`
	DoneReason      string `json:"done_reason,omitempty"`
	EvalCount       int    `json:"eval_count,omitempty"`
	TotalDuration   int64  `json:"total_duration,omitempty"`
	PromptEvalCount int    `json:"prompt_eval_count,omitempty"`
}

func (o *ollamaProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	request := o.buildOllamaRequest(req)

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/generate", o.baseURL), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	decoder := json.NewDecoder(reader)

	var (
		builder       strings.Builder
		lastModel     string
		finishReason  string
		tokensUsed    int
		totalDuration int64
	)

	streamEnabled := request.Stream

	for {
		var chunk ollamaResponse
		if err := decoder.Decode(&chunk); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("parse ollama response: %w", err)
		}

		if chunk.Response != "" {
			builder.WriteString(chunk.Response)
		}
		if chunk.Model != "" {
			lastModel = chunk.Model
		}
		if chunk.DoneReason != "" {
			finishReason = chunk.DoneReason
		}
		if chunk.EvalCount > tokensUsed {
			tokensUsed = chunk.EvalCount
		}
		if chunk.TotalDuration > totalDuration {
			totalDuration = chunk.TotalDuration
		}

		if chunk.Done {
			if finishReason == "" {
				finishReason = "stop"
			}
			break
		}

		if !streamEnabled {
			break
		}
	}

	if lastModel == "" {
		lastModel = o.model
	}
	if tokensUsed == 0 {
		tokensUsed = 1
	}
	if finishReason == "" {
		finishReason = "stop"
	}

	return &provider.GenerateResponse{
		Content:      builder.String(),
		TokensUsed:   tokensUsed,
		Model:        lastModel,
		Latency:      time.Since(start),
		FinishReason: finishReason,
		Provider:     "ollama",
		Metadata: map[string]interface{}{
			"ollama_total_duration": totalDuration,
		},
	}, nil
}

func (o *ollamaProvider) buildOllamaRequest(req *provider.GenerateRequest) *ollamaRequest {
	prompt := req.Prompt
	if len(req.Context) > 0 {
		var builder strings.Builder
		for _, msg := range req.Context {
			if msg.Role == "assistant" {
				builder.WriteString("Assistant: ")
			} else {
				builder.WriteString("User: ")
			}
			builder.WriteString(msg.Content)
			builder.WriteString("\n")
		}
		builder.WriteString("User: ")
		builder.WriteString(req.Prompt)
		builder.WriteString("\nAssistant:")
		prompt = builder.String()
	}

	model := o.model
	if req.Config != nil {
		if m, ok := req.Config["model"].(string); ok && m != "" {
			model = m
		}
	}

	stream := false
	if req.Config != nil {
		if s, ok := req.Config["stream"].(bool); ok {
			stream = s
		}
	}

	request := &ollamaRequest{
		Model:  model,
		Prompt: prompt,
		System: req.SystemPrompt,
		Stream: stream,
	}

	if req.Temperature > 0 || req.TopP > 0 || req.MaxTokens > 0 {
		request.Options = &options{
			Temperature: req.Temperature,
			TopP:        req.TopP,
			NumPredict:  req.MaxTokens,
		}
	}

	return request
}

func (o *ollamaProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (<-chan provider.StreamChunk, error) {
	return nil, fmt.Errorf("streaming not supported by Ollama native provider")
}

func (o *ollamaProvider) GetCapabilities() *provider.ProviderCapabilities {
	return o.caps
}

func (o *ollamaProvider) GetInfo() *provider.ProviderInfo {
	return o.info
}

func (o *ollamaProvider) IsAvailable() bool {
	return o.ping(context.Background()) == nil
}

func (o *ollamaProvider) Health(ctx context.Context) error {
	return o.ping(ctx)
}

func (o *ollamaProvider) ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/version", o.baseURL), nil)
	if err != nil {
		return err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("unexpected status %d", resp.StatusCode)
}

func (o *ollamaProvider) Close() error {
	return nil
}
