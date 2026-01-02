package awssm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: Config{
				Region:          "us-west-2",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Endpoint:        "http://localhost:4566", // LocalStack
			},
			wantErr: false,
		},
		{
			name: "missing region",
			config: Config{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			wantErr: true,
			errMsg:  "aws region is required",
		},
		{
			name: "with secondary region",
			config: Config{
				Region:          "us-west-2",
				SecondaryRegion: "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Endpoint:        "http://localhost:4566",
			},
			wantErr: false,
		},
		{
			name: "with non-existent profile",
			config: Config{
				Region:   "us-west-2",
				Profile:  "test-profile-nonexistent",
				Endpoint: "http://localhost:4566",
			},
			// This will fail during config load if profile doesn't exist
			wantErr: true,
			errMsg:  "failed to load AWS config",
		},
		{
			name: "with assume role",
			config: Config{
				Region:          "us-west-2",
				AssumeRoleARN:   "arn:aws:iam::123456789012:role/test-role",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Endpoint:        "http://localhost:4566",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.config.Region, client.Region())

				if tt.config.SecondaryRegion != "" {
					assert.True(t, client.HasSecondary())
					assert.Equal(t, tt.config.SecondaryRegion, client.SecondaryRegion())
					assert.NotNil(t, client.SecondarySecretsManager())
				} else {
					assert.False(t, client.HasSecondary())
					assert.Empty(t, client.SecondaryRegion())
					assert.Nil(t, client.SecondarySecretsManager())
				}

				assert.Equal(t, tt.config.Endpoint, client.Endpoint())
				assert.NotNil(t, client.SecretsManager())

				// Cleanup
				err = client.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, Config{
		Region:          "us-west-2",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:4566",
	})
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

func TestClient_Getters(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, Config{
		Region:          "us-west-2",
		SecondaryRegion: "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:4566",
	})
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, "us-west-2", client.Region())
	assert.Equal(t, "us-east-1", client.SecondaryRegion())
	assert.Equal(t, "http://localhost:4566", client.Endpoint())
	assert.True(t, client.HasSecondary())
	assert.NotNil(t, client.SecretsManager())
	assert.NotNil(t, client.SecondarySecretsManager())
}
