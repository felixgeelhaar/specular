package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadPolicy reads a Policy from a YAML file
func LoadPolicy(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy file: %w", err)
	}

	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("unmarshal policy: %w", err)
	}

	return &policy, nil
}

// DefaultPolicy returns a policy with sensible defaults
func DefaultPolicy() *Policy {
	return &Policy{
		Execution: ExecutionPolicy{
			AllowLocal: false,
			Docker: DockerPolicy{
				Required:       true,
				ImageAllowlist: []string{},
				CPULimit:       "2",
				MemLimit:       "2g",
				Network:        "none",
			},
		},
		Linters:    make(map[string]ToolConfig),
		Formatters: make(map[string]ToolConfig),
		Tests: TestPolicy{
			RequirePass: true,
			MinCoverage: 0.70,
		},
		Security: SecurityPolicy{
			SecretsScan: true,
			DepScan:     true,
		},
		Routing: RoutingPolicy{
			AllowModels: []ModelAllow{},
			DenyTools:   []string{},
		},
	}
}

// SavePolicy writes a Policy to a YAML file
func SavePolicy(policy *Policy, path string) error {
	data, err := yaml.Marshal(policy)
	if err != nil {
		return fmt.Errorf("marshal policy: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write policy file: %w", err)
	}

	return nil
}

// ValidateToolConfig checks if a tool configuration is valid
func ValidateToolConfig(config ToolConfig) error {
	if config.Enabled && config.Cmd == "" {
		return fmt.Errorf("enabled tool must have a command configured")
	}
	return nil
}
