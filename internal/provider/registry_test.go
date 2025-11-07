package provider

import (
	"context"
	"fmt"
	"testing"
)

func TestLoadFromConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *ProviderConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "empty provider name",
			config: &ProviderConfig{
				Name:    "",
				Type:    ProviderTypeAPI,
				Enabled: true,
			},
			wantErr:     true,
			errContains: "provider name is required",
		},
		{
			name: "disabled provider",
			config: &ProviderConfig{
				Name:    "test-provider",
				Type:    ProviderTypeAPI,
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "CLI provider missing path",
			config: &ProviderConfig{
				Name:    "test-cli",
				Type:    ProviderTypeCLI,
				Enabled: true,
				Config:  map[string]interface{}{},
			},
			wantErr:     true,
			errContains: "executable path required",
		},
		{
			name: "CLI provider with invalid path type",
			config: &ProviderConfig{
				Name:    "test-cli",
				Type:    ProviderTypeCLI,
				Enabled: true,
				Config: map[string]interface{}{
					"path": 123, // invalid type
				},
			},
			wantErr:     true,
			errContains: "executable path required",
		},
		{
			name: "unknown API provider",
			config: &ProviderConfig{
				Name:    "unknown-api",
				Type:    ProviderTypeAPI,
				Enabled: true,
			},
			wantErr:     true,
			errContains: "unknown API provider",
		},
		{
			name: "gRPC provider not implemented",
			config: &ProviderConfig{
				Name:    "test-grpc",
				Type:    ProviderTypeGRPC,
				Enabled: true,
			},
			wantErr:     true,
			errContains: "gRPC providers not yet implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			err := registry.LoadFromConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadFromConfig() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadFromConfig() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("LoadFromConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestRegistry_GetConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig *ProviderConfig
		queryName   string
		wantErr     bool
		errContains string
	}{
		{
			name: "get existing config",
			setupConfig: &ProviderConfig{
				Name: "test-provider",
				Type: ProviderTypeAPI,
			},
			queryName: "test-provider",
			wantErr:   false,
		},
		{
			name:        "get non-existent config",
			setupConfig: nil,
			queryName:   "non-existent",
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()

			// Setup: register provider config if provided
			if tt.setupConfig != nil {
				// Use Register directly to bypass provider creation
				err := registry.Register(tt.setupConfig.Name, nil, tt.setupConfig)
				if err != nil {
					t.Fatalf("Setup failed: Register() error = %v", err)
				}
			}

			// Test GetConfig
			config, err := registry.GetConfig(tt.queryName)

			if tt.wantErr {
				if err == nil {
					t.Error("GetConfig() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("GetConfig() error = %v, want error containing %q", err, tt.errContains)
				}
				if config != nil {
					t.Errorf("GetConfig() returned config %v, want nil", config)
				}
			} else {
				if err != nil {
					t.Errorf("GetConfig() unexpected error = %v", err)
				}
				if config == nil {
					t.Error("GetConfig() returned nil config")
				} else if config.Name != tt.queryName {
					t.Errorf("GetConfig() config name = %v, want %v", config.Name, tt.queryName)
				}
			}
		})
	}
}

func TestRegistry_Remove(t *testing.T) {
	tests := []struct {
		name        string
		setupName   string
		removeName  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "remove non-existent provider",
			setupName:  "",
			removeName: "non-existent",
			wantErr:    true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()

			// Setup: register provider if specified
			if tt.setupName != "" {
				config := &ProviderConfig{
					Name: tt.setupName,
					Type: ProviderTypeAPI,
				}
				err := registry.Register(tt.setupName, nil, config)
				if err != nil {
					t.Fatalf("Setup failed: Register() error = %v", err)
				}
			}

			// Test Remove
			err := registry.Remove(tt.removeName)

			if tt.wantErr {
				if err == nil {
					t.Error("Remove() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Remove() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Remove() unexpected error = %v", err)
				}

				// Verify provider was removed
				_, err = registry.Get(tt.removeName)
				if err == nil {
					t.Error("Remove() provider still exists after removal")
				}
			}
		})
	}
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name        string
		firstReg    string
		secondReg   string
		wantErr     bool
		errContains string
	}{
		{
			name:        "duplicate registration",
			firstReg:    "test-provider",
			secondReg:   "test-provider",
			wantErr:     true,
			errContains: "already registered",
		},
		{
			name:      "different providers",
			firstReg:  "provider-1",
			secondReg: "provider-2",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()

			// Register first provider
			config1 := &ProviderConfig{
				Name: tt.firstReg,
				Type: ProviderTypeAPI,
			}
			err := registry.Register(tt.firstReg, nil, config1)
			if err != nil {
				t.Fatalf("First Register() error = %v", err)
			}

			// Register second provider
			config2 := &ProviderConfig{
				Name: tt.secondReg,
				Type: ProviderTypeAPI,
			}
			err = registry.Register(tt.secondReg, nil, config2)

			if tt.wantErr {
				if err == nil {
					t.Error("Register() expected error for duplicate, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Register() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Register() unexpected error = %v", err)
				}
			}
		})
	}
}


// mockProvider is a minimal mock implementation for testing
type mockProvider struct{}

func (m *mockProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Content: "test"}, nil
}

func (m *mockProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockProvider) GetCapabilities() *ProviderCapabilities {
	return &ProviderCapabilities{}
}

func (m *mockProvider) GetInfo() *ProviderInfo {
	return &ProviderInfo{Name: "mock"}
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

func TestRegistry_CloseAll(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		registry := NewRegistry()
		err := registry.CloseAll()
		if err != nil {
			t.Errorf("CloseAll() on empty registry unexpected error = %v", err)
		}
	})

	t.Run("with providers", func(t *testing.T) {
		registry := NewRegistry()

		// Register multiple providers
		for i := 1; i <= 3; i++ {
			config := &ProviderConfig{
				Name: fmt.Sprintf("provider-%d", i),
				Type: ProviderTypeAPI,
			}
			err := registry.Register(config.Name, &mockProvider{}, config)
			if err != nil {
				t.Fatalf("Register() error = %v", err)
			}
		}

		// Verify providers exist
		if len(registry.List()) != 3 {
			t.Errorf("Registry has %d providers, want 3", len(registry.List()))
		}

		// Close all
		err := registry.CloseAll()
		if err != nil {
			t.Errorf("CloseAll() unexpected error = %v", err)
		}

		// Verify registry is empty
		if len(registry.List()) != 0 {
			t.Errorf("Registry has %d providers after CloseAll, want 0", len(registry.List()))
		}
	})
}


// mockProviderWithCloseError is a mock provider that fails to close
type mockProviderWithCloseError struct {
	mockProvider
}

func (m *mockProviderWithCloseError) Close() error {
	return fmt.Errorf("close failed")
}

func TestRegistry_CloseAll_WithErrors(t *testing.T) {
	registry := NewRegistry()

	// Register a provider that will fail to close
	config := &ProviderConfig{
		Name: "failing-provider",
		Type: ProviderTypeAPI,
	}
	err := registry.Register(config.Name, &mockProviderWithCloseError{}, config)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Close all should return an error
	err = registry.CloseAll()
	if err == nil {
		t.Error("CloseAll() expected error when provider Close fails, got nil")
	}
	if !contains(err.Error(), "close failed") {
		t.Errorf("CloseAll() error = %v, want error containing 'close failed'", err)
	}

	// Registry should still be empty after CloseAll (even with errors)
	if len(registry.List()) != 0 {
		t.Errorf("Registry has %d providers after CloseAll, want 0", len(registry.List()))
	}
}
