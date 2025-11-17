package interview

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// generateSpec converts interview answers to a ProductSpec
func (e *Engine) generateSpec() (*spec.ProductSpec, error) {
	// Extract common fields
	productName := e.getAnswer("product-name", "service-name", "pipeline-name")
	if productName == "" {
		return nil, fmt.Errorf("product name not found in answers")
	}

	// Build goals
	goals := e.buildGoals()

	// Build features
	features := e.buildFeatures()

	// Build non-functional requirements
	nonFunctional := e.buildNonFunctional()

	// Build acceptance criteria
	acceptance := e.getAnswerList("success-criteria")

	// Build milestones
	milestones := e.buildMilestones()

	return &spec.ProductSpec{
		Product:       productName,
		Goals:         goals,
		Features:      features,
		NonFunctional: nonFunctional,
		Acceptance:    acceptance,
		Milestones:    milestones,
	}, nil
}

// getAnswer retrieves the first non-empty answer from a list of question IDs
func (e *Engine) getAnswer(questionIDs ...string) string {
	for _, id := range questionIDs {
		if answer, ok := e.session.Answers[id]; ok {
			if answer.Value != "" {
				return answer.Value
			}
		}
	}
	return ""
}

// getAnswerList retrieves a list answer
func (e *Engine) getAnswerList(questionID string) []string {
	if answer, ok := e.session.Answers[questionID]; ok {
		if len(answer.Values) > 0 {
			return answer.Values
		}
		if answer.Value != "" {
			return []string{answer.Value}
		}
	}
	return []string{}
}

// buildGoals constructs product goals from answers
func (e *Engine) buildGoals() []string {
	goals := []string{}

	// Add purpose as primary goal
	purpose := e.getAnswer("product-purpose", "api-purpose", "tool-purpose", "service-responsibility", "pipeline-purpose")
	if purpose != "" {
		goals = append(goals, purpose)
	}

	// Add target users as a goal
	targetUsers := e.getAnswer("target-users")
	if targetUsers != "" {
		goals = append(goals, fmt.Sprintf("Serve %s with an intuitive interface", targetUsers))
	}

	// Add performance goals if specified
	perf := e.getAnswer("performance-requirements", "expected-throughput", "data-volume")
	if perf != "" && strings.ToLower(perf) != "none" && strings.ToLower(perf) != "unknown" {
		goals = append(goals, fmt.Sprintf("Achieve performance targets: %s", perf))
	}

	if len(goals) == 0 {
		goals = append(goals, "Deliver a functional MVP")
	}

	return goals
}

// buildFeatures constructs feature list from answers
func (e *Engine) buildFeatures() []spec.Feature {
	features := []spec.Feature{}

	// Get core features/commands based on preset
	coreFeatures := e.getAnswerList("core-features")
	mainCommands := e.getAnswerList("main-commands")
	mainResources := e.getAnswerList("main-resources")

	allFeatures := append(coreFeatures, mainCommands...)
	allFeatures = append(allFeatures, mainResources...)

	// Create features with appropriate priorities
	for i, featureName := range allFeatures {
		priority := "P0"
		if i >= 3 {
			priority = "P1"
		}
		if i >= 6 {
			priority = "P2"
		}

		feature := spec.Feature{
			ID:       types.FeatureID(fmt.Sprintf("feat-%03d", i+1)),
			Title:    strings.TrimSpace(featureName),
			Desc:     fmt.Sprintf("Implement %s functionality", featureName),
			Priority: types.Priority(priority),
			Success:  []string{fmt.Sprintf("%s is implemented and functional", featureName)},
			Trace:    []string{},
		}

		// Add context-specific details
		e.enrichFeature(&feature)

		features = append(features, feature)
	}

	// Add authentication feature if required
	if e.isYes("auth-required") {
		authType := e.getAnswer("auth-type", "auth-method")
		feature := spec.Feature{
			ID:       types.FeatureID(fmt.Sprintf("feat-%03d", len(features)+1)),
			Title:    "User Authentication",
			Desc:     "Implement user authentication and session management",
			Priority: types.Priority("P0"),
			API: []spec.API{
				{
					Path:   "/api/auth/login",
					Method: "POST",
				},
				{
					Path:   "/api/auth/logout",
					Method: "POST",
				},
			},
			Success: []string{
				fmt.Sprintf("Users can authenticate using %s", authType),
				"Sessions are securely managed",
			},
			Trace: []string{},
		}
		features = append(features, feature)
	}

	// Add configuration feature if needed
	if e.isYes("config-file") {
		configFormat := e.getAnswer("config-format")
		feature := spec.Feature{
			ID:       types.FeatureID(fmt.Sprintf("feat-%03d", len(features)+1)),
			Title:    "Configuration Management",
			Desc:     "Manage application configuration from file",
			Priority: types.Priority("P1"),
			Success: []string{
				fmt.Sprintf("Configuration loaded from %s file", configFormat),
				"Default configuration provided",
			},
			Trace: []string{},
		}
		features = append(features, feature)
	}

	return features
}

