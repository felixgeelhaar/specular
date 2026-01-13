package provider

// ProviderDescriptor holds metadata used to seed provider configuration files.
type ProviderDetectionHints struct {
	Binaries []string
	EnvVars  []string
}

type ProviderDescriptor struct {
	Name           string
	Type           ProviderType
	Source         string
	TrustLevel     TrustLevel
	Description    string
	Capabilities   []string
	DefaultEnabled bool
	EnableIfEnv    string
	Config         map[string]interface{}
	Models         map[string]string
	Hints          ProviderDetectionHints
	Constructor    NativeConstructor
}
