package nativeprovider

import "github.com/felixgeelhaar/specular/internal/provider"

// Constructor creates a ProviderClient from a provider config.
type Constructor func(config *provider.ProviderConfig) (provider.ProviderClient, error)

var constructorRegistry = make(map[string]Constructor)

// Register makes a constructor available under the given provider name.
func Register(name string, constructor Constructor) {
	constructorRegistry[name] = constructor
}

// Lookup finds a registered constructor by provider name.
func Lookup(name string) (Constructor, bool) {
	constructor, ok := constructorRegistry[name]
	return constructor, ok
}
