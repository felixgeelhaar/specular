package provider

func snapshotDescriptorState() (map[string]*ProviderDescriptor, []string) {
	descriptorMu.RLock()
	defer descriptorMu.RUnlock()

	storeCopy := make(map[string]*ProviderDescriptor, len(descriptorStore))
	for name, desc := range descriptorStore {
		copied := *desc
		storeCopy[name] = &copied
	}

	orderCopy := append([]string(nil), descriptorOrder...)
	return storeCopy, orderCopy
}

func restoreDescriptorState(store map[string]*ProviderDescriptor, order []string) {
	descriptorMu.Lock()
	defer descriptorMu.Unlock()
	descriptorStore = store
	descriptorOrder = append([]string(nil), order...)
}

func clearDescriptorRegistry() {
	descriptorMu.Lock()
	defer descriptorMu.Unlock()
	descriptorStore = make(map[string]*ProviderDescriptor)
	descriptorOrder = nil
}
