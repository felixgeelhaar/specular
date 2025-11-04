package drift

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/ai-dev/internal/spec"
	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPIValidator validates API implementations against OpenAPI specs
type OpenAPIValidator struct {
	spec *openapi3.T
	path string
}

// NewOpenAPIValidator creates a validator from an OpenAPI spec file
func NewOpenAPIValidator(specPath string) (*OpenAPIValidator, error) {
	// Load the OpenAPI spec
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Validate the spec
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	return &OpenAPIValidator{
		spec: doc,
		path: specPath,
	}, nil
}

// ValidateEndpoints checks if API endpoints from features exist in OpenAPI spec
func (v *OpenAPIValidator) ValidateEndpoints(features []spec.Feature) []Finding {
	var findings []Finding

	for _, feature := range features {
		if len(feature.API) == 0 {
			continue
		}

		for _, api := range feature.API {
			// Normalize path (remove query params, ensure leading slash)
			path := normalizePath(api.Path)
			method := strings.ToUpper(api.Method)

			// Check if path exists in OpenAPI spec
			pathItem := v.spec.Paths.Find(path)
			if pathItem == nil {
				// Try to find with path parameters
				pathItem = v.findPathWithParams(path)
			}

			if pathItem == nil {
				findings = append(findings, Finding{
					Code:      "MISSING_API_PATH",
					FeatureID: feature.ID,
					Message:   fmt.Sprintf("API path not found in OpenAPI spec: %s %s", method, path),
					Severity:  "error",
					Location:  fmt.Sprintf("%s:%s", v.path, path),
				})
				continue
			}

			// Check if method exists for the path
			if !v.hasMethod(pathItem, method) {
				findings = append(findings, Finding{
					Code:      "MISSING_API_METHOD",
					FeatureID: feature.ID,
					Message:   fmt.Sprintf("API method not found in OpenAPI spec: %s %s", method, path),
					Severity:  "error",
					Location:  fmt.Sprintf("%s:%s", v.path, path),
				})
			}
		}
	}

	return findings
}

// GetEndpointSummary returns a summary of all endpoints in the OpenAPI spec
func (v *OpenAPIValidator) GetEndpointSummary() map[string][]string {
	summary := make(map[string][]string)

	if v.spec.Paths == nil {
		return summary
	}

	for path, pathItem := range v.spec.Paths.Map() {
		methods := []string{}

		if pathItem.Get != nil {
			methods = append(methods, "GET")
		}
		if pathItem.Post != nil {
			methods = append(methods, "POST")
		}
		if pathItem.Put != nil {
			methods = append(methods, "PUT")
		}
		if pathItem.Patch != nil {
			methods = append(methods, "PATCH")
		}
		if pathItem.Delete != nil {
			methods = append(methods, "DELETE")
		}
		if pathItem.Head != nil {
			methods = append(methods, "HEAD")
		}
		if pathItem.Options != nil {
			methods = append(methods, "OPTIONS")
		}

		if len(methods) > 0 {
			summary[path] = methods
		}
	}

	return summary
}

// hasMethod checks if a path item has the specified HTTP method
func (v *OpenAPIValidator) hasMethod(pathItem *openapi3.PathItem, method string) bool {
	switch strings.ToUpper(method) {
	case "GET":
		return pathItem.Get != nil
	case "POST":
		return pathItem.Post != nil
	case "PUT":
		return pathItem.Put != nil
	case "PATCH":
		return pathItem.Patch != nil
	case "DELETE":
		return pathItem.Delete != nil
	case "HEAD":
		return pathItem.Head != nil
	case "OPTIONS":
		return pathItem.Options != nil
	default:
		return false
	}
}

// findPathWithParams tries to match a path that might have different parameter names
func (v *OpenAPIValidator) findPathWithParams(requestPath string) *openapi3.PathItem {
	// Split the request path into segments
	requestSegments := strings.Split(strings.Trim(requestPath, "/"), "/")

	for specPath, pathItem := range v.spec.Paths.Map() {
		specSegments := strings.Split(strings.Trim(specPath, "/"), "/")

		// Paths must have same number of segments
		if len(requestSegments) != len(specSegments) {
			continue
		}

		// Check if segments match (considering parameters)
		match := true
		for i := 0; i < len(requestSegments); i++ {
			reqSeg := requestSegments[i]
			specSeg := specSegments[i]

			// If spec segment is a parameter (starts with {)
			if strings.HasPrefix(specSeg, "{") && strings.HasSuffix(specSeg, "}") {
				continue // Parameters match any value
			}

			// If request segment is a parameter
			if strings.HasPrefix(reqSeg, "{") && strings.HasSuffix(reqSeg, "}") {
				continue // Parameters match any value
			}

			// Otherwise, segments must match exactly
			if reqSeg != specSeg {
				match = false
				break
			}
		}

		if match {
			return pathItem
		}
	}

	return nil
}

// normalizePath normalizes an API path
func normalizePath(path string) string {
	// Remove query parameters
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// Ensure leading slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Remove trailing slash (except for root)
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	return path
}

// ValidateAPISpec performs comprehensive OpenAPI spec validation
func ValidateAPISpec(specPath string, projectRoot string, features []spec.Feature) []Finding {
	var findings []Finding

	// Check if spec file exists
	fullPath := filepath.Join(projectRoot, specPath)
	if _, err := os.Stat(fullPath); err != nil {
		findings = append(findings, Finding{
			Code:     "MISSING_API_SPEC",
			Message:  fmt.Sprintf("OpenAPI spec not found at: %s", specPath),
			Severity: "error",
			Location: specPath,
		})
		return findings
	}

	// Create validator
	validator, err := NewOpenAPIValidator(fullPath)
	if err != nil {
		findings = append(findings, Finding{
			Code:     "INVALID_API_SPEC",
			Message:  fmt.Sprintf("Invalid OpenAPI spec: %v", err),
			Severity: "error",
			Location: specPath,
		})
		return findings
	}

	// Validate endpoints
	endpointFindings := validator.ValidateEndpoints(features)
	findings = append(findings, endpointFindings...)

	return findings
}
