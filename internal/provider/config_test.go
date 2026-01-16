package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyRecommendedProviders(t *testing.T) {
	cfg := &ProvidersConfig{
		Providers: []ProviderConfig{
			{Name: "one", Enabled: true},
			{Name: "two", Enabled: true},
		},
	}

	ApplyRecommendedProviders(cfg, []string{"two"})

	if cfg.Providers[0].Enabled {
		t.Error("Expected provider 'one' to be disabled")
	}
	if !cfg.Providers[1].Enabled {
		t.Error("Expected provider 'two' to be enabled")
	}

	ApplyRecommendedProviders(cfg, []string{})
	if !cfg.Providers[0].Enabled {
		t.Error("Expected fallback to first provider to be enabled")
	}
}

func TestConfigFromRecommended(t *testing.T) {
	cfg := ConfigFromRecommended([]string{"alpha", "beta"})
	if len(cfg.Providers) == 0 {
		t.Fatal("Expected default providers to be present")
	}
}

func TestWriteProvidersConfigFromDescriptors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.yaml")
	config, err := WriteProvidersConfigFromDescriptors(path, nil)
	if err != nil {
		t.Fatalf("WriteProvidersConfigFromDescriptors() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file at %s, got %v", path, err)
	}

	examplePath := filepath.Join(filepath.Dir(path), "providers.yaml.example")
	if err := SaveProvidersConfigExample(config, examplePath); err != nil {
		t.Fatalf("SaveProvidersConfigExample() error = %v", err)
	}
	if _, err := os.Stat(examplePath); err != nil {
		t.Fatalf("expected example file at %s, got %v", examplePath, err)
	}
}

func TestLoadProvidersConfigFromBytes(t *testing.T) {
	t.Setenv("SPEC_PROVIDER_PATH", "/tmp/specular-provider")
	data := `
providers:
  - name: test-cli
    type: cli
    enabled: true
    config:
      path: ${SPEC_PROVIDER_PATH}
`
	cfg, err := LoadProvidersConfigFromBytes([]byte(data))
	if err != nil {
		t.Fatalf("LoadProvidersConfigFromBytes() error = %v", err)
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(cfg.Providers))
	}
	path, _ := cfg.Providers[0].Config["path"].(string)
	if path != os.Getenv("SPEC_PROVIDER_PATH") {
		t.Fatalf("env vars were not expanded: got %q", path)
	}
}

func TestLoadProvidersConfigFromBytesInvalid(t *testing.T) {
	_, err := LoadProvidersConfigFromBytes([]byte("providers: []"))
	if err == nil {
		t.Fatalf("expected error for empty provider list")
	}
	if !strings.Contains(err.Error(), "no providers configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRegistryFromProvidersConfig(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-key")
	cfg := &ProvidersConfig{
		Providers: []ProviderConfig{{
			Name:    "openai",
			Type:    ProviderTypeAPI,
			Enabled: true,
			Config: map[string]interface{}{
				"api_key": "fake-key",
			},
		}},
	}

	registry, err := LoadRegistryFromProvidersConfig(cfg)
	if err != nil {
		t.Fatalf("LoadRegistryFromProvidersConfig() error = %v", err)
	}
	list := registry.List()
	if len(list) == 0 {
		t.Fatal("expected registry to contain at least one provider")
	}
	if list[0] != "openai" {
		t.Fatalf("unexpected provider registered: %v", list)
	}
}

