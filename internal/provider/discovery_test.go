package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCmdStub(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	body := `#!/bin/sh
exit 0
`
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("failed to write stub %s: %v", name, err)
	}
}

func TestGenerateProviderConfigAPIsFromEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "secret")
	t.Setenv("OPENAI_API_KEY", "secret")

	anthropic := generateProviderConfig("anthropic")
	if anthropic == nil || anthropic.Type != ProviderTypeAPI {
		t.Fatalf("expected anthropic API config, got %+v", anthropic)
	}

	openai := generateProviderConfig("openai")
	if openai == nil || openai.Type != ProviderTypeAPI {
		t.Fatalf("expected openai API config, got %+v", openai)
	}
}

func TestGenerateProviderConfigCLIStubs(t *testing.T) {
	stubDir := t.TempDir()
	writeCmdStub(t, stubDir, "claude")
	orig := os.Getenv("PATH")
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+orig)

	config := generateProviderConfig("claude-code")
	if config == nil {
		t.Fatal("expected cli provider config")
	}
	if config.Type != ProviderTypeCLI {
		t.Fatalf("expected CLI type, got %s", config.Type)
	}
	if _, ok := config.Config["path"]; !ok {
		t.Fatalf("expected CLI config to include path, got %+v", config.Config)
	}
}

func TestLoadRegistryFromProvidersConfigSuccess(t *testing.T) {
	originalStore, originalOrder := snapshotDescriptorState()
	defer restoreDescriptorState(originalStore, originalOrder)
	clearDescriptorRegistry()

	RegisterProviderDescriptor(ProviderDescriptor{
		Name: "test-native",
		Constructor: func(_ *ProviderConfig) (ProviderClient, error) {
			return &fakeProvider{}, nil
		},
	})

	cfg := &ProvidersConfig{
		Providers: []ProviderConfig{
			{
				Name:    "test-native",
				Type:    ProviderTypeNative,
				Enabled: true,
			},
		},
	}

	reg, err := LoadRegistryFromProvidersConfig(cfg)
	if err != nil {
		t.Fatalf("LoadRegistryFromProvidersConfig() error = %v", err)
	}

	if got := len(reg.List()); got != 1 {
		t.Fatalf("expected 1 provider loaded, got %d", got)
	}
}

func TestLoadRegistryFromConfigFileLoadsProvider(t *testing.T) {
	originalStore, originalOrder := snapshotDescriptorState()
	defer restoreDescriptorState(originalStore, originalOrder)
	clearDescriptorRegistry()

	RegisterProviderDescriptor(ProviderDescriptor{
		Name: "file-native",
		Constructor: func(_ *ProviderConfig) (ProviderClient, error) {
			return &fakeProvider{}, nil
		},
	})

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "providers.yaml")
	content := `
providers:
  - name: file-native
    type: native
    enabled: true
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	reg, err := LoadRegistryFromConfig(path)
	if err != nil {
		t.Fatalf("LoadRegistryFromConfig() error = %v", err)
	}

	if got := len(reg.List()); got != 1 {
		t.Fatalf("expected 1 provider loaded from file, got %d", got)
	}
}
