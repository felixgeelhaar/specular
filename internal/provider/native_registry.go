package provider

import "sync"

// NativeConstructor builds a ProviderClient from a provider config.
type NativeConstructor func(config *ProviderConfig) (ProviderClient, error)

var (
	nativeConstructorMu sync.RWMutex
	nativeConstructors  = make(map[string]NativeConstructor)
)

// RegisterNativeProvider registers a native provider constructor.
func RegisterNativeProvider(name string, constructor NativeConstructor) {
	nativeConstructorMu.Lock()
	defer nativeConstructorMu.Unlock()
	nativeConstructors[name] = constructor
}

// lookupNativeConstructor retrieves a registered native provider constructor.
func lookupNativeConstructor(name string) (NativeConstructor, bool) {
	nativeConstructorMu.RLock()
	defer nativeConstructorMu.RUnlock()
	constructor, ok := nativeConstructors[name]
	return constructor, ok
}
