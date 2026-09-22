package policy

// Policy represents the complete policy configuration
type Policy struct {
	Execution  ExecutionPolicy       `yaml:"execution"`
	Linters    map[string]ToolConfig `yaml:"linters"`
	Formatters map[string]ToolConfig `yaml:"formatters"`
	Tests      TestPolicy            `yaml:"tests"`
	Security   SecurityPolicy        `yaml:"security"`
	Routing    RoutingPolicy         `yaml:"routing"`
	// Risk is opt-in risk-adaptive governance (PRODUCT_INTENT §13).
	// When unset, gate Risk stays advisory and never flips ALLOW→DENY.
	Risk *RiskGovernance `yaml:"risk,omitempty"`
	// Provenance is opt-in Agent Provenance Protocol enforcement (§9).
	// When unset, APP doc counts stay advisory and never flip ALLOW→DENY.
	Provenance *ProvenanceGovernance `yaml:"provenance,omitempty"`
}

// RiskGovernance maps change-risk levels to approval requirements.
// Keys match gate advisory levels: low, medium, high, critical (case-insensitive).
type RiskGovernance struct {
	Low      *RiskTier `yaml:"low,omitempty"`
	Medium   *RiskTier `yaml:"medium,omitempty"`
	High     *RiskTier `yaml:"high,omitempty"`
	Critical *RiskTier `yaml:"critical,omitempty"`
}

// RiskTier is one risk level's governance requirements.
// PRODUCT_INTENT allows either `approval:` (string or list) or `approvals:`.
type RiskTier struct {
	Approval            StringList `yaml:"approval,omitempty"`
	Approvals           []string   `yaml:"approvals,omitempty"`
	AutonomousExecution *bool      `yaml:"autonomous_execution,omitempty"`
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
	User           string   `yaml:"user"`    // non-root container user; empty defaults to nobody
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
