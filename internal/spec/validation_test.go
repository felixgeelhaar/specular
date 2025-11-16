package spec

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func TestFeature_Validate(t *testing.T) {
	validFeature := Feature{
		ID:       "user-auth",
		Title:    "User Authentication",
		Desc:     "Implement user authentication with JWT",
		Priority: "P0",
		Success:  []string{"Users can log in", "Tokens are secure"},
		API: []API{
			{Method: "POST", Path: "/api/login"},
		},
		Trace: []string{"Login flow"},
	}

	tests := []struct {
		name    string
		feature Feature
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid feature",
			feature: validFeature,
			wantErr: false,
		},
		{
			name: "invalid ID - empty",
			feature: Feature{
				ID:       "",
				Title:    "Test",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Test"},
			},
			wantErr: true,
			errMsg:  "invalid feature ID",
		},
		{
			name: "invalid ID - uppercase",
			feature: Feature{
				ID:       "User-Auth",
				Title:    "Test",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Test"},
			},
			wantErr: true,
			errMsg:  "invalid feature ID",
		},
		{
			name: "invalid ID - starts with number",
			feature: Feature{
				ID:       "123-feature",
				Title:    "Test",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Test"},
			},
			wantErr: true,
			errMsg:  "invalid feature ID",
		},
		{
			name: "empty title",
			feature: Feature{
				ID:       "test-feature",
				Title:    "",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Test"},
			},
			wantErr: true,
			errMsg:  "title cannot be empty",
		},
		{
			name: "whitespace-only title",
			feature: Feature{
				ID:       "test-feature",
				Title:    "   ",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Test"},
			},
			wantErr: true,
			errMsg:  "title cannot be empty",
		},
		{
			name: "empty description",
			feature: Feature{
				ID:       "test-feature",
				Title:    "Test Feature",
				Desc:     "",
				Priority: "P0",
				Success:  []string{"Test"},
			},
			wantErr: true,
			errMsg:  "description cannot be empty",
		},
		{
			name: "invalid priority",
			feature: Feature{
				ID:       "test-feature",
				Title:    "Test Feature",
				Desc:     "Test description",
				Priority: "P3",
				Success:  []string{"Test"},
			},
			wantErr: true,
			errMsg:  "invalid feature priority",
		},
		{
			name: "no success criteria",
			feature: Feature{
				ID:       "test-feature",
				Title:    "Test Feature",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{},
			},
			wantErr: true,
			errMsg:  "at least one success criterion",
		},
		{
			name: "empty success criterion",
			feature: Feature{
				ID:       "test-feature",
				Title:    "Test Feature",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Valid criterion", ""},
			},
			wantErr: true,
			errMsg:  "success criterion at index 1 cannot be empty",
		},
		{
			name: "invalid API endpoint",
			feature: Feature{
				ID:       "test-feature",
				Title:    "Test Feature",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Test"},
				API: []API{
					{Method: "INVALID", Path: "/api/test"},
				},
			},
			wantErr: true,
			errMsg:  "API endpoint at index 0 is invalid",
		},
		{
			name: "empty trace item",
			feature: Feature{
				ID:       "test-feature",
				Title:    "Test Feature",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Test"},
				Trace:    []string{"Valid trace", ""},
			},
			wantErr: true,
			errMsg:  "trace at index 1 cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.feature.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Feature.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Feature.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestAPI_Validate(t *testing.T) {
	tests := []struct {
		name    string
		api     API
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid GET endpoint",
			api:     API{Method: "GET", Path: "/api/users"},
			wantErr: false,
		},
		{
			name:    "valid POST endpoint",
			api:     API{Method: "POST", Path: "/api/users"},
			wantErr: false,
		},
		{
			name:    "valid with lowercase method",
			api:     API{Method: "get", Path: "/api/users"},
			wantErr: false,
		},
		{
			name:    "valid with request and response",
			api:     API{Method: "POST", Path: "/api/users", Request: "User data", Response: "User ID"},
			wantErr: false,
		},
		{
			name:    "empty method",
			api:     API{Method: "", Path: "/api/users"},
			wantErr: true,
			errMsg:  "method cannot be empty",
		},
		{
			name:    "invalid HTTP method",
			api:     API{Method: "INVALID", Path: "/api/users"},
			wantErr: true,
			errMsg:  "not a valid HTTP method",
		},
		{
			name:    "empty path",
			api:     API{Method: "GET", Path: ""},
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name:    "path without leading slash",
			api:     API{Method: "GET", Path: "api/users"},
			wantErr: true,
			errMsg:  "must start with /",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.api.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("API.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("API.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestProductSpec_Validate(t *testing.T) {
	validSpec := ProductSpec{
		Product: "Test Product",
		Goals:   []string{"Goal 1", "Goal 2"},
		Features: []Feature{
			{
				ID:       "feature-1",
				Title:    "Feature 1",
				Desc:     "Description 1",
				Priority: "P0",
				Success:  []string{"Success 1"},
			},
		},
		Acceptance: []string{"Acceptance 1"},
		Milestones: []Milestone{
			{
				ID:         "milestone-1",
				Name:       "Milestone 1",
				FeatureIDs: []types.FeatureID{types.FeatureID("feature-1")},
			},
		},
	}

	tests := []struct {
		name    string
		spec    ProductSpec
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid product spec",
			spec:    validSpec,
			wantErr: false,
		},
		{
			name: "empty product name",
			spec: ProductSpec{
				Product:    "",
				Goals:      []string{"Goal"},
				Features:   validSpec.Features,
				Acceptance: []string{"Acceptance"},
			},
			wantErr: true,
			errMsg:  "product name cannot be empty",
		},
		{
			name: "no goals",
			spec: ProductSpec{
				Product:    "Product",
				Goals:      []string{},
				Features:   validSpec.Features,
				Acceptance: []string{"Acceptance"},
			},
			wantErr: true,
			errMsg:  "at least one goal",
		},
		{
			name: "empty goal",
			spec: ProductSpec{
				Product:    "Product",
				Goals:      []string{"Goal 1", ""},
				Features:   validSpec.Features,
				Acceptance: []string{"Acceptance"},
			},
			wantErr: true,
			errMsg:  "goal at index 1 cannot be empty",
		},
		{
			name: "no features",
			spec: ProductSpec{
				Product:    "Product",
				Goals:      []string{"Goal"},
				Features:   []Feature{},
				Acceptance: []string{"Acceptance"},
			},
			wantErr: true,
			errMsg:  "at least one feature",
		},
		{
			name: "invalid feature",
			spec: ProductSpec{
				Product: "Product",
				Goals:   []string{"Goal"},
				Features: []Feature{
					{
						ID:       "feature-1",
						Title:    "",
						Desc:     "Desc",
						Priority: "P0",
						Success:  []string{"Success"},
					},
				},
				Acceptance: []string{"Acceptance"},
			},
			wantErr: true,
			errMsg:  "feature at index 0",
		},
		{
			name: "no acceptance criteria",
			spec: ProductSpec{
				Product:    "Product",
				Goals:      []string{"Goal"},
				Features:   validSpec.Features,
				Acceptance: []string{},
			},
			wantErr: true,
			errMsg:  "at least one acceptance criterion",
		},
		{
			name: "empty acceptance criterion",
			spec: ProductSpec{
				Product:    "Product",
				Goals:      []string{"Goal"},
				Features:   validSpec.Features,
				Acceptance: []string{"Acceptance 1", ""},
			},
			wantErr: true,
			errMsg:  "acceptance criterion at index 1 cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProductSpec.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ProductSpec.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestMilestone_Validate(t *testing.T) {
	tests := []struct {
		name      string
		milestone Milestone
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid milestone",
			milestone: Milestone{
				ID:         "milestone-1",
				Name:       "Milestone 1",
				FeatureIDs: []types.FeatureID{types.FeatureID("feature-1"), types.FeatureID("feature-2")},
			},
			wantErr: false,
		},
		{
			name: "valid with optional fields",
			milestone: Milestone{
				ID:          "milestone-1",
				Name:        "Milestone 1",
				FeatureIDs:  []types.FeatureID{types.FeatureID("feature-1")},
				TargetDate:  "2024-12-31",
				Description: "Q4 milestone",
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			milestone: Milestone{
				ID:         "",
				Name:       "Milestone",
				FeatureIDs: []types.FeatureID{types.FeatureID("feature-1")},
			},
			wantErr: true,
			errMsg:  "ID cannot be empty",
		},
		{
			name: "empty name",
			milestone: Milestone{
				ID:         "milestone-1",
				Name:       "",
				FeatureIDs: []types.FeatureID{types.FeatureID("feature-1")},
			},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name: "no feature IDs",
			milestone: Milestone{
				ID:         "milestone-1",
				Name:       "Milestone",
				FeatureIDs: []types.FeatureID{},
			},
			wantErr: true,
			errMsg:  "at least one feature",
		},
		{
			name: "invalid feature ID",
			milestone: Milestone{
				ID:         "milestone-1",
				Name:       "Milestone",
				FeatureIDs: []types.FeatureID{types.FeatureID("Feature-1")},
			},
			wantErr: true,
			errMsg:  "feature ID at index 0 is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.milestone.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Milestone.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Milestone.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}
