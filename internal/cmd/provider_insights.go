package cmd

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/internal/detect"
	"github.com/felixgeelhaar/specular/internal/docgen"
	"github.com/felixgeelhaar/specular/internal/provider"
)

func providerDescriptorDetail(name string) string {
	desc := provider.DescriptorByName(name)
	if desc == nil {
		return ""
	}

	var attrs []string
	if desc.Source != "" {
		attrs = append(attrs, fmt.Sprintf("Source: %s", desc.Source))
	}
	if desc.TrustLevel != "" {
		attrs = append(attrs, fmt.Sprintf("Trust: %s", desc.TrustLevel))
	}
	if desc.Description != "" {
		attrs = append(attrs, desc.Description)
	}
	if len(desc.Capabilities) > 0 {
		attrs = append(attrs, fmt.Sprintf("Capabilities: %s", strings.Join(desc.Capabilities, ", ")))
	}
	if hints := docgen.FormatDetectionHints(desc.Hints); hints != "" {
		attrs = append(attrs, hints)
	}

	if len(attrs) == 0 {
		return ""
	}

	return strings.Join(attrs, " | ")
}

func providerStrategyInsight(ctx *detect.Context) string {
	if ctx == nil {
		return ""
	}

	recommended := ctx.GetRecommendedProviders()
	if len(recommended) == 0 {
		return ""
	}

	var parts []string
	for _, name := range recommended {
		if desc := provider.DescriptorByName(name); desc != nil {
			parts = append(parts, fmt.Sprintf("%s (%s)", providerDisplayName(name), desc.TrustLevel))
			continue
		}
		parts = append(parts, providerDisplayName(name))
	}

	return fmt.Sprintf("Recommended providers: %s", strings.Join(parts, ", "))
}
