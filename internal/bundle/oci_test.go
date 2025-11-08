package bundle

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRegistry creates a local registry server for testing
func setupTestRegistry(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	// Create a registry handler
	regHandler := registry.New()

	// Start test server
	server := httptest.NewServer(regHandler)
	t.Cleanup(server.Close)

	// Parse URL to get host
	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	return server, u.Host
}

// createTestBundle creates a test bundle for registry operations
func createTestBundle(t *testing.T) (string, string) {
	t.Helper()

	tempDir := t.TempDir()

	// Create test spec
	specPath := filepath.Join(tempDir, "spec.yaml")
	specContent := `product: oci-test-bundle
goals:
  - Test OCI registry operations
features: []
non_functional:
  performance: []
  security: []
  scalability: []
acceptance: []
milestones: []
`
	require.NoError(t, os.WriteFile(specPath, []byte(specContent), 0600))

	// Create test lock file
	lockPath := filepath.Join(tempDir, "spec.lock.json")
	lockContent := `{
  "version": "1.0.0",
  "spec_hash": "test-hash-123",
  "locked_at": "2024-01-01T00:00:00Z"
}
`
	require.NoError(t, os.WriteFile(lockPath, []byte(lockContent), 0600))

	// Create test routing config
	routingPath := filepath.Join(tempDir, "routing.yaml")
	routingContent := `default_model: gpt-4
fallback_models:
  - gpt-3.5-turbo
`
	require.NoError(t, os.WriteFile(routingPath, []byte(routingContent), 0600))

	// Build bundle
	opts := BundleOptions{
		SpecPath:        specPath,
		LockPath:        lockPath,
		RoutingPath:     routingPath,
		GovernanceLevel: "L2",
	}

	builder, err := NewBuilder(opts)
	require.NoError(t, err)

	bundlePath := filepath.Join(tempDir, "test.sbundle.tgz")
	err = builder.Build(bundlePath)
	require.NoError(t, err)

	return bundlePath, tempDir
}

// TestOCIPushPull tests the complete push/pull workflow
func TestOCIPushPull(t *testing.T) {
	// Setup test registry
	_, registryHost := setupTestRegistry(t)

	// Create test bundle
	bundlePath, tempDir := createTestBundle(t)

	// Push bundle to registry
	ref := fmt.Sprintf("%s/test/bundle:v1.0.0", registryHost)

	pushOpts := OCIOptions{
		Reference: ref,
		Insecure:  true,
		Keychain:  authn.DefaultKeychain,
	}

	pusher := NewOCIPusher(pushOpts)
	err := pusher.Push(bundlePath)
	require.NoError(t, err, "Push should succeed")

	// Pull bundle from registry
	pullPath := filepath.Join(tempDir, "pulled.sbundle.tgz")

	pullOpts := OCIOptions{
		Reference: ref,
		Insecure:  true,
		Keychain:  authn.DefaultKeychain,
	}

	puller := NewOCIPuller(pullOpts)
	err = puller.Pull(pullPath)
	require.NoError(t, err, "Pull should succeed")

	// Verify pulled bundle
	stat, err := os.Stat(pullPath)
	require.NoError(t, err)
	assert.Greater(t, stat.Size(), int64(0))

	// Verify bundle integrity
	validator := NewValidator(VerifyOptions{})
	result, err := validator.Verify(pullPath)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.True(t, result.ChecksumValid)
}

// TestOCIPushPullWithPlatform tests platform-specific push/pull
func TestOCIPushPullWithPlatform(t *testing.T) {
	// Setup test registry
	_, registryHost := setupTestRegistry(t)

	// Create test bundle
	bundlePath, tempDir := createTestBundle(t)

	// Push with platform specification
	ref := fmt.Sprintf("%s/test/platform-bundle:v1.0.0", registryHost)

	pushOpts := OCIOptions{
		Reference: ref,
		Insecure:  true,
		Platform: &v1.Platform{
			OS:           "linux",
			Architecture: "arm64",
		},
	}

	pusher := NewOCIPusher(pushOpts)
	err := pusher.Push(bundlePath)
	require.NoError(t, err)

	// Pull bundle
	pullPath := filepath.Join(tempDir, "pulled-platform.sbundle.tgz")

	pullOpts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	puller := NewOCIPuller(pullOpts)
	err = puller.Pull(pullPath)
	require.NoError(t, err)

	// Verify pulled bundle
	validator := NewValidator(VerifyOptions{})
	result, err := validator.Verify(pullPath)
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

// TestGetRemoteBundleInfo tests metadata retrieval without downloading
func TestGetRemoteBundleInfo(t *testing.T) {
	// Setup test registry
	_, registryHost := setupTestRegistry(t)

	// Create and push test bundle
	bundlePath, _ := createTestBundle(t)

	ref := fmt.Sprintf("%s/test/info-bundle:v1.0.0", registryHost)

	pushOpts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	pusher := NewOCIPusher(pushOpts)
	err := pusher.Push(bundlePath)
	require.NoError(t, err)

	// Get bundle info without downloading
	opts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	info, err := GetRemoteBundleInfo(ref, opts)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "oci-test-bundle", info.ID)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "L2", info.GovernanceLevel)
	assert.NotEmpty(t, info.IntegrityDigest)
}

