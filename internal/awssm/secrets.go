package awssm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// Secret represents an AWS Secrets Manager secret.
type Secret struct {
	// Name is the secret name/ARN
	Name string `json:"name"`

	// Data is the secret data (parsed JSON)
	Data map[string]interface{} `json:"data"`

	// VersionID is the unique version identifier
	VersionID string `json:"version_id,omitempty"`

	// VersionStage indicates the stage (AWSCURRENT, AWSPREVIOUS, etc.)
	VersionStage string `json:"version_stage,omitempty"`

	// ARN is the secret ARN
	ARN string `json:"arn,omitempty"`

	// CreatedDate is when the secret was created
	CreatedDate string `json:"created_date,omitempty"`
}

// Secrets provides access to AWS Secrets Manager operations.
type Secrets struct {
	client *Client
}

// Secrets returns a Secrets client for the Client.
func (c *Client) Secrets() *Secrets {
	return &Secrets{client: c}
}

// Put creates or updates a secret.
//
// Example:
//
//	err := client.Secrets().Put(ctx, "my-app/db-password", map[string]interface{}{
//	    "username": "admin",
//	    "password": "secret123",
//	})
func (s *Secrets) Put(ctx context.Context, name string, data map[string]interface{}) error {
	return s.PutWithTags(ctx, name, data, nil)
}

// PutWithTags creates or updates a secret with tags.
func (s *Secrets) PutWithTags(ctx context.Context, name string, data map[string]interface{}, tags map[string]string) error {
	// Serialize data to JSON
	secretBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal secret data: %w", err)
	}
	secretString := string(secretBytes)

	// Try to update existing secret first
	updateInput := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(name),
		SecretString: aws.String(secretString),
	}

	_, err = s.client.smClient.PutSecretValue(ctx, updateInput)
	if err != nil {
		// If secret doesn't exist, create it
		var notFoundErr *types.ResourceNotFoundException
		if isNotFoundError(err) {
			return s.create(ctx, name, secretString, tags)
		}
		// Check for resource not found type
		if _, ok := err.(*types.ResourceNotFoundException); ok {
			return s.create(ctx, name, secretString, tags)
		}
		_ = notFoundErr // Satisfy linter
		return fmt.Errorf("failed to put secret: %w", err)
	}

	return nil
}

