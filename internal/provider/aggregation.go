package provider

import "github.com/felixgeelhaar/specular/internal/detect"

// AggregatesFromDetection converts a detect.Context into provider aggregates.
func AggregatesFromDetection(ctx *detect.Context) []*ProviderAggregate {
	if ctx == nil {
		return nil
	}

	var aggregates []*ProviderAggregate
	for _, desc := range Descriptors() {
		agg := NewProviderAggregate(desc)
		if info, ok := ctx.Providers[desc.Name]; ok {
			agg.Status.Available = info.Available
			agg.Status.VisibleReason = info.EnvVar
			if agg.Status.VisibleReason == "" {
				agg.Status.VisibleReason = desc.Description
			}
		}
		aggregates = append(aggregates, agg)
	}

	return aggregates
}
