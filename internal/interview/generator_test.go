package interview

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/spec"
)

func TestIsComplete(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		expected bool
	}{
		{
			name:     "not complete - at start",
			current:  0,
			total:    10,
			expected: false,
		},
		{
			name:     "not complete - in middle",
			current:  5,
			total:    10,
			expected: false,
		},
		{
			name:     "complete - at end",
			current:  10,
			total:    10,
			expected: true,
		},
		{
			name:     "complete - past end",
			current:  15,
			total:    10,
			expected: true,
		},
		{
			name:     "empty interview",
			current:  0,
			total:    0,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Questions: make([]Question, tt.total),
					Current:   tt.current,
				},
			}

			got := engine.IsComplete()
			if got != tt.expected {
				t.Errorf("IsComplete() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetSession(t *testing.T) {
	session := &Session{
		ID:     "test-session-id",
		Preset: "web-app",
		Strict: true,
	}

	engine := &Engine{
		session: session,
	}

	got := engine.GetSession()
	if got != session {
		t.Error("GetSession() returned different session")
	}
	if got.ID != "test-session-id" {
		t.Errorf("GetSession().ID = %v, want test-session-id", got.ID)
	}
}

func TestGetResult(t *testing.T) {
	tests := []struct {
		name         string
		setupEngine  func() *Engine
		wantErr      bool
		errContains  string
		validateSpec func(*testing.T, *spec.ProductSpec)
	}{
		{
			name: "complete interview with all answers",
			setupEngine: func() *Engine {
				engine := &Engine{
					session: &Session{
						Preset:    "cli-tool",
						Questions: []Question{{ID: "q1"}},
						Current:   1, // Complete
						Answers: map[string]Answer{
							"product-name":     {Value: "TestTool"},
							"tool-purpose":     {Value: "Test automation tool"},
							"main-commands":    {Values: []string{"run", "test", "deploy"}},
							"auth-required":    {Value: "no"},
							"config-file":      {Value: "no"},
							"observability":    {Values: []string{"logging", "metrics"}},
							"success-criteria": {Values: []string{"Tests pass", "Coverage > 80%"}},
						},
					},
				}
				return engine
			},
			wantErr: false,
			validateSpec: func(t *testing.T, s *spec.ProductSpec) {
				if s.Product != "TestTool" {
					t.Errorf("Product = %v, want TestTool", s.Product)
				}
				if len(s.Features) == 0 {
					t.Error("Features should not be empty")
				}
				if len(s.Goals) == 0 {
					t.Error("Goals should not be empty")
				}
				if len(s.Milestones) != 3 {
					t.Errorf("Milestones length = %d, want 3", len(s.Milestones))
				}
			},
		},
		{
			name: "incomplete interview",
			setupEngine: func() *Engine {
				return &Engine{
					session: &Session{
						Questions: []Question{{ID: "q1"}, {ID: "q2"}},
						Current:   0, // Not complete
						Answers:   make(map[string]Answer),
					},
				}
			},
			wantErr:     true,
			errContains: "not complete",
		},
		{
			name: "missing product name",
			setupEngine: func() *Engine {
				return &Engine{
					session: &Session{
						Questions: []Question{{ID: "q1"}},
						Current:   1, // Complete
						Answers:   make(map[string]Answer),
					},
				}
			},
			wantErr:     true,
			errContains: "product name not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := tt.setupEngine()
			result, err := engine.GetResult()

			if tt.wantErr {
				if err == nil {
					t.Error("GetResult() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("GetResult() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetResult() unexpected error = %v", err)
			}

			if result == nil {
				t.Fatal("GetResult() returned nil result")
			}

			if result.Spec == nil {
				t.Fatal("GetResult() returned nil spec")
			}

			if tt.validateSpec != nil {
				tt.validateSpec(t, result.Spec)
			}

			if result.Duration < 0 {
				t.Error("GetResult() Duration should be >= 0")
			}
		})
	}
}

func TestGetAnswer(t *testing.T) {
	tests := []struct {
		name        string
		answers     map[string]Answer
		questionIDs []string
		expected    string
	}{
		{
			name: "single question ID - found",
			answers: map[string]Answer{
				"product-name": {Value: "MyProduct"},
			},
			questionIDs: []string{"product-name"},
			expected:    "MyProduct",
		},
		{
			name: "multiple question IDs - first found",
			answers: map[string]Answer{
				"product-name": {Value: "MyProduct"},
				"service-name": {Value: "MyService"},
			},
			questionIDs: []string{"product-name", "service-name"},
			expected:    "MyProduct",
		},
		{
			name: "multiple question IDs - second found",
			answers: map[string]Answer{
				"service-name": {Value: "MyService"},
			},
			questionIDs: []string{"product-name", "service-name"},
			expected:    "MyService",
		},
		{
			name: "not found",
			answers: map[string]Answer{
				"other": {Value: "value"},
			},
			questionIDs: []string{"product-name"},
			expected:    "",
		},
		{
			name: "empty value",
			answers: map[string]Answer{
				"product-name": {Value: ""},
			},
			questionIDs: []string{"product-name"},
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Answers: tt.answers,
				},
			}

			got := engine.getAnswer(tt.questionIDs...)
			if got != tt.expected {
				t.Errorf("getAnswer() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetAnswerList(t *testing.T) {
	tests := []struct {
		name       string
		answers    map[string]Answer
		questionID string
		expected   []string
	}{
		{
			name: "values array",
			answers: map[string]Answer{
				"features": {Values: []string{"feat1", "feat2", "feat3"}},
			},
			questionID: "features",
			expected:   []string{"feat1", "feat2", "feat3"},
		},
		{
			name: "single value converted to array",
			answers: map[string]Answer{
				"feature": {Value: "single-feature"},
			},
			questionID: "feature",
			expected:   []string{"single-feature"},
		},
		{
			name:       "not found",
			answers:    map[string]Answer{},
			questionID: "missing",
			expected:   []string{},
		},
		{
			name: "empty values",
			answers: map[string]Answer{
				"empty": {Values: []string{}},
			},
			questionID: "empty",
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Answers: tt.answers,
				},
			}

			got := engine.getAnswerList(tt.questionID)
			if len(got) != len(tt.expected) {
				t.Errorf("getAnswerList() length = %d, want %d", len(got), len(tt.expected))
				return
			}

			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("getAnswerList()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestIsYes(t *testing.T) {
	tests := []struct {
		name       string
		answers    map[string]Answer
		questionID string
		expected   bool
	}{
		{
			name:       "lowercase yes",
			answers:    map[string]Answer{"auth": {Value: "yes"}},
			questionID: "auth",
			expected:   true,
		},
		{
			name:       "uppercase YES",
			answers:    map[string]Answer{"auth": {Value: "YES"}},
			questionID: "auth",
			expected:   true,
		},
		{
			name:       "mixed case Yes",
			answers:    map[string]Answer{"auth": {Value: "Yes"}},
			questionID: "auth",
			expected:   true,
		},
		{
			name:       "with whitespace",
			answers:    map[string]Answer{"auth": {Value: "  yes  "}},
			questionID: "auth",
			expected:   true,
		},
		{
			name:       "no answer",
			answers:    map[string]Answer{"auth": {Value: "no"}},
			questionID: "auth",
			expected:   false,
		},
		{
			name:       "other answer",
			answers:    map[string]Answer{"auth": {Value: "maybe"}},
			questionID: "auth",
			expected:   false,
		},
		{
			name:       "not found",
			answers:    map[string]Answer{},
			questionID: "missing",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Answers: tt.answers,
				},
			}

			got := engine.isYes(tt.questionID)
			if got != tt.expected {
				t.Errorf("isYes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildGoals(t *testing.T) {
	tests := []struct {
		name     string
		answers  map[string]Answer
		expected []string
	}{
		{
			name: "complete goals",
			answers: map[string]Answer{
				"product-purpose":          {Value: "Automate testing workflows"},
				"target-users":             {Value: "developers and QA teams"},
				"performance-requirements": {Value: "Process 1000 tests/second"},
			},
			expected: []string{
				"Automate testing workflows",
				"Serve developers and QA teams with an intuitive interface",
				"Achieve performance targets: Process 1000 tests/second",
			},
		},
		{
			name: "minimal goals - fallback to MVP",
			answers: map[string]Answer{
				"product-purpose": {Value: ""},
			},
			expected: []string{"Deliver a functional MVP"},
		},
		{
			name: "skip none/unknown performance",
			answers: map[string]Answer{
				"product-purpose":          {Value: "Test tool"},
				"performance-requirements": {Value: "none"},
			},
			expected: []string{"Test tool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Answers: tt.answers,
				},
			}

			got := engine.buildGoals()
			if len(got) != len(tt.expected) {
				t.Errorf("buildGoals() length = %d, want %d", len(got), len(tt.expected))
				return
			}

			for i, goal := range got {
				if goal != tt.expected[i] {
					t.Errorf("buildGoals()[%d] = %v, want %v", i, goal, tt.expected[i])
				}
			}
		})
	}
}

func TestBuildFeatures(t *testing.T) {
	tests := []struct {
		name             string
		answers          map[string]Answer
		minFeatureCount  int
		hasAuthFeature   bool
		hasConfigFeature bool
	}{
		{
			name: "basic features without auth or config",
			answers: map[string]Answer{
				"core-features": {Values: []string{"feat1", "feat2"}},
				"auth-required": {Value: "no"},
				"config-file":   {Value: "no"},
			},
			minFeatureCount:  2,
			hasAuthFeature:   false,
			hasConfigFeature: false,
		},
		{
			name: "features with auth",
			answers: map[string]Answer{
				"core-features": {Values: []string{"feat1"}},
				"auth-required": {Value: "yes"},
				"auth-type":     {Value: "JWT"},
				"config-file":   {Value: "no"},
			},
			minFeatureCount:  2,
			hasAuthFeature:   true,
			hasConfigFeature: false,
		},
		{
			name: "features with config",
			answers: map[string]Answer{
				"main-commands": {Values: []string{"cmd1", "cmd2"}},
				"auth-required": {Value: "no"},
				"config-file":   {Value: "yes"},
				"config-format": {Value: "YAML"},
			},
			minFeatureCount:  3,
			hasAuthFeature:   false,
			hasConfigFeature: true,
		},
		{
			name: "many features - priority assignment",
			answers: map[string]Answer{
				"core-features": {Values: []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7"}},
				"auth-required": {Value: "no"},
				"config-file":   {Value: "no"},
			},
			minFeatureCount: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Answers: tt.answers,
				},
			}

			got := engine.buildFeatures()

			if len(got) < tt.minFeatureCount {
				t.Errorf("buildFeatures() length = %d, want at least %d", len(got), tt.minFeatureCount)
			}

			// Check for auth feature
			hasAuth := false
			for _, f := range got {
				if f.Title == "User Authentication" {
					hasAuth = true
					if f.Priority != "P0" {
						t.Error("Auth feature should have P0 priority")
					}
					if len(f.API) != 2 {
						t.Errorf("Auth feature API endpoints = %d, want 2", len(f.API))
					}
				}
			}
			if hasAuth != tt.hasAuthFeature {
				t.Errorf("buildFeatures() has auth = %v, want %v", hasAuth, tt.hasAuthFeature)
			}

			// Check for config feature
			hasConfig := false
			for _, f := range got {
				if f.Title == "Configuration Management" {
					hasConfig = true
					if f.Priority != "P1" {
						t.Error("Config feature should have P1 priority")
					}
				}
			}
			if hasConfig != tt.hasConfigFeature {
				t.Errorf("buildFeatures() has config = %v, want %v", hasConfig, tt.hasConfigFeature)
			}

			// Verify feature structure
			for i, f := range got {
				if f.ID == "" {
					t.Errorf("Feature[%d] has empty ID", i)
				}
				if f.Title == "" {
					t.Errorf("Feature[%d] has empty Title", i)
				}
				if f.Priority != "P0" && f.Priority != "P1" && f.Priority != "P2" {
					t.Errorf("Feature[%d] has invalid priority: %s", i, f.Priority)
				}
			}
		})
	}
}

func TestEnrichFeature(t *testing.T) {
	tests := []struct {
		name      string
		preset    string
		feature   spec.Feature
		expectAPI bool
	}{
		{
			name:   "api-service preset adds API endpoints",
			preset: "api-service",
			feature: spec.Feature{
				Title: "Users",
			},
			expectAPI: true,
		},
		{
			name:   "non-api preset doesn't add endpoints",
			preset: "cli-tool",
			feature: spec.Feature{
				Title: "Command",
			},
			expectAPI: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Preset: tt.preset,
				},
			}

			feature := tt.feature
			engine.enrichFeature(&feature)

			if tt.expectAPI {
				if len(feature.API) == 0 {
					t.Error("enrichFeature() should add API endpoints for api-service preset")
				}
				// Verify GET and POST methods
				hasGet := false
				hasPost := false
				for _, api := range feature.API {
					if api.Method == "GET" {
						hasGet = true
					}
					if api.Method == "POST" {
						hasPost = true
					}
				}
				if !hasGet || !hasPost {
					t.Error("enrichFeature() should add both GET and POST endpoints")
				}
			} else {
				if len(feature.API) > 0 {
					t.Error("enrichFeature() should not add API endpoints for non-api preset")
				}
			}
		})
	}
}

