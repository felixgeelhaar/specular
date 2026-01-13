package provider

import "sync"

var (
	descriptorMu    sync.RWMutex
	descriptorStore = make(map[string]*ProviderDescriptor)
	descriptorOrder []string
)

// RegisterProviderDescriptor registers or updates a provider descriptor.
func RegisterProviderDescriptor(desc ProviderDescriptor) {
	descriptorMu.Lock()
	defer descriptorMu.Unlock()

	if _, exists := descriptorStore[desc.Name]; !exists {
		descriptorOrder = append(descriptorOrder, desc.Name)
	}
	copied := desc
	descriptorStore[desc.Name] = &copied
}

// Descriptors returns a copy of the registered descriptors in registration order.
func Descriptors() []ProviderDescriptor {
	descriptorMu.RLock()
	defer descriptorMu.RUnlock()

	result := make([]ProviderDescriptor, 0, len(descriptorOrder))
	for _, name := range descriptorOrder {
		if desc, ok := descriptorStore[name]; ok {
			result = append(result, *desc)
		}
	}
	return result
}

// DescriptorByName returns the descriptor for the given provider, if registered.
func DescriptorByName(name string) *ProviderDescriptor {
	descriptorMu.RLock()
	defer descriptorMu.RUnlock()
	if desc, ok := descriptorStore[name]; ok {
		return desc
	}
	return nil
}
