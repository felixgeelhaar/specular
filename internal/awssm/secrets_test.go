package awssm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecrets_Basic(t *testing.T) {
	// Test that Secrets() returns a Secrets instance
	ctx := context.Background()
	client, err := NewClient(ctx, Config{
		Region:          "us-west-2",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:4566",
	})
	require.NoError(t, err)
	defer client.Close()

	secrets := client.Secrets()
	assert.NotNil(t, secrets)
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "contains substring",
			s:        "ResourceNotFoundException: not found",
			substr:   "ResourceNotFoundException",
			expected: true,
		},
		{
			name:     "does not contain substring",
			s:        "Some other error",
			substr:   "ResourceNotFoundException",
			expected: false,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "test",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "test",
			substr:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecret_Structure(t *testing.T) {
	// Test Secret struct fields
	secret := &Secret{
		Name:         "test-secret",
		Data:         map[string]interface{}{"key": "value"},
		VersionID:    "v1",
		VersionStage: "AWSCURRENT",
		ARN:          "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret",
		CreatedDate:  "2024-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-secret", secret.Name)
	assert.Equal(t, "value", secret.Data["key"])
	assert.Equal(t, "v1", secret.VersionID)
	assert.Equal(t, "AWSCURRENT", secret.VersionStage)
	assert.Contains(t, secret.ARN, "test-secret")
}

func TestSecretInfo_Structure(t *testing.T) {
	// Test SecretInfo struct fields
	info := &SecretInfo{
		Name:             "test-secret",
		ARN:              "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret",
		Description:      "Test secret",
		Tags:             map[string]string{"env": "test"},
		LastChangedDate:  "2024-01-01T00:00:00Z",
		LastAccessedDate: "2024-01-02T00:00:00Z",
		DeletedDate:      "",
		VersionIDs:       []string{"v1", "v2"},
	}

	assert.Equal(t, "test-secret", info.Name)
	assert.Contains(t, info.ARN, "test-secret")
	assert.Equal(t, "Test secret", info.Description)
	assert.Equal(t, "test", info.Tags["env"])
	assert.Len(t, info.VersionIDs, 2)
}
