package spec

import (
	"encoding/json"
	"testing"
)

func TestCanonicalizeFeature(t *testing.T) {
	tests := []struct {
		name    string
		feature Feature
		wantErr bool
	}{
		{
			name: "basic feature",
			feature: Feature{
				ID:       "feat-001",
				Title:    "User Authentication",
				Desc:     "Implement JWT-based authentication",
				Priority: "P0",
				Success:  []string{"Users can login", "Tokens expire after 1h"},
				Trace:    []string{"PRD-001"},
			},
			wantErr: false,
		},
		{
			name: "feature with API",
			feature: Feature{
				ID:       "feat-002",
				Title:    "User Registration",
				Desc:     "Allow users to register",
				Priority: "P1",
				API: []API{
					{
						Method:   "POST",
						Path:     "/api/register",
						Request:  "UserRegistrationRequest",
						Response: "UserRegistrationResponse",
					},
				},
				Success: []string{"Users can register"},
				Trace:   []string{"PRD-002"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, err := Canonicalize(tt.feature)
			if (err != nil) != tt.wantErr {
				t.Errorf("Canonicalize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(canonical) == 0 {
				t.Error("Canonicalize() returned empty bytes")
			}
		})
	}
}

func TestHashFeature(t *testing.T) {
	feature := Feature{
		ID:       "feat-001",
		Title:    "Test Feature",
		Desc:     "Test description",
		Priority: "P0",
		Success:  []string{"Success criterion 1"},
		Trace:    []string{"PRD-001"},
	}

	// Hash the same feature twice - should be identical
	hash1, err := Hash(feature)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	hash2, err := Hash(feature)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash() not deterministic: %s != %s", hash1, hash2)
	}

	if len(hash1) != 64 { // blake3 produces 32 bytes = 64 hex chars
		t.Errorf("Hash() length = %d, want 64", len(hash1))
	}
}

func TestHashStability(t *testing.T) {
	// Create two identical features with different field ordering
	feature1 := Feature{
		ID:       "feat-001",
		Title:    "Test",
		Desc:     "Description",
		Priority: "P0",
		Success:  []string{"A", "B"},
		Trace:    []string{"X"},
	}

	feature2 := Feature{
		Success:  []string{"A", "B"},
		Priority: "P0",
		ID:       "feat-001",
		Trace:    []string{"X"},
		Title:    "Test",
		Desc:     "Description",
	}

	hash1, err := Hash(feature1)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	hash2, err := Hash(feature2)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash() not stable across field ordering: %s != %s", hash1, hash2)
	}
}

func TestSortKeys(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string // JSON representation for comparison
	}{
		{
			name: "map with unsorted keys",
			input: map[string]interface{}{
				"zebra":   "z",
				"alpha":   "a",
				"charlie": "c",
				"bravo":   "b",
			},
			want: `{"alpha":"a","bravo":"b","charlie":"c","zebra":"z"}`,
		},
		{
			name: "nested maps",
			input: map[string]interface{}{
				"outer": map[string]interface{}{
					"z": "last",
					"a": "first",
				},
			},
			want: `{"outer":{"a":"first","z":"last"}}`,
		},
		{
			name: "slice of interfaces",
			input: []interface{}{
				map[string]interface{}{
					"z": 1,
					"a": 2,
				},
				"plain string",
				123,
			},
			want: `[{"a":2,"z":1},"plain string",123]`,
		},
		{
			name: "slice of maps",
			input: []map[string]interface{}{
				{
					"z": "last",
					"a": "first",
				},
				{
					"y": "second-last",
					"b": "second-first",
				},
			},
			want: `[{"a":"first","z":"last"},{"b":"second-first","y":"second-last"}]`,
		},
		{
			name:  "primitive types",
			input: "plain string",
			want:  `"plain string"`,
		},
		{
			name:  "number",
			input: 42,
			want:  `42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortKeys(tt.input)

			// Marshal to JSON for comparison
			got, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("sortKeys() = %s, want %s", string(got), tt.want)
			}
		})
	}
}

func TestCanonicalizeWithMultipleAPIs(t *testing.T) {
	feature := Feature{
		ID:       "feat-003",
		Title:    "Multi-endpoint Feature",
		Desc:     "Feature with multiple API endpoints",
		Priority: "P0",
		API: []API{
			{
				Method:   "GET",
				Path:     "/api/users",
				Response: "UserListResponse",
			},
			{
				Method:   "POST",
				Path:     "/api/users",
				Request:  "CreateUserRequest",
				Response: "UserResponse",
			},
			{
				Method:  "DELETE",
				Path:    "/api/users/{id}",
				Request: "", // Empty request/response fields
			},
		},
		Success: []string{"CRUD operations work"},
		Trace:   []string{"PRD-003"},
	}

	canonical1, err := Canonicalize(feature)
	if err != nil {
		t.Fatalf("Canonicalize() error = %v", err)
	}

	// Canonicalize again to verify determinism
	canonical2, err := Canonicalize(feature)
	if err != nil {
		t.Fatalf("Canonicalize() error = %v", err)
	}

	if string(canonical1) != string(canonical2) {
		t.Errorf("Canonicalize() not deterministic with multiple APIs")
	}

	// Verify the JSON is valid
	var result map[string]interface{}
	if err := json.Unmarshal(canonical1, &result); err != nil {
		t.Fatalf("Canonicalize() produced invalid JSON: %v", err)
	}

	// Verify API field is present
	apis, ok := result["api"]
	if !ok {
		t.Error("Canonicalize() missing 'api' field")
	}

	// Verify it's an array
	apisArray, ok := apis.([]interface{})
	if !ok {
		t.Error("Canonicalize() 'api' field is not an array")
	}

	if len(apisArray) != 3 {
		t.Errorf("Canonicalize() 'api' array length = %d, want 3", len(apisArray))
	}
}

func TestHashWithComplexFeature(t *testing.T) {
	feature := Feature{
		ID:       "feat-004",
		Title:    "Complex Feature",
		Desc:     "Feature with all fields populated",
		Priority: "P1",
		API: []API{
			{
				Method:   "POST",
				Path:     "/api/complex",
				Request:  "ComplexRequest",
				Response: "ComplexResponse",
			},
			{
				Method: "GET",
				Path:   "/api/simple",
			},
		},
		Success: []string{
			"Requirement 1",
			"Requirement 2",
			"Requirement 3",
		},
		Trace: []string{
			"PRD-001",
			"PRD-002",
		},
	}

	hash1, err := Hash(feature)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Modify a field and verify hash changes
	feature.Title = "Modified Title"
	hash2, err := Hash(feature)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("Hash() should change when feature is modified")
	}
}

func TestCanonicalizeEmptyAPI(t *testing.T) {
	// Feature with empty API slice
	feature := Feature{
		ID:       "feat-005",
		Title:    "No API Feature",
		Desc:     "Feature without API endpoints",
		Priority: "P2",
		Success:  []string{"Works without API"},
		Trace:    []string{"PRD-005"},
		API:      []API{}, // Empty slice
	}

	canonical, err := Canonicalize(feature)
	if err != nil {
		t.Fatalf("Canonicalize() error = %v", err)
	}

	// Verify the JSON doesn't contain 'api' field when API is empty
	var result map[string]interface{}
	if err := json.Unmarshal(canonical, &result); err != nil {
		t.Fatalf("Canonicalize() produced invalid JSON: %v", err)
	}

	if _, ok := result["api"]; ok {
		t.Error("Canonicalize() should not include 'api' field when API slice is empty")
	}
}
