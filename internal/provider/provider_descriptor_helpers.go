package provider

func (d ProviderDescriptor) ToProviderConfig() ProviderConfig {
	config := ProviderConfig{
		Name:    d.Name,
		Type:    d.Type,
		Source:  d.Source,
		Config:  copyInterfaceMap(d.Config),
		Models:  copyStringMap(d.Models),
		Enabled: d.DefaultEnabled,
	}

	if d.EnableIfEnv != "" {
		config.Enabled = IsEnvVarSet(d.EnableIfEnv)
	}

	return config
}

func copyInterfaceMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
