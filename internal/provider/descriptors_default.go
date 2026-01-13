package provider

func init() {
	registerDefaultDescriptors()
}

func registerDefaultDescriptors() {
	RegisterProviderDescriptor(ProviderDescriptor{
		Name:           "claude-code",
		Type:           ProviderTypeCLI,
		Source:         "local",
		TrustLevel:     TrustLevelVerified,
		Description:    "Claude Code CLI wrapper",
		Capabilities:   []string{"chat", "codegen"},
		DefaultEnabled: true,
		Config: map[string]interface{}{
			"path": "./providers/claude-code/claude-code-provider",
		},
		Hints: ProviderDetectionHints{
			Binaries: []string{"claude"},
		},
		Models: map[string]string{
			"fast":         "claude-haiku-4-5-20251015",
			"codegen":      "claude-sonnet-4-5-20250929",
			"agentic":      "claude-opus-4-1-20250805",
			"long-context": "claude-sonnet-4-5-20250929",
			"cheap":        "claude-haiku-4-5-20251015",
		},
	})

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:           "gemini-cli",
		Type:           ProviderTypeCLI,
		Source:         "local",
		TrustLevel:     TrustLevelCommunity,
		Description:    "Gemini CLI wrapper",
		Capabilities:   []string{"chat", "codegen"},
		DefaultEnabled: true,
		Config: map[string]interface{}{
			"path": "./providers/gemini-cli/gemini-cli-provider",
		},
		Hints: ProviderDetectionHints{
			Binaries: []string{"gemini"},
		},
		Models: map[string]string{
			"fast":         "gemini-2.0-flash-exp",
			"codegen":      "gemini-exp-1206",
			"agentic":      "gemini-exp-1206",
			"long-context": "gemini-exp-1206",
			"cheap":        "gemini-2.0-flash-exp",
		},
	})

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:           "codex-cli",
		Type:           ProviderTypeCLI,
		Source:         "local",
		TrustLevel:     TrustLevelCommunity,
		Description:    "Codex CLI wrapper",
		Capabilities:   []string{"codegen"},
		DefaultEnabled: true,
		Config: map[string]interface{}{
			"path": "./providers/codex-cli/codex-cli-provider",
		},
		Hints: ProviderDetectionHints{
			Binaries: []string{"codex"},
		},
		Models: map[string]string{
			"fast":         "codex",
			"codegen":      "codex",
			"agentic":      "codex",
			"long-context": "codex",
			"cheap":        "codex",
		},
	})

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:         "anthropic",
		Type:         ProviderTypeAPI,
		Source:       "builtin",
		TrustLevel:   TrustLevelBuiltin,
		Description:  "Anthropic Claude API",
		Constructor: func(config *ProviderConfig) (ProviderClient, error) {
			return NewAnthropicProvider(config)
		},
		Capabilities: []string{"chat"},
		Config: map[string]interface{}{
			"api_key":  "${ANTHROPIC_API_KEY}",
			"base_url": "https://api.anthropic.com/v1",
		},
		EnableIfEnv: "ANTHROPIC_API_KEY",
		Hints: ProviderDetectionHints{
			EnvVars: []string{"ANTHROPIC_API_KEY"},
		},
		Models: map[string]string{
			"fast":         "claude-haiku-4-5-20251015",
			"codegen":      "claude-sonnet-4-5-20250929",
			"agentic":      "claude-opus-4-1-20250805",
			"long-context": "claude-sonnet-4-5-20250929",
			"cheap":        "claude-haiku-4-5-20251015",
		},
	})

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:         "openai",
		Type:         ProviderTypeAPI,
		Source:       "builtin",
		TrustLevel:   TrustLevelBuiltin,
		Description:  "OpenAI API",
		Constructor: func(config *ProviderConfig) (ProviderClient, error) {
			return NewOpenAIProvider(config)
		},
		Capabilities: []string{"chat", "codegen"},
		Config: map[string]interface{}{
			"api_key":  "${OPENAI_API_KEY}",
			"base_url": "https://api.openai.com/v1",
		},
		EnableIfEnv: "OPENAI_API_KEY",
		Models: map[string]string{
			"fast":         "gpt-5-mini",
			"codegen":      "gpt-5",
			"agentic":      "gpt-5",
			"long-context": "gpt-5",
			"cheap":        "gpt-5-nano",
		},
	})

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:         "gemini",
		Type:         ProviderTypeAPI,
		Source:       "builtin",
		TrustLevel:   TrustLevelBuiltin,
		Description:  "Google Gemini API",
		Constructor: func(config *ProviderConfig) (ProviderClient, error) {
			return NewGeminiProvider(config)
		},
		Capabilities: []string{"chat", "codegen"},
		Config: map[string]interface{}{
			"api_key":  "${GEMINI_API_KEY}",
			"base_url": "https://generativelanguage.googleapis.com/v1beta",
		},
		EnableIfEnv: "GEMINI_API_KEY",
		Models: map[string]string{
			"fast":         "gemini-2.0-flash-exp",
			"codegen":      "gemini-2.0-flash-exp",
			"agentic":      "gemini-2.5-pro-exp-03",
			"long-context": "gemini-2.5-pro-exp-03",
		},
	})
}