func TestBuildNonFunctional(t *testing.T) {
	tests := []struct {
		name     string
		answers  map[string]Answer
		validate func(*testing.T, spec.NonFunctional)
	}{
		{
			name: "with auth and observability",
			answers: map[string]Answer{
				"auth-required":            {Value: "yes"},
				"auth-type":                {Value: "OAuth2"},
				"rate-limiting":            {Value: "yes"},
				"observability":            {Values: []string{"logging", "metrics", "tracing"}},
				"performance-requirements": {Value: "1000 req/s"},
			},
			validate: func(t *testing.T, nf spec.NonFunctional) {
				if len(nf.Security) < 3 {
					t.Errorf("Security requirements = %d, want at least 3", len(nf.Security))
				}
				if len(nf.Availability) < 3 {
					t.Errorf("Availability requirements = %d, want at least 3", len(nf.Availability))
				}
				if len(nf.Performance) == 0 {
					t.Error("Performance requirements should not be empty")
				}

				// Check for auth in security
				hasAuth := false
				for _, s := range nf.Security {
					if contains(s, "OAuth2") {
						hasAuth = true
					}
				}
				if !hasAuth {
					t.Error("Security should include auth type")
				}
			},
		},
		{
			name: "minimal - defaults",
			answers: map[string]Answer{
				"auth-required": {Value: "no"},
			},
			validate: func(t *testing.T, nf spec.NonFunctional) {
				// Should have defaults
				if len(nf.Performance) == 0 {
					t.Error("Should have default performance requirements")
				}
				if len(nf.Security) < 2 {
					t.Error("Should have default security requirements (HTTPS, input validation)")
				}
				if len(nf.Availability) < 2 {
					t.Error("Should have default availability requirements")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Answers: tt.answers,
				},
			}

			got := engine.buildNonFunctional()
			tt.validate(t, got)
		})
	}
}

