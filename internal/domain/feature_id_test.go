package domain

import (
	"strings"
	"testing"
)

func TestNewFeatureID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid simple ID",
			value:   "auth",
			wantErr: false,
		},
		{
			name:    "valid ID with hyphen",
			value:   "user-auth",
			wantErr: false,
		},
		{
			name:    "valid ID with multiple hyphens",
			value:   "user-profile-management",
			wantErr: false,
		},
		{
			name:    "valid ID with numbers",
			value:   "feature-001",
			wantErr: false,
		},
		{
			name:    "valid ID starts with letter",
			value:   "f123",
			wantErr: false,
		},
		{
			name:    "empty ID",
			value:   "",
			wantErr: true,
		},
		{
			name:    "ID starts with number",
			value:   "123-feature",
			wantErr: true,
		},
		{
			name:    "ID starts with hyphen",
			value:   "-feature",
			wantErr: true,
		},
		{
			name:    "ID ends with hyphen",
			value:   "feature-",
			wantErr: true,
		},
		{
			name:    "ID with consecutive hyphens",
			value:   "feature--auth",
			wantErr: true,
		},
		{
			name:    "ID with uppercase letters",
			value:   "Feature-Auth",
			wantErr: true,
		},
		{
			name:    "ID with special characters",
			value:   "feature_auth",
			wantErr: true,
		},
		{
			name:    "ID with spaces",
			value:   "feature auth",
			wantErr: true,
		},
		{
			name:    "ID exceeds max length",
			value:   strings.Repeat("a", 101),
			wantErr: true,
		},
		{
			name:    "ID at max length",
			value:   strings.Repeat("a", 100),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFeatureID(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFeatureID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.value {
				t.Errorf("NewFeatureID() = %v, want %v", got, tt.value)
			}
		})
	}
}

func TestFeatureID_Validate(t *testing.T) {
	tests := []struct {
		name     string
		featureID FeatureID
		wantErr  bool
	}{
		{"valid simple ID", FeatureID("auth"), false},
		{"valid with hyphens", FeatureID("user-profile-management"), false},
		{"valid with numbers", FeatureID("feature-001"), false},
		{"empty is invalid", FeatureID(""), true},
		{"starts with number is invalid", FeatureID("123-feature"), true},
		{"ends with hyphen is invalid", FeatureID("feature-"), true},
		{"consecutive hyphens are invalid", FeatureID("feature--auth"), true},
		{"uppercase is invalid", FeatureID("Feature"), true},
		{"too long is invalid", FeatureID(strings.Repeat("a", 101)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.featureID.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("FeatureID.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFeatureID_String(t *testing.T) {
	tests := []struct {
		name     string
		featureID FeatureID
		want     string
	}{
		{"simple ID", FeatureID("auth"), "auth"},
		{"ID with hyphens", FeatureID("user-auth"), "user-auth"},
		{"ID with numbers", FeatureID("feature-001"), "feature-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.featureID.String(); got != tt.want {
				t.Errorf("FeatureID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeatureID_Equals(t *testing.T) {
	tests := []struct {
		name  string
		id1   FeatureID
		id2   FeatureID
		want  bool
	}{
		{"same IDs", FeatureID("auth"), FeatureID("auth"), true},
		{"different IDs", FeatureID("auth"), FeatureID("profile"), false},
		{"empty IDs", FeatureID(""), FeatureID(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id1.Equals(tt.id2); got != tt.want {
				t.Errorf("FeatureID.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