// TestOCIPullNotFound tests error handling for missing bundles
func TestOCIPullNotFound(t *testing.T) {
	// Setup test registry
	_, registryHost := setupTestRegistry(t)

	tempDir := t.TempDir()
	pullPath := filepath.Join(tempDir, "not-found.sbundle.tgz")

	// Try to pull non-existent bundle
	ref := fmt.Sprintf("%s/test/does-not-exist:v1.0.0", registryHost)

	opts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	puller := NewOCIPuller(opts)
	err := puller.Pull(pullPath)
	require.Error(t, err)

	// Verify it's a registry error
	var regErr *RegistryError
	require.ErrorAs(t, err, &regErr)
	assert.Equal(t, ErrTypeNotFound, regErr.Type)
}

// TestOCIPullInvalidBundle tests rejection of non-bundle artifacts
func TestOCIPullInvalidBundle(t *testing.T) {
	// Setup test registry
	_, registryHost := setupTestRegistry(t)

	// Create and push a regular container image (not a bundle)
	ref := fmt.Sprintf("%s/test/not-a-bundle:v1.0.0", registryHost)

	parsedRef, err := name.ParseReference(ref, name.Insecure)
	require.NoError(t, err)

	// Create a regular image
	img, err := random.Image(1024, 1)
	require.NoError(t, err)

	// Push the regular image
	err = remote.Write(parsedRef, img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(t, err)

	// Try to pull as a bundle
	tempDir := t.TempDir()
	pullPath := filepath.Join(tempDir, "invalid.sbundle.tgz")

	opts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	puller := NewOCIPuller(opts)
	err = puller.Pull(pullPath)
	require.Error(t, err)

	// Verify it's an invalid bundle error
	var regErr *RegistryError
	require.ErrorAs(t, err, &regErr)
	assert.Equal(t, ErrTypeInvalidBundle, regErr.Type)
	// Error message could be about artifact type or layer media type
	// Just verify it's categorized as invalid bundle
}

// TestOCIPullInvalidLayerMediaType tests rejection of wrong media type
func TestOCIPullInvalidLayerMediaType(t *testing.T) {
	// Setup test registry
	_, registryHost := setupTestRegistry(t)

	// Create image with correct artifact type but wrong layer media type
	ref := fmt.Sprintf("%s/test/wrong-media-type:v1.0.0", registryHost)

	parsedRef, err := name.ParseReference(ref, name.Insecure)
	require.NoError(t, err)

	// Create image with wrong layer media type
	layer, err := random.Layer(1024, "application/vnd.oci.image.layer.v1.tar+gzip")
	require.NoError(t, err)

	img, err := mutate.AppendLayers(empty.Image, layer)
	require.NoError(t, err)

	// Set the correct artifact type annotation
	img = mutate.Annotations(img, map[string]string{
		"org.opencontainers.image.artifactType": BundleManifestArtifactType,
	}).(v1.Image)

	// Push the image
	err = remote.Write(parsedRef, img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(t, err)

	// Try to pull as a bundle
	tempDir := t.TempDir()
	pullPath := filepath.Join(tempDir, "wrong-media.sbundle.tgz")

	opts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	puller := NewOCIPuller(opts)
	err = puller.Pull(pullPath)
	require.Error(t, err)

	// Verify it's an invalid bundle error
	var regErr *RegistryError
	require.ErrorAs(t, err, &regErr)
	assert.Equal(t, ErrTypeInvalidBundle, regErr.Type)
	assert.Contains(t, regErr.Message, "Invalid layer media type")
}

// TestOCIPullMultipleLayers tests rejection of multi-layer artifacts
func TestOCIPullMultipleLayers(t *testing.T) {
	// Setup test registry
	_, registryHost := setupTestRegistry(t)

	// Create image with multiple layers
	ref := fmt.Sprintf("%s/test/multi-layer:v1.0.0", registryHost)

	parsedRef, err := name.ParseReference(ref, name.Insecure)
	require.NoError(t, err)

	// Create image with multiple layers
	layer1, err := random.Layer(1024, BundleLayerMediaType)
	require.NoError(t, err)
	layer2, err := random.Layer(1024, BundleLayerMediaType)
	require.NoError(t, err)

	img, err := mutate.AppendLayers(empty.Image, layer1, layer2)
	require.NoError(t, err)

	// Set the correct artifact type annotation
	img = mutate.Annotations(img, map[string]string{
		"org.opencontainers.image.artifactType": BundleManifestArtifactType,
	}).(v1.Image)

	// Push the image
	err = remote.Write(parsedRef, img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(t, err)

	// Try to pull as a bundle
	tempDir := t.TempDir()
	pullPath := filepath.Join(tempDir, "multi-layer.sbundle.tgz")

	opts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	puller := NewOCIPuller(opts)
	err = puller.Pull(pullPath)
	require.Error(t, err)

	// Verify it's an invalid bundle error
	var regErr *RegistryError
	require.ErrorAs(t, err, &regErr)
	assert.Equal(t, ErrTypeInvalidBundle, regErr.Type)
	assert.Contains(t, regErr.Message, "expected 1 layer")
}

// TestOCIPushInvalidReference tests error handling for invalid references
func TestOCIPushInvalidReference(t *testing.T) {
	bundlePath, _ := createTestBundle(t)

	// Try to push with invalid reference
	opts := OCIOptions{
		Reference: "invalid reference with spaces",
	}

	pusher := NewOCIPusher(opts)
	err := pusher.Push(bundlePath)
	require.Error(t, err)

	// Verify it's an invalid reference error
	var regErr *RegistryError
	require.ErrorAs(t, err, &regErr)
	assert.Equal(t, ErrTypeInvalidRef, regErr.Type)
}

// TestOCIPullInvalidReference tests error handling for invalid pull references
func TestOCIPullInvalidReference(t *testing.T) {
	tempDir := t.TempDir()
	pullPath := filepath.Join(tempDir, "invalid-ref.sbundle.tgz")

	// Try to pull with invalid reference
	opts := OCIOptions{
		Reference: "not a valid:reference:format",
	}

	puller := NewOCIPuller(opts)
	err := puller.Pull(pullPath)
	require.Error(t, err)

	// Verify it's an invalid reference error
	var regErr *RegistryError
	require.ErrorAs(t, err, &regErr)
	assert.Equal(t, ErrTypeInvalidRef, regErr.Type)
}

// TestOCIPusherDefaults tests default values are set correctly
func TestOCIPusherDefaults(t *testing.T) {
	opts := OCIOptions{
		Reference: "example.com/repo:tag",
	}

	pusher := NewOCIPusher(opts)
	assert.NotNil(t, pusher.opts.Keychain, "Should set default keychain")
	assert.Equal(t, "specular-bundle/1.0", pusher.opts.UserAgent, "Should set default user agent")
	assert.NotNil(t, pusher.opts.Platform, "Should set default platform")
	assert.Equal(t, "linux", pusher.opts.Platform.OS)
	assert.Equal(t, "amd64", pusher.opts.Platform.Architecture)
}

// TestOCIPullerDefaults tests default values are set correctly
func TestOCIPullerDefaults(t *testing.T) {
	opts := OCIOptions{
		Reference: "example.com/repo:tag",
	}

	puller := NewOCIPuller(opts)
	assert.NotNil(t, puller.opts.Keychain, "Should set default keychain")
	assert.Equal(t, "specular-bundle/1.0", puller.opts.UserAgent, "Should set default user agent")
}

// TestBundleRoundTrip tests complete bundle lifecycle through registry
func TestBundleRoundTrip(t *testing.T) {
	// Setup test registry
	_, registryHost := setupTestRegistry(t)

	// Create test bundle with metadata
	bundlePath, tempDir := createTestBundle(t)

	// Get original bundle info
	originalInfo, err := GetBundleInfo(bundlePath)
	require.NoError(t, err)

	// Push to registry
	ref := fmt.Sprintf("%s/test/roundtrip:v1.0.0", registryHost)

	pushOpts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	pusher := NewOCIPusher(pushOpts)
	err = pusher.Push(bundlePath)
	require.NoError(t, err)

	// Get remote info (without downloading)
	remoteInfo, err := GetRemoteBundleInfo(ref, OCIOptions{Insecure: true})
	require.NoError(t, err)
	assert.Equal(t, originalInfo.ID, remoteInfo.ID)
	assert.Equal(t, originalInfo.Version, remoteInfo.Version)
	assert.Equal(t, originalInfo.GovernanceLevel, remoteInfo.GovernanceLevel)

	// Pull from registry
	pullPath := filepath.Join(tempDir, "roundtrip.sbundle.tgz")

	pullOpts := OCIOptions{
		Reference: ref,
		Insecure:  true,
	}

	puller := NewOCIPuller(pullOpts)
	err = puller.Pull(pullPath)
	require.NoError(t, err)

	// Get pulled bundle info
	pulledInfo, err := GetBundleInfo(pullPath)
	require.NoError(t, err)

	// Verify all metadata matches
	assert.Equal(t, originalInfo.ID, pulledInfo.ID)
	assert.Equal(t, originalInfo.Version, pulledInfo.Version)
	assert.Equal(t, originalInfo.GovernanceLevel, pulledInfo.GovernanceLevel)
	assert.Equal(t, originalInfo.Schema, pulledInfo.Schema)

	// Verify bundle integrity
	validator := NewValidator(VerifyOptions{})
	result, err := validator.Verify(pullPath)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.True(t, result.ChecksumValid)
}
