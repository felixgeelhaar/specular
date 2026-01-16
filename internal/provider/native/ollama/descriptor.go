package ollama

import "github.com/felixgeelhaar/specular/internal/provider"

func init() {
	provider.RegisterNativeProvider("ollama", newOllamaProvider)

	provider.RegisterProviderDescriptor(provider.ProviderDescriptor{
		Name:           "ollama",
		Type:           provider.ProviderTypeNative,
		Source:         "builtin",
		TrustLevel:     provider.TrustLevelBuiltin,
		Description:    "Local Ollama models via HTTP API",
		DefaultEnabled: true,
		Config: map[string]interface{}{
			"base_url": "http://localhost:11434",
			"model":    "llama3.2",
		},
		Models: map[string]string{
			"fast":         "llama3.2",
			"codegen":      "llama3.2",
			"cheap":        "llama3.2",
			"long-context": "llama3.2",
		},
		Constructor: newOllamaProvider,
	})
}
