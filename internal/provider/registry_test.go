package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type fakeProvider struct {
	closeErr error
	closed   bool
}

func (f *fakeProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Content: "ok", Provider: "fake"}, nil
}

func (f *fakeProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	close(ch)
	return ch, nil
}

func (f *fakeProvider) GetCapabilities() *ProviderCapabilities {
	return &ProviderCapabilities{SupportsStreaming: true}
}

func (f *fakeProvider) GetInfo() *ProviderInfo {
	return &ProviderInfo{Name: "fake", Type: ProviderTypeNative}
}

func (f *fakeProvider) IsAvailable() bool {
	return true
}

func (f *fakeProvider) Health(ctx context.Context) error {
	return nil
}

func (f *fakeProvider) Close() error {
	if f.closeErr != nil {
		return f.closeErr
	}
	f.closed = true
	return nil
}

func TestRegistryLifecycle(t *testing.T) {
	reg := NewRegistry()

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:        "test-registry",
		Description: "fake provider",
		Constructor: func(_ *ProviderConfig) (ProviderClient, error) {
			return &fakeProvider{}, nil
		},
	})

	cfg := &ProviderConfig{
		Name:    "test-registry",
		Type:    ProviderTypeNative,
		Enabled: true,
	}

	if err := reg.LoadFromConfig(cfg); err != nil {
		t.Fatalf("LoadFromConfig failed: %v", err)
	}

	client, err := reg.Get(cfg.Name)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected provider client, got nil")
	}

	if err := reg.Remove(cfg.Name); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if _, err := reg.Get(cfg.Name); err == nil {
		t.Fatal("expected error after removing provider")
	}
}

func TestRegistryCloseAllCollectsErrors(t *testing.T) {
	reg := NewRegistry()

	p1 := &fakeProvider{closeErr: fmt.Errorf("boom")}
	p2 := &fakeProvider{}

	if err := reg.Register("first", p1, &ProviderConfig{Name: "first"}); err != nil {
		t.Fatalf("failed to register first provider: %v", err)
	}
	if err := reg.Register("second", p2, &ProviderConfig{Name: "second"}); err != nil {
		t.Fatalf("failed to register second provider: %v", err)
	}
	if err := reg.Register("second", p2, &ProviderConfig{Name: "second"}); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}

	err := reg.CloseAll()
	if err == nil {
		t.Fatal("expected CloseAll to return error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected error to mention boom, got %v", err)
	}
	if !p2.closed {
		t.Fatal("expected second provider to be closed")
	}
}

func TestRegistryGetConfig(t *testing.T) {
	reg := NewRegistry()

	expectedConfig := &ProviderConfig{
		Name:    "test-provider",
		Type:    ProviderTypeNative,
		Enabled: true,
		Config:  map[string]interface{}{"key": "value"},
	}

	provider := &fakeProvider{}
	if err := reg.Register("test-provider", provider, expectedConfig); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Test successful GetConfig
	config, err := reg.GetConfig("test-provider")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if config.Name != expectedConfig.Name {
		t.Errorf("GetConfig returned wrong name: got %s, want %s", config.Name, expectedConfig.Name)
	}
	if config.Type != expectedConfig.Type {
		t.Errorf("GetConfig returned wrong type: got %s, want %s", config.Type, expectedConfig.Type)
	}

	// Test GetConfig for non-existent provider
	_, err = reg.GetConfig("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()

	// Empty registry
	names := reg.List()
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}

	// Add providers
	_ = reg.Register("provider-a", &fakeProvider{}, &ProviderConfig{Name: "provider-a"})
	_ = reg.Register("provider-b", &fakeProvider{}, &ProviderConfig{Name: "provider-b"})

	names = reg.List()
	if len(names) != 2 {
		t.Errorf("expected 2 providers, got %d", len(names))
	}
}

func TestRegistryRemoveErrors(t *testing.T) {
	reg := NewRegistry()

	// Remove non-existent provider
	err := reg.Remove("nonexistent")
	if err == nil {
		t.Fatal("expected error removing nonexistent provider")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}

	// Provider with close error
	closeErr := fmt.Errorf("close failed")
	provider := &fakeProvider{closeErr: closeErr}
	_ = reg.Register("failing", provider, &ProviderConfig{Name: "failing"})

	err = reg.Remove("failing")
	if err == nil {
		t.Fatal("expected error when provider close fails")
	}
	if !strings.Contains(err.Error(), "close failed") {
		t.Errorf("expected 'close failed' error, got: %v", err)
	}
}