// enrichFeature adds context-specific details to a feature
func (e *Engine) enrichFeature(feature *spec.Feature) {
	// Add API endpoints for API services
	if e.session.Preset == "api-service" {
		resourceName := strings.ToLower(feature.Title)
		feature.API = []spec.API{
			{
				Path:   fmt.Sprintf("/api/%s", resourceName),
				Method: "GET",
			},
			{
				Path:   fmt.Sprintf("/api/%s", resourceName),
				Method: "POST",
			},
		}
		feature.Desc = fmt.Sprintf("Manage %s resources via REST API", feature.Title)
	}
}

// buildNonFunctional constructs non-functional requirements
func (e *Engine) buildNonFunctional() spec.NonFunctional {
	nf := spec.NonFunctional{
		Performance:  []string{},
		Security:     []string{},
		Scalability:  []string{},
		Availability: []string{},
	}

	// Performance requirements
	perf := e.getAnswer("performance-requirements", "expected-throughput", "data-volume")
	if perf != "" && strings.ToLower(perf) != "none" && strings.ToLower(perf) != "unknown" {
		nf.Performance = append(nf.Performance, perf)
	} else {
		nf.Performance = append(nf.Performance, "Response time < 2s for typical operations")
	}

	// Security requirements
	if e.isYes("auth-required") {
		authType := e.getAnswer("auth-type", "auth-method")
		nf.Security = append(nf.Security, fmt.Sprintf("Secure authentication using %s", authType))
		nf.Security = append(nf.Security, "All API endpoints require authentication")
	}

	if e.isYes("rate-limiting") {
		nf.Security = append(nf.Security, "Rate limiting to prevent abuse")
	}

	nf.Security = append(nf.Security, "HTTPS/TLS for all communications")
	nf.Security = append(nf.Security, "Input validation on all user inputs")

	// Availability requirements
	observability := e.getAnswerList("observability")
	if len(observability) > 0 {
		nf.Availability = append(nf.Availability, observability...)
	} else {
		nf.Availability = append(nf.Availability, "Structured logging")
		nf.Availability = append(nf.Availability, "Health check endpoints")
	}

	if e.isYes("data-quality") {
		nf.Availability = append(nf.Availability, "Data quality validation and monitoring")
	}

	// Scalability requirements
	throughput := e.getAnswer("expected-throughput", "data-volume")
	if throughput != "" && strings.ToLower(throughput) != "none" && strings.ToLower(throughput) != "unknown" {
		nf.Scalability = append(nf.Scalability, fmt.Sprintf("Support %s", throughput))
	}

	return nf
}

// buildMilestones constructs milestone list
func (e *Engine) buildMilestones() []spec.Milestone {
	return []spec.Milestone{
		{
			ID:          "m1",
			Name:        "MVP Launch",
			TargetDate:  "4 weeks",
			Description: "Core features functional and tested",
			FeatureIDs:  []types.FeatureID{}, // Filled in during plan generation
		},
		{
			ID:          "m2",
			Name:        "Beta Release",
			TargetDate:  "8 weeks",
			Description: "All P0 and P1 features complete",
			FeatureIDs:  []types.FeatureID{},
		},
		{
			ID:          "m3",
			Name:        "Production Ready",
			TargetDate:  "12 weeks",
			Description: "Full feature set with security and monitoring",
			FeatureIDs:  []types.FeatureID{},
		},
	}
}

// isYes checks if a question was answered with "yes"
func (e *Engine) isYes(questionID string) bool {
	answer := e.getAnswer(questionID)
	return strings.ToLower(strings.TrimSpace(answer)) == "yes"
}
