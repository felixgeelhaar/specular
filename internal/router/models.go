package router

// GetAvailableModels returns the catalog of known AI models
func GetAvailableModels() []Model {
	return []Model{
		// Anthropic Claude Models
		{
			ID:              "claude-sonnet-4",
			Provider:        ProviderAnthropic,
			Name:            "claude-sonnet-4-20250514",
			Type:            ModelTypeAgentic,
			ContextWindow:   200000,
			CostPerMToken:   3.00,  // $3 per million tokens (input)
			MaxLatencyMs:    5000,
			CapabilityScore: 95,
			Available:       true,
		},
		{
			ID:              "claude-sonnet-3.5",
			Provider:        ProviderAnthropic,
			Name:            "claude-3-5-sonnet-20241022",
			Type:            ModelTypeCodegen,
			ContextWindow:   200000,
			CostPerMToken:   3.00,
			MaxLatencyMs:    4000,
			CapabilityScore: 92,
			Available:       true,
		},
		{
			ID:              "claude-haiku-3.5",
			Provider:        ProviderAnthropic,
			Name:            "claude-3-5-haiku-20241022",
			Type:            ModelTypeFast,
			ContextWindow:   200000,
			CostPerMToken:   0.80,  // $0.80 per million tokens
			MaxLatencyMs:    2000,
			CapabilityScore: 75,
			Available:       true,
		},

		// OpenAI Models
		{
			ID:              "gpt-4-turbo",
			Provider:        ProviderOpenAI,
			Name:            "gpt-4-turbo-2024-04-09",
			Type:            ModelTypeLongContext,
			ContextWindow:   128000,
			CostPerMToken:   10.00,  // $10 per million tokens
			MaxLatencyMs:    6000,
			CapabilityScore: 90,
			Available:       true,
		},
		{
			ID:              "gpt-4o",
			Provider:        ProviderOpenAI,
			Name:            "gpt-4o-2024-08-06",
			Type:            ModelTypeCodegen,
			ContextWindow:   128000,
			CostPerMToken:   2.50,  // $2.50 per million tokens
			MaxLatencyMs:    4000,
			CapabilityScore: 88,
			Available:       true,
		},
		{
			ID:              "gpt-4o-mini",
			Provider:        ProviderOpenAI,
			Name:            "gpt-4o-mini-2024-07-18",
			Type:            ModelTypeCheap,
			ContextWindow:   128000,
			CostPerMToken:   0.15,  // $0.15 per million tokens
			MaxLatencyMs:    2000,
			CapabilityScore: 70,
			Available:       true,
		},
		{
			ID:              "gpt-3.5-turbo",
			Provider:        ProviderOpenAI,
			Name:            "gpt-3.5-turbo-0125",
			Type:            ModelTypeFast,
			ContextWindow:   16385,
			CostPerMToken:   0.50,  // $0.50 per million tokens
			MaxLatencyMs:    1500,
			CapabilityScore: 65,
			Available:       true,
		},
	}
}

// GetModelByID finds a model by its ID
func GetModelByID(id string) *Model {
	models := GetAvailableModels()
	for _, m := range models {
		if m.ID == id {
			return &m
		}
	}
	return nil
}

// GetModelsByType returns all models of a specific type
func GetModelsByType(modelType ModelType) []Model {
	models := GetAvailableModels()
	var result []Model
	for _, m := range models {
		if m.Type == modelType {
			result = append(result, m)
		}
	}
	return result
}

// GetModelsByProvider returns all models for a provider
func GetModelsByProvider(provider Provider) []Model {
	models := GetAvailableModels()
	var result []Model
	for _, m := range models {
		if m.Provider == provider {
			result = append(result, m)
		}
	}
	return result
}

// GetCheapestModel returns the cheapest available model
func GetCheapestModel() *Model {
	models := GetAvailableModels()
	if len(models) == 0 {
		return nil
	}

	cheapest := &models[0]
	for i := range models {
		if models[i].Available && models[i].CostPerMToken < cheapest.CostPerMToken {
			cheapest = &models[i]
		}
	}
	return cheapest
}

// GetFastestModel returns the fastest available model
func GetFastestModel() *Model {
	models := GetAvailableModels()
	if len(models) == 0 {
		return nil
	}

	fastest := &models[0]
	for i := range models {
		if models[i].Available && models[i].MaxLatencyMs < fastest.MaxLatencyMs {
			fastest = &models[i]
		}
	}
	return fastest
}

// GetBestModel returns the highest capability model
func GetBestModel() *Model {
	models := GetAvailableModels()
	if len(models) == 0 {
		return nil
	}

	best := &models[0]
	for i := range models {
		if models[i].Available && models[i].CapabilityScore > best.CapabilityScore {
			best = &models[i]
		}
	}
	return best
}