func TestLoadRegistryFromProvidersConfigNoProviders(t *testing.T) {
	cfg := &ProvidersConfig{
		Providers: []ProviderConfig{{
			Name:    "openai",
			Type:    ProviderTypeAPI,
			Enabled: false,
			Config: map[string]interface{}{
				"api_key": "fake-key",
			},
		}},
	}

	_, err := LoadRegistryFromProvidersConfig(cfg)
	if err == nil {
		t.Fatal("expected error when no providers load")
	}
	if !strings.Contains(err.Error(), "no providers loaded successfully") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRegistryFromConfig(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy-key")
	cfg := &ProvidersConfig{
		Providers: []ProviderConfig{{
			Name:    "openai",
			Type:    ProviderTypeAPI,
			Enabled: true,
			Config: map[string]interface{}{
				"api_key": "dummy-key",
			},
		}},
	}
	path := filepath.Join(t.TempDir(), "providers.yaml")
	if err := SaveProvidersConfig(cfg, path); err != nil {
		t.Fatalf("SaveProvidersConfig() error = %v", err)
	}

	registry, err := LoadRegistryFromConfig(path)
	if err != nil {
		t.Fatalf("LoadRegistryFromConfig() error = %v", err)
	}
	if len(registry.List()) == 0 {
		t.Fatal("expected registry to have entries")
	}
}

func TestLoadRegistryWithAutoDiscovery(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "auto-key")
	configPath := filepath.Join(t.TempDir(), "providers.yaml")

	registry, err := LoadRegistryWithAutoDiscovery(configPath)
	if err != nil {
		t.Fatalf("LoadRegistryWithAutoDiscovery() error = %v", err)
	}
	if len(registry.List()) == 0 {
		t.Fatal("expected auto-discovery to register providers")
	}
}

func TestLoadRegistryFromAutoDiscoveryNoProviders(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	emptyBin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(emptyBin, 0700); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	t.Setenv("PATH", emptyBin)

	registry, err := LoadRegistryFromAutoDiscovery()
	if err != nil {
		t.Fatalf("LoadRegistryFromAutoDiscovery() error = %v", err)
	}
	found := false
	for _, name := range registry.List() {
		if name == "ollama" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ollama to be registered, got %v", registry.List())
	}
}

func TestGenerateProviderConfigCLI(t *testing.T) {
	tmpBin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(tmpBin, 0700); err != nil {
		t.Fatalf("failed to create temp bin dir: %v", err)
	}

	for _, name := range []string{"claude", "gemini", "copilot", "codex"} {
		exePath := filepath.Join(tmpBin, name)
		if err := os.WriteFile(exePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("failed to create fake binary %s: %v", name, err)
		}
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpBin+string(os.PathListSeparator)+origPath)

	for _, providerName := range []string{"claude-code", "gemini-cli", "copilot-cli", "codex-cli"} {
		cfg := generateProviderConfig(providerName)
		if cfg == nil {
			t.Fatalf("expected CLI provider config for %s", providerName)
		}
		if cfg.Type != ProviderTypeCLI {
			t.Fatalf("expected CLI provider type for %s, got %s", providerName, cfg.Type)
		}
		pathVal, _ := cfg.Config["path"].(string)
		if pathVal == "" {
			t.Fatalf("expected provider config %s to include path", providerName)
		}
	}
}

func TestLoadProvidersConfigMissingFile(t *testing.T) {
	_, err := LoadProvidersConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error when config file is missing")
	}
}

