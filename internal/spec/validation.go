package spec

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/internal/domain"
)

// Validate checks if the Feature is valid according to domain rules
func (f *Feature) Validate() error {
	// Validate ID using domain FeatureID value object
	if _, err := domain.NewFeatureID(f.ID); err != nil {
		return fmt.Errorf("invalid feature ID: %w", err)
	}

	// Validate Title
	if strings.TrimSpace(f.Title) == "" {
		return fmt.Errorf("feature title cannot be empty")
	}

	// Validate Description
	if strings.TrimSpace(f.Desc) == "" {
		return fmt.Errorf("feature description cannot be empty")
	}

	// Validate Priority using domain Priority value object
	if _, err := domain.NewPriority(f.Priority); err != nil {
		return fmt.Errorf("invalid feature priority: %w", err)
	}

	// Validate Success criteria - must have at least one
	if len(f.Success) == 0 {
		return fmt.Errorf("feature must have at least one success criterion")
	}

	// Validate each success criterion is non-empty
	for i, criterion := range f.Success {
		if strings.TrimSpace(criterion) == "" {
			return fmt.Errorf("success criterion at index %d cannot be empty", i)
		}
	}

	// Validate API endpoints if present
	for i, api := range f.API {
		if err := api.Validate(); err != nil {
			return fmt.Errorf("API endpoint at index %d is invalid: %w", i, err)
		}
	}

	// Trace is optional, but validate non-empty if present
	for i, trace := range f.Trace {
		if strings.TrimSpace(trace) == "" {
			return fmt.Errorf("trace at index %d cannot be empty", i)
		}
	}

	return nil
}

// Validate checks if the API endpoint is valid
func (a *API) Validate() error {
	// Validate HTTP method
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"PATCH":   true,
		"DELETE":  true,
		"HEAD":    true,
		"OPTIONS": true,
	}

	method := strings.ToUpper(strings.TrimSpace(a.Method))
	if method == "" {
		return fmt.Errorf("API method cannot be empty")
	}

	if !validMethods[method] {
		return fmt.Errorf("API method %q is not a valid HTTP method (must be GET, POST, PUT, PATCH, DELETE, HEAD, or OPTIONS)", a.Method)
	}

	// Validate path
	path := strings.TrimSpace(a.Path)
	if path == "" {
		return fmt.Errorf("API path cannot be empty")
	}

	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("API path %q must start with /", a.Path)
	}

	// Request and Response are optional, no validation needed

	return nil
}

// Validate checks if the ProductSpec is valid
func (p *ProductSpec) Validate() error {
	// Validate Product name
	if strings.TrimSpace(p.Product) == "" {
		return fmt.Errorf("product name cannot be empty")
	}

	// Validate Goals - must have at least one
	if len(p.Goals) == 0 {
		return fmt.Errorf("product must have at least one goal")
	}

	// Validate each goal is non-empty
	for i, goal := range p.Goals {
		if strings.TrimSpace(goal) == "" {
			return fmt.Errorf("goal at index %d cannot be empty", i)
		}
	}

	// Validate Features - must have at least one
	if len(p.Features) == 0 {
		return fmt.Errorf("product must have at least one feature")
	}

	// Validate each feature
	for i, feature := range p.Features {
		if err := feature.Validate(); err != nil {
			return fmt.Errorf("feature at index %d (%s) is invalid: %w", i, feature.ID, err)
		}
	}

	// Validate Acceptance criteria - must have at least one
	if len(p.Acceptance) == 0 {
		return fmt.Errorf("product must have at least one acceptance criterion")
	}

	// Validate each acceptance criterion is non-empty
	for i, criterion := range p.Acceptance {
		if strings.TrimSpace(criterion) == "" {
			return fmt.Errorf("acceptance criterion at index %d cannot be empty", i)
		}
	}

	// Validate Milestones if present
	for i, milestone := range p.Milestones {
		if err := milestone.Validate(); err != nil {
			return fmt.Errorf("milestone at index %d (%s) is invalid: %w", i, milestone.ID, err)
		}
	}

	return nil
}

// Validate checks if the Milestone is valid
func (m *Milestone) Validate() error {
	// Validate ID - must be non-empty and follow naming rules
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return fmt.Errorf("milestone ID cannot be empty")
	}

	// Validate Name
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("milestone name cannot be empty")
	}

	// Validate FeatureIDs - must have at least one
	if len(m.FeatureIDs) == 0 {
		return fmt.Errorf("milestone must reference at least one feature")
	}

	// Validate each feature ID
	for i, featureID := range m.FeatureIDs {
		if _, err := domain.NewFeatureID(featureID); err != nil {
			return fmt.Errorf("feature ID at index %d is invalid: %w", i, err)
		}
	}

	// TargetDate and Description are optional, no validation needed

	return nil
}
