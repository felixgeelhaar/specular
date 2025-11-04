package exec

import "time"

// Step represents a single execution step
type Step struct {
	ID      string
	Runner  string   // "docker" or "local"
	Image   string   // Docker image name
	Cmd     []string // Command and arguments
	Workdir string   // Working directory path
	Env     map[string]string
	Network string // Network mode
	CPU     string // CPU limit
	Mem     string // Memory limit
}

// Result represents the outcome of an execution step
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
	Error    error
}

// RunManifest represents the audit log for a run
type RunManifest struct {
	Timestamp    time.Time         `json:"timestamp"`
	StepID       string            `json:"step_id"`
	Runner       string            `json:"runner"`
	Image        string            `json:"image,omitempty"`
	Command      []string          `json:"command"`
	Env          map[string]string `json:"env,omitempty"`
	ExitCode     int               `json:"exit_code"`
	Duration     string            `json:"duration"`
	InputHashes  map[string]string `json:"input_hashes"`
	OutputHashes map[string]string `json:"output_hashes"`
}
