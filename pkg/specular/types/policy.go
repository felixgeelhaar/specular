package types

// Policy represents the complete policy configuration
type Policy struct {
	Execution  ExecutionPolicy       `yaml:"execution"`
	Linters    map[string]ToolConfig `yaml:"linters"`
	Formatters map[string]ToolConfig `yaml:"formatters"`
	Tests      TestPolicy            `yaml:"tests"`
	Security   SecurityPolicy        `yaml:"security"`
	Routing    RoutingPolicy         `yaml:"routing"`
}

// ExecutionPolicy defines execution constraints
type ExecutionPolicy struct {
	AllowLocal bool         `yaml:"allow_local"`
	Docker     DockerPolicy `yaml:"docker"`
}

// DockerPolicy defines Docker-specific constraints
type DockerPolicy struct {
	Required       bool     `yaml:"required"`
	ImageAllowlist []string `yaml:"image_allowlist"`
	CPULimit       string   `yaml:"cpu_limit"`
	MemLimit       string   `yaml:"mem_limit"`
	Network        string   `yaml:"network"` // none, allowlist profile, etc.
}

// ToolConfig defines configuration for a tool (linter, formatter, etc.)
type ToolConfig struct {
	Enabled bool   `yaml:"enabled"`
	Cmd     string `yaml:"cmd"`
}

// TestPolicy defines testing requirements
type TestPolicy struct {
	RequirePass bool    `yaml:"require_pass"`
	MinCoverage float64 `yaml:"min_coverage"`
}

// SecurityPolicy defines security scanning requirements
type SecurityPolicy struct {
	SecretsScan bool `yaml:"secrets_scan"`
	DepScan     bool `yaml:"dep_scan"`
}

// RoutingPolicy defines AI model routing constraints
type RoutingPolicy struct {
	AllowModels []ModelAllow `yaml:"allow_models"`
	DenyTools   []string     `yaml:"deny_tools"`
}

// ModelAllow defines allowed models per provider
type ModelAllow struct {
	Provider string   `yaml:"provider"`
	Names    []string `yaml:"names"`
}