func TestBuildMilestones(t *testing.T) {
	engine := &Engine{
		session: &Session{},
	}

	milestones := engine.buildMilestones()

	if len(milestones) != 3 {
		t.Errorf("buildMilestones() length = %d, want 3", len(milestones))
	}

	expectedIDs := []string{"m1", "m2", "m3"}
	expectedNames := []string{"MVP Launch", "Beta Release", "Production Ready"}

	for i, m := range milestones {
		if m.ID != expectedIDs[i] {
			t.Errorf("Milestone[%d].ID = %v, want %v", i, m.ID, expectedIDs[i])
		}
		if m.Name != expectedNames[i] {
			t.Errorf("Milestone[%d].Name = %v, want %v", i, m.Name, expectedNames[i])
		}
		if m.TargetDate == "" {
			t.Errorf("Milestone[%d] has empty TargetDate", i)
		}
		if m.Description == "" {
			t.Errorf("Milestone[%d] has empty Description", i)
		}
	}
}

func TestGenerateSpec(t *testing.T) {
	tests := []struct {
		name    string
		answers map[string]Answer
		wantErr bool
	}{
		{
			name: "complete spec generation",
			answers: map[string]Answer{
				"product-name":     {Value: "TestProduct"},
				"product-purpose":  {Value: "Test automation"},
				"core-features":    {Values: []string{"run", "test"}},
				"success-criteria": {Values: []string{"Tests pass"}},
			},
			wantErr: false,
		},
		{
			name:    "missing product name",
			answers: map[string]Answer{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				session: &Session{
					Answers: tt.answers,
				},
			}

			spec, err := engine.generateSpec()

			if tt.wantErr {
				if err == nil {
					t.Error("generateSpec() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("generateSpec() unexpected error = %v", err)
			}

			if spec == nil {
				t.Fatal("generateSpec() returned nil spec")
			}

			if spec.Product == "" {
				t.Error("generateSpec() returned spec with empty Product")
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
