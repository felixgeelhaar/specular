package spec

import (
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