func TestLoadFromConfigDisabled(t *testing.T) {
	reg := NewRegistry()

	cfg := &ProviderConfig{
		Name:    "disabled-provider",
		Type:    ProviderTypeNative,
		Enabled: false,
	}

	// Loading a disabled provider should succeed silently
	if err := reg.LoadFromConfig(cfg); err != nil {
		t.Fatalf("LoadFromConfig should skip disabled providers: %v", err)
	}

	// Provider should not be registered
	_, err := reg.Get("disabled-provider")
	if err == nil {
		t.Fatal("disabled provider should not be registered")
	}
}

func TestLoadFromConfigEmptyName(t *testing.T) {
	reg := NewRegistry()

	cfg := &ProviderConfig{
		Name:    "",
		Enabled: true,
	}

	err := reg.LoadFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty provider name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("expected 'name is required' error, got: %v", err)
	}
}

func TestNewProviderClientFromConfigCLI(t *testing.T) {
	// CLI provider without path should fail
	cfg := &ProviderConfig{
		Name:    "cli-test",
		Type:    ProviderTypeCLI,
		Enabled: true,
		Config:  map[string]interface{}{},
	}

	_, err := newProviderClientFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for CLI provider without path")
	}
	if !strings.Contains(err.Error(), "executable path required") {
		t.Errorf("expected 'executable path required' error, got: %v", err)
	}

	// CLI provider with empty path
	cfg.Config["path"] = ""
	_, err = newProviderClientFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for CLI provider with empty path")
	}
}

func TestNewProviderClientFromConfigUnknownAPI(t *testing.T) {
	cfg := &ProviderConfig{
		Name:    "unknown-api",
		Type:    ProviderTypeAPI,
		Enabled: true,
	}

	_, err := newProviderClientFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown API provider")
	}
	if !strings.Contains(err.Error(), "unknown API provider") {
		t.Errorf("expected 'unknown API provider' error, got: %v", err)
	}
}

func TestNewProviderClientFromConfigGRPC(t *testing.T) {
	cfg := &ProviderConfig{
		Name:    "grpc-test",
		Type:    ProviderTypeGRPC,
		Enabled: true,
	}

	_, err := newProviderClientFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for gRPC provider (not implemented)")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("expected 'not yet implemented' error, got: %v", err)
	}
}

func TestNewProviderClientFromConfigUnknownType(t *testing.T) {
	cfg := &ProviderConfig{
		Name:    "unknown-type",
		Type:    "invalid-type",
		Enabled: true,
	}

	_, err := newProviderClientFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider type")
	}
	if !strings.Contains(err.Error(), "unknown provider type") {
		t.Errorf("expected 'unknown provider type' error, got: %v", err)
	}
}

func TestNewProviderClientFromConfigNativeNotRegistered(t *testing.T) {
	cfg := &ProviderConfig{
		Name:    "unregistered-native",
		Type:    ProviderTypeNative,
		Enabled: true,
	}

	_, err := newProviderClientFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unregistered native provider")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("expected 'not registered' error, got: %v", err)
	}
}

func TestNativeConstructorFor(t *testing.T) {
	// Register a descriptor with constructor
	RegisterProviderDescriptor(ProviderDescriptor{
		Name:        "native-test-constructor",
		Description: "test native provider",
		Constructor: func(_ *ProviderConfig) (ProviderClient, error) {
			return &fakeProvider{}, nil
		},
	})

	constructor, ok := nativeConstructorFor("native-test-constructor")
	if !ok {
		t.Fatal("expected to find constructor for registered descriptor")
	}
	if constructor == nil {
		t.Fatal("expected non-nil constructor")
	}

	// Test for unregistered provider
	_, ok = nativeConstructorFor("nonexistent-native")
	if ok {
		t.Fatal("expected no constructor for unregistered native provider")
	}
}
