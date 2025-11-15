package telemetry

import "testing"

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ServiceName != "specular" {
		t.Errorf("ServiceName = %q, want %q", config.ServiceName, "specular")
	}

	if config.ServiceVersion != "dev" {
		t.Errorf("ServiceVersion = %q, want %q", config.ServiceVersion, "dev")
	}

	if config.Environment != "development" {
		t.Errorf("Environment = %q, want %q", config.Environment, "development")
	}

	if config.Enabled {
		t.Error("Enabled should be false by default")
	}

	if config.Endpoint != "" {
		t.Error("Endpoint should be empty by default")
	}

	if config.SampleRate != 1.0 {
		t.Errorf("SampleRate = %v, want 1.0", config.SampleRate)
	}
}

func TestDevelopmentConfig(t *testing.T) {
	config := DevelopmentConfig()

	if config.ServiceName != "specular" {
		t.Errorf("ServiceName = %q, want %q", config.ServiceName, "specular")
	}

	if config.ServiceVersion != "dev" {
		t.Errorf("ServiceVersion = %q, want %q", config.ServiceVersion, "dev")
	}

	if config.Environment != "development" {
		t.Errorf("Environment = %q, want %q", config.Environment, "development")
	}

	if !config.Enabled {
		t.Error("Enabled should be true for development")
	}

	if config.Endpoint != "" {
		t.Error("Endpoint should be empty for development (no export)")
	}

	if config.SampleRate != 1.0 {
		t.Errorf("SampleRate = %v, want 1.0", config.SampleRate)
	}
}

func TestProductionConfig(t *testing.T) {
	endpoint := "http://localhost:4318"
	config := ProductionConfig(endpoint)

	if config.ServiceName != "specular" {
		t.Errorf("ServiceName = %q, want %q", config.ServiceName, "specular")
	}

	if config.ServiceVersion != "unknown" {
		t.Errorf("ServiceVersion = %q, want %q", config.ServiceVersion, "unknown")
	}

	if config.Environment != "production" {
		t.Errorf("Environment = %q, want %q", config.Environment, "production")
	}

	if !config.Enabled {
		t.Error("Enabled should be true for production")
	}

	if config.Endpoint != endpoint {
		t.Errorf("Endpoint = %q, want %q", config.Endpoint, endpoint)
	}

	if config.SampleRate != 0.1 {
		t.Errorf("SampleRate = %v, want 0.1", config.SampleRate)
	}
}

func TestConfigDefaults(t *testing.T) {
	configs := []struct {
		name   string
		config Config
	}{
		{"default", DefaultConfig()},
		{"development", DevelopmentConfig()},
		{"production", ProductionConfig("http://localhost:4318")},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.config.ServiceName == "" {
				t.Error("ServiceName should not be empty")
			}

			if tc.config.ServiceVersion == "" {
				t.Error("ServiceVersion should not be empty")
			}

			if tc.config.Environment == "" {
				t.Error("Environment should not be empty")
			}

			if tc.config.SampleRate < 0 || tc.config.SampleRate > 1.0 {
				t.Errorf("SampleRate = %v, should be between 0 and 1.0", tc.config.SampleRate)
			}
		})
	}
}