// create creates a new secret.
func (s *Secrets) create(ctx context.Context, name, secretString string, tags map[string]string) error {
	createInput := &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(secretString),
	}

	// Add tags if provided
	if len(tags) > 0 {
		var awsTags []types.Tag
		for k, v := range tags {
			awsTags = append(awsTags, types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
		createInput.Tags = awsTags
	}

	_, err := s.client.smClient.CreateSecret(ctx, createInput)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

// Get reads a secret (current version).
//
// Example:
//
//	secret, err := client.Secrets().Get(ctx, "my-app/db-password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	password := secret.Data["password"].(string)
func (s *Secrets) Get(ctx context.Context, name string) (*Secret, error) {
	return s.GetByStage(ctx, name, "AWSCURRENT")
}

// GetByStage reads a specific version stage of a secret.
//
// Stages:
//   - AWSCURRENT: The current/active version
//   - AWSPREVIOUS: The previous version (after rotation)
//   - AWSPENDING: A version being staged during rotation
func (s *Secrets) GetByStage(ctx context.Context, name, stage string) (*Secret, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(name),
		VersionStage: aws.String(stage),
	}

	output, err := s.client.smClient.GetSecretValue(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("secret not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	// Parse JSON secret string
	var data map[string]interface{}
	if output.SecretString != nil {
		if err := json.Unmarshal([]byte(*output.SecretString), &data); err != nil {
			return nil, fmt.Errorf("failed to parse secret data: %w", err)
		}
	}

	secret := &Secret{
		Name:         aws.ToString(output.Name),
		Data:         data,
		ARN:          aws.ToString(output.ARN),
		VersionStage: stage,
	}

	if output.VersionId != nil {
		secret.VersionID = *output.VersionId
	}

	if output.CreatedDate != nil {
		secret.CreatedDate = output.CreatedDate.String()
	}

	return secret, nil
}

// GetByVersionID reads a specific version of a secret by version ID.
func (s *Secrets) GetByVersionID(ctx context.Context, name, versionID string) (*Secret, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:  aws.String(name),
		VersionId: aws.String(versionID),
	}

	output, err := s.client.smClient.GetSecretValue(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("secret version not found: %s@%s", name, versionID)
		}
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	// Parse JSON secret string
	var data map[string]interface{}
	if output.SecretString != nil {
		if err := json.Unmarshal([]byte(*output.SecretString), &data); err != nil {
			return nil, fmt.Errorf("failed to parse secret data: %w", err)
		}
	}

	secret := &Secret{
		Name:      aws.ToString(output.Name),
		Data:      data,
		VersionID: aws.ToString(output.VersionId),
		ARN:       aws.ToString(output.ARN),
	}

	if output.CreatedDate != nil {
		secret.CreatedDate = output.CreatedDate.String()
	}

	return secret, nil
}

// Delete schedules a secret for deletion.
//
// By default, secrets are scheduled for deletion after 30 days.
// Use DeleteImmediately for immediate deletion.
func (s *Secrets) Delete(ctx context.Context, name string) error {
	return s.DeleteWithRecovery(ctx, name, 30)
}

// DeleteWithRecovery schedules a secret for deletion with a custom recovery window.
//
// recoveryDays must be between 7 and 30 (or 0 for immediate deletion).
func (s *Secrets) DeleteWithRecovery(ctx context.Context, name string, recoveryDays int) error {
	input := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(name),
	}

	if recoveryDays == 0 {
		input.ForceDeleteWithoutRecovery = aws.Bool(true)
	} else {
		input.RecoveryWindowInDays = aws.Int64(int64(recoveryDays))
	}

	_, err := s.client.smClient.DeleteSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

// Restore restores a previously deleted secret that is still in recovery.
func (s *Secrets) Restore(ctx context.Context, name string) error {
	input := &secretsmanager.RestoreSecretInput{
		SecretId: aws.String(name),
	}

	_, err := s.client.smClient.RestoreSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to restore secret: %w", err)
	}

	return nil
}

// SecretInfo contains metadata about a secret.
type SecretInfo struct {
	Name            string            `json:"name"`
	ARN             string            `json:"arn"`
	Description     string            `json:"description,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
	LastChangedDate string            `json:"last_changed_date,omitempty"`
	LastAccessedDate string           `json:"last_accessed_date,omitempty"`
	DeletedDate     string            `json:"deleted_date,omitempty"`
	VersionIDs      []string          `json:"version_ids,omitempty"`
}

// List lists secrets with optional prefix filtering.
func (s *Secrets) List(ctx context.Context, prefix string) ([]SecretInfo, error) {
	var secrets []SecretInfo

	input := &secretsmanager.ListSecretsInput{}

	// Add filter if prefix is provided
	if prefix != "" {
		input.Filters = []types.Filter{
			{
				Key:    types.FilterNameStringTypeName,
				Values: []string{prefix},
			},
		}
	}

	paginator := secretsmanager.NewListSecretsPaginator(s.client.smClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, entry := range page.SecretList {
			info := SecretInfo{
				Name: aws.ToString(entry.Name),
				ARN:  aws.ToString(entry.ARN),
			}

			if entry.Description != nil {
				info.Description = *entry.Description
			}

			if len(entry.Tags) > 0 {
				info.Tags = make(map[string]string)
				for _, tag := range entry.Tags {
					if tag.Key != nil && tag.Value != nil {
						info.Tags[*tag.Key] = *tag.Value
					}
				}
			}

			if entry.LastChangedDate != nil {
				info.LastChangedDate = entry.LastChangedDate.String()
			}

			if entry.LastAccessedDate != nil {
				info.LastAccessedDate = entry.LastAccessedDate.String()
			}

			if entry.DeletedDate != nil {
				info.DeletedDate = entry.DeletedDate.String()
			}

			secrets = append(secrets, info)
		}
	}

	return secrets, nil
}

// Describe retrieves metadata about a secret without its value.
func (s *Secrets) Describe(ctx context.Context, name string) (*SecretInfo, error) {
	input := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(name),
	}

	output, err := s.client.smClient.DescribeSecret(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("secret not found: %s", name)
		}
		return nil, fmt.Errorf("failed to describe secret: %w", err)
	}

	info := &SecretInfo{
		Name: aws.ToString(output.Name),
		ARN:  aws.ToString(output.ARN),
	}

	if output.Description != nil {
		info.Description = *output.Description
	}

	if len(output.Tags) > 0 {
		info.Tags = make(map[string]string)
		for _, tag := range output.Tags {
			if tag.Key != nil && tag.Value != nil {
				info.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	if output.LastChangedDate != nil {
		info.LastChangedDate = output.LastChangedDate.String()
	}

	if output.LastAccessedDate != nil {
		info.LastAccessedDate = output.LastAccessedDate.String()
	}

	if output.DeletedDate != nil {
		info.DeletedDate = output.DeletedDate.String()
	}

	// Get version IDs
	if output.VersionIdsToStages != nil {
		for versionID := range output.VersionIdsToStages {
			info.VersionIDs = append(info.VersionIDs, versionID)
		}
	}

	return info, nil
}

// isNotFoundError checks if an error is a "not found" error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for the specific AWS error type
	var notFoundErr *types.ResourceNotFoundException
	if _, ok := err.(*types.ResourceNotFoundException); ok {
		return true
	}
	_ = notFoundErr // Satisfy linter

	// Also check error message as fallback
	errMsg := err.Error()
	return contains(errMsg, "ResourceNotFoundException") ||
		contains(errMsg, "Secrets Manager can't find")
}

// contains checks if s contains substr (simple helper to avoid strings import).
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