func TestValidateProvidersConfigErrors(t *testing.T) {
	cases := []struct {
		name    string
		config  *ProvidersConfig
		wantErr string
	}{
		{"no providers", &ProvidersConfig{}, "no providers configured"},
		{"no enabled provider", &ProvidersConfig{
			Providers: []ProviderConfig{{Name: "openai", Type: ProviderTypeAPI, Enabled: false}},
		}, "at least one provider must be enabled"},
		{"budget max cost", func() *ProvidersConfig {
			cfg := makeValidProvidersConfig()
			cfg.Strategy.Budget.MaxCostPerDay = -1
			return cfg
		}(), "budget max_cost_per_day must be non-negative"},
		{"budget per request", func() *ProvidersConfig {
			cfg := makeValidProvidersConfig()
			cfg.Strategy.Budget.MaxCostPerRequest = -1
			return cfg
		}(), "budget max_cost_per_request must be non-negative"},
		{"performance latency", func() *ProvidersConfig {
			cfg := makeValidProvidersConfig()
			cfg.Strategy.Performance.MaxLatencyMs = -1
			return cfg
		}(), "performance max_latency_ms must be non-negative"},
		{"fallback retries", func() *ProvidersConfig {
			cfg := makeValidProvidersConfig()
			cfg.Strategy.Fallback.MaxRetries = -1
			return cfg
		}(), "fallback max_retries must be non-negative"},
		{"fallback delay", func() *ProvidersConfig {
			cfg := makeValidProvidersConfig()
			cfg.Strategy.Fallback.RetryDelayMs = -1
			return cfg
		}(), "fallback retry_delay_ms must be non-negative"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProvidersConfig(tc.config)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateProviderConfigErrors(t *testing.T) {
	cases := []struct {
		name   string
		config ProviderConfig
		want   string
	}{
		{"missing name", ProviderConfig{Type: ProviderTypeAPI}, "name is required"},
		{"missing type", ProviderConfig{Name: "foo"}, "type is required"},
		{"invalid type", ProviderConfig{Name: "foo", Type: "bad"}, "invalid provider type"},
		{"cli missing path", ProviderConfig{Name: "cli", Type: ProviderTypeCLI, Enabled: true}, "CLI providers require 'path'"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProviderConfig(&tc.config)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestGenerateProviderConfigAPIs(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "anthropic")
	t.Setenv("OPENAI_API_KEY", "openai")

	cfg := generateProviderConfig("anthropic")
	if cfg == nil || cfg.Type != ProviderTypeAPI {
		t.Fatalf("expected API config for anthropic, got %#v", cfg)
	}

	cfg = generateProviderConfig("openai")
	if cfg == nil || cfg.Type != ProviderTypeAPI {
		t.Fatalf("expected API config for openai, got %#v", cfg)
	}
}

func TestLookupCommandPaths(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(tmp, 0700); err != nil {
		t.Fatalf("failed to create temp bin: %v", err)
	}
	exe := filepath.Join(tmp, "copilot")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}
	t.Setenv("PATH", tmp)

	path, err := lookupCommand("copilot")
	if err != nil {
		t.Fatalf("lookupCommand() unexpected error: %v", err)
	}
	if path == "" {
		t.Fatal("expected copilot path")
	}

	if _, err := lookupCommand("does-not-exist"); err == nil {
		t.Fatal("expected lookupCommand to fail for missing binary")
	}
}

func makeValidProvidersConfig() *ProvidersConfig {
	return &ProvidersConfig{
		Providers: []ProviderConfig{{
			Name:    "openai",
			Type:    ProviderTypeAPI,
			Enabled: true,
			Config: map[string]interface{}{
				"api_key": "valid",
			},
		}},
	}
}

func TestExecutableProviderFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-provider")
	script := `#!/bin/sh
cmd="$1"
case "$cmd" in
  generate)
    cat <<'JSON'
{"content":"hello","tokens_used":1,"provider":"fake","model":"fake","finish_reason":"stop","latency":0}
JSON
    ;;
  stream)
    echo '{"content":"chunk1","delta":"chunk1","done":false}'
    echo '{"content":"chunk2","delta":"chunk2","done":true}'
    ;;
  health)
    exit 0
    ;;
  *)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write executable script: %v", err)
	}

	config := &ProviderConfig{
		Name:    "fake-cli",
		Type:    ProviderTypeCLI,
		Enabled: true,
		Config: map[string]interface{}{
			"capabilities": map[string]interface{}{
				"streaming": true,
			},
		},
	}

	provider, err := NewExecutableProvider(path, config)
	if err != nil {
		t.Fatalf("NewExecutableProvider() error = %v", err)
	}

	ctx := context.Background()
	req := &GenerateRequest{Prompt: "hi"}
	resp, err := provider.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Content != "hello" {
		t.Fatalf("unexpected generate content: %s", resp.Content)
	}

	streamCh, err := provider.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	var seen []string
	for chunk := range streamCh {
		seen = append(seen, chunk.Content)
	}
	if len(seen) != 2 {
		t.Fatalf("expected 2 stream chunks, got %d", len(seen))
	}

	if !provider.IsAvailable() {
		t.Fatal("expected provider to be available")
	}

	if err := provider.Health(ctx); err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if err := provider.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
