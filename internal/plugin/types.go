// Package plugin provides the plugin system for extending Specular functionality.
package plugin

import (
	"time"
)

// PluginType represents the type of extension a plugin provides
type PluginType string

const (
	// PluginTypeProvider is an AI provider plugin
	PluginTypeProvider PluginType = "provider"
	// PluginTypeValidator is a policy validator plugin
	PluginTypeValidator PluginType = "validator"
	// PluginTypeFormatter is an output formatter plugin
	PluginTypeFormatter PluginType = "formatter"
	// PluginTypeHook is an event hook plugin
	PluginTypeHook PluginType = "hook"
	// PluginTypeNotifier is a notification plugin
	PluginTypeNotifier PluginType = "notifier"
)

// PluginState represents the current state of a plugin
type PluginState string

const (
	// PluginStateUnknown is the initial state
	PluginStateUnknown PluginState = "unknown"
	// PluginStateDiscovered means the plugin was found but not loaded
	PluginStateDiscovered PluginState = "discovered"
	// PluginStateLoaded means the plugin manifest was loaded
	PluginStateLoaded PluginState = "loaded"
	// PluginStateEnabled means the plugin is active
	PluginStateEnabled PluginState = "enabled"
	// PluginStateDisabled means the plugin is installed but not active
	PluginStateDisabled PluginState = "disabled"
	// PluginStateError means the plugin failed to load or initialize
	PluginStateError PluginState = "error"
)

// Manifest represents a plugin's metadata and configuration
type Manifest struct {
	// Name is the unique identifier for the plugin
	Name string `json:"name" yaml:"name"`
	// Version follows semver (e.g., "1.0.0")
	Version string `json:"version" yaml:"version"`
	// Description is a short description of the plugin
	Description string `json:"description" yaml:"description"`
	// Author is the plugin author's name or organization
	Author string `json:"author" yaml:"author"`
	// License is the SPDX license identifier (e.g., "MIT", "Apache-2.0")
	License string `json:"license" yaml:"license"`
	// Homepage is the URL to the plugin's homepage or repository
	Homepage string `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	// Type specifies what kind of extension this plugin provides
	Type PluginType `json:"type" yaml:"type"`
	// Entrypoint is the executable or script to run
	Entrypoint string `json:"entrypoint" yaml:"entrypoint"`
	// MinSpecularVersion is the minimum required Specular version
	MinSpecularVersion string `json:"min_specular_version,omitempty" yaml:"min_specular_version,omitempty"`
	// Config defines plugin-specific configuration schema
	Config []ConfigField `json:"config,omitempty" yaml:"config,omitempty"`
	// Capabilities lists the specific capabilities the plugin provides
	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// ConfigField defines a configuration field for a plugin
type ConfigField struct {
	// Name is the configuration key
	Name string `json:"name" yaml:"name"`
	// Type is the value type (string, int, bool, etc.)
	Type string `json:"type" yaml:"type"`
	// Description explains what this configuration does
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Required indicates if this field must be set
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`
	// Default is the default value if not specified
	Default interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	// Secret indicates this value should be treated as sensitive
	Secret bool `json:"secret,omitempty" yaml:"secret,omitempty"`
}

// Plugin represents a loaded plugin instance
type Plugin struct {
	// Manifest contains the plugin metadata
	Manifest Manifest `json:"manifest"`
	// Path is the filesystem path to the plugin
	Path string `json:"path"`
	// State is the current plugin state
	State PluginState `json:"state"`
	// Error contains any error message if State is Error
	Error string `json:"error,omitempty"`
	// LoadedAt is when the plugin was loaded
	LoadedAt time.Time `json:"loaded_at"`
	// Config holds the runtime configuration
	Config map[string]interface{} `json:"config,omitempty"`
}

// PluginRequest is sent to a plugin for execution
type PluginRequest struct {
	// Action is the operation to perform
	Action string `json:"action"`
	// Params contains action-specific parameters
	Params map[string]interface{} `json:"params,omitempty"`
	// Config is the plugin's configuration
	Config map[string]interface{} `json:"config,omitempty"`
}

// PluginResponse is returned from a plugin execution
type PluginResponse struct {
	// Success indicates if the action completed successfully
	Success bool `json:"success"`
	// Result contains the action's output
	Result interface{} `json:"result,omitempty"`
	// Error contains an error message if Success is false
	Error string `json:"error,omitempty"`
}

// HealthRequest is sent to check plugin health
type HealthRequest struct {
	Action string `json:"action"`
}

// HealthResponse is returned from a health check
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

// ValidatorRequest is sent to validator plugins
type ValidatorRequest struct {
	Action  string                 `json:"action"`
	Content string                 `json:"content"`
	Rules   map[string]interface{} `json:"rules,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// ValidatorResponse is returned from validator plugins
type ValidatorResponse struct {
	Valid    bool             `json:"valid"`
	Messages []ValidatorIssue `json:"messages,omitempty"`
	Error    string           `json:"error,omitempty"`
}

// ValidatorIssue represents a validation issue
type ValidatorIssue struct {
	Severity string `json:"severity"` // error, warning, info
	Message  string `json:"message"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Rule     string `json:"rule,omitempty"`
}

// NotifierRequest is sent to notifier plugins
type NotifierRequest struct {
	Action  string                 `json:"action"`
	Event   string                 `json:"event"`
	Data    map[string]interface{} `json:"data"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// NotifierResponse is returned from notifier plugins
type NotifierResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// FormatterRequest is sent to formatter plugins
type FormatterRequest struct {
	Action string                 `json:"action"`
	Data   interface{}            `json:"data"`
	Format string                 `json:"format"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// FormatterResponse is returned from formatter plugins
type FormatterResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}
