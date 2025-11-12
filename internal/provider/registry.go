package provider

import (
	"fmt"
	"sync"
)

// ProviderRegistry defines the interface for managing AI providers.
// This interface enables dependency injection and makes testing easier.
type ProviderRegistry interface {
	// Register adds a provider to the registry
	Register(name string, provider ProviderClient, config *ProviderConfig) error

	// Get retrieves a provider by name
	Get(name string) (ProviderClient, error)

	// GetConfig retrieves a provider's configuration
	GetConfig(name string) (*ProviderConfig, error)

	// List returns all registered provider names
	List() []string

	// Remove removes a provider from the registry and closes it
	Remove(name string) error

	// CloseAll closes all registered providers
	CloseAll() error

	// LoadFromConfig loads a provider from configuration
	LoadFromConfig(config *ProviderConfig) error
}

// Registry manages all loaded providers and implements ProviderRegistry interface
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ProviderClient
	configs   map[string]*ProviderConfig
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]ProviderClient),
		configs:   make(map[string]*ProviderConfig),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(name string, provider ProviderClient, config *ProviderConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider
	r.configs[name] = config

	return nil
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (ProviderClient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// GetConfig retrieves a provider's configuration
func (r *Registry) GetConfig(name string) (*ProviderConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.configs[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return config, nil
}

// List returns all registered provider names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}

// Remove removes a provider from the registry and closes it
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Close the provider
	if err := provider.Close(); err != nil {
		return fmt.Errorf("failed to close provider %s: %w", name, err)
	}

	delete(r.providers, name)
	delete(r.configs, name)

	return nil
}

// CloseAll closes all registered providers
func (r *Registry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for name, provider := range r.providers {
		if err := provider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close provider %s: %w", name, err))
		}
	}

	r.providers = make(map[string]ProviderClient)
	r.configs = make(map[string]*ProviderConfig)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %v", errs)
	}

	return nil
}

// LoadFromConfig loads a provider from configuration
func (r *Registry) LoadFromConfig(config *ProviderConfig) error {
	// Validate configuration
	if config.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if !config.Enabled {
		// Skip disabled providers
		return nil
	}

	// Create provider based on type
	var provider ProviderClient
	var err error

	switch config.Type {
	case ProviderTypeCLI:
		// All CLI providers use the generic ExecutableProvider
		// which expects executables that implement generate/stream/health commands
		path, ok := config.Config["path"].(string)
		if !ok || path == "" {
			return fmt.Errorf("executable path required for CLI provider %s", config.Name)
		}
		provider, err = NewExecutableProvider(path, config)

	case ProviderTypeAPI:
		// Determine which API provider to create based on name
		switch config.Name {
		case "openai":
			provider, err = NewOpenAIProvider(config)
		case "anthropic":
			provider, err = NewAnthropicProvider(config)
		case "gemini":
			provider, err = NewGeminiProvider(config)
		default:
			return fmt.Errorf("unknown API provider: %s", config.Name)
		}

	case ProviderTypeGRPC:
		return fmt.Errorf("gRPC providers not yet implemented")

	case ProviderTypeNative:
		return fmt.Errorf("native Go plugins not yet implemented")

	default:
		return fmt.Errorf("unknown provider type: %s", config.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create provider %s: %w", config.Name, err)
	}

	// Register the provider
	return r.Register(config.Name, provider, config)
}

// Compile-time verification that Registry implements ProviderRegistry
var _ ProviderRegistry = (*Registry)(nil)
