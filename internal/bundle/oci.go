package bundle

import (
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const (
	// BundleLayerMediaType is the OCI media type for Specular bundle layers
	BundleLayerMediaType = "application/vnd.specular.bundle.layer.v1.tar+gzip"
	// BundleConfigMediaType is the OCI media type for Specular bundle configuration
	BundleConfigMediaType = "application/vnd.specular.bundle.config.v1+json"
	// BundleManifestArtifactType is the OCI artifact type for Specular bundles
	BundleManifestArtifactType = "application/vnd.specular.bundle.v1"
)

// OCIOptions configures OCI registry operations
type OCIOptions struct {
	// Reference is the full OCI reference (e.g., ghcr.io/org/bundle:tag)
	Reference string

	// Platform specifies the target platform (defaults to linux/amd64)
	Platform *v1.Platform

	// Insecure allows insecure registries (http instead of https)
	Insecure bool

	// Keychain provides authentication credentials
	Keychain authn.Keychain

	// UserAgent for registry requests
	UserAgent string
}

// OCIPusher handles pushing bundles to OCI registries
type OCIPusher struct {
	opts OCIOptions
}

// NewOCIPusher creates a new OCI pusher
func NewOCIPusher(opts OCIOptions) *OCIPusher {
	if opts.Keychain == nil {
		opts.Keychain = authn.DefaultKeychain
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "specular-bundle/1.0"
	}
	if opts.Platform == nil {
		opts.Platform = &v1.Platform{
			OS:           "linux",
			Architecture: "amd64",
		}
	}

	return &OCIPusher{opts: opts}
}

// Push uploads a bundle to an OCI registry
func (p *OCIPusher) Push(bundlePath string) error {
	// Parse the reference
	ref, parseErr := name.ParseReference(p.opts.Reference)
	if parseErr != nil {
		return WrapRegistryError(parseErr, p.opts.Reference, "push")
	}

	// Get bundle info for metadata
	info, infoErr := GetBundleInfo(bundlePath)
	if infoErr != nil {
		return fmt.Errorf("failed to get bundle info: %w", infoErr)
	}

	// Create layer from bundle tarball
	layer, layerErr := tarball.LayerFromFile(bundlePath, tarball.WithMediaType(BundleLayerMediaType))
	if layerErr != nil {
		return fmt.Errorf("failed to create layer from bundle: %w", layerErr)
	}

	// Start with empty image
	img := empty.Image

	// Add the bundle layer
	var appendErr error
	img, appendErr = mutate.AppendLayers(img, layer)
	if appendErr != nil {
		return fmt.Errorf("failed to append layer: %w", appendErr)
	}

	// Get the current config to preserve DiffIDs
	currentConfig, configErr := img.ConfigFile()
	if configErr != nil {
		return fmt.Errorf("failed to get config: %w", configErr)
	}

	// Update config with bundle metadata while preserving DiffIDs
	configFile := &v1.ConfigFile{
		Architecture: p.opts.Platform.Architecture,
		OS:           p.opts.Platform.OS,
		Config: v1.Config{
			Labels: map[string]string{
				"org.opencontainers.image.title":       info.ID,
				"org.opencontainers.image.version":     info.Version,
				"org.opencontainers.artifact.created":  info.Created.Format("2006-01-02T15:04:05Z"),
				"dev.specular.bundle.schema":           info.Schema,
				"dev.specular.bundle.governance-level": info.GovernanceLevel,
			},
		},
		RootFS: currentConfig.RootFS, // Preserve the DiffIDs from appended layers
	}

	var mutateErr error
	img, mutateErr = mutate.ConfigFile(img, configFile)
	if mutateErr != nil {
		return fmt.Errorf("failed to set config: %w", mutateErr)
	}

	// Set artifact type annotation
	annotated := mutate.Annotations(img, map[string]string{
		"org.opencontainers.image.artifactType": BundleManifestArtifactType,
	})
	var ok bool
	img, ok = annotated.(v1.Image)
	if !ok {
		return fmt.Errorf("failed to assert annotated image to v1.Image")
	}

	// Configure remote options
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(p.opts.Keychain),
		remote.WithUserAgent(p.opts.UserAgent),
		remote.WithPlatform(*p.opts.Platform),
	}

	if p.opts.Insecure {
		remoteOpts = append(remoteOpts, remote.WithTransport(remote.DefaultTransport))
	}

	// Push the image
	if writeErr := remote.Write(ref, img, remoteOpts...); writeErr != nil {
		return WrapRegistryError(writeErr, p.opts.Reference, "push")
	}

	// Get the digest
	digest, digestErr := img.Digest()
	if digestErr != nil {
		return fmt.Errorf("failed to get digest: %w", digestErr)
	}

	fmt.Printf("✓ Pushed bundle to %s\n", ref.String())
	fmt.Printf("  Digest: %s\n", digest.String())

	return nil
}

// OCIPuller handles pulling bundles from OCI registries
type OCIPuller struct {
	opts OCIOptions
}

// NewOCIPuller creates a new OCI puller
func NewOCIPuller(opts OCIOptions) *OCIPuller {
	if opts.Keychain == nil {
		opts.Keychain = authn.DefaultKeychain
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "specular-bundle/1.0"
	}

	return &OCIPuller{opts: opts}
}

// Pull downloads a bundle from an OCI registry
func (p *OCIPuller) Pull(outputPath string) error {
	// Parse reference and fetch image
	ref, img, err := p.fetchBundleImage()
	if err != nil {
		return err
	}

	// Validate manifest
	err = p.validateBundleManifest(img)
	if err != nil {
		return err
	}

	// Extract and save bundle
	err = p.extractBundleToFile(img, outputPath)
	if err != nil {
		return err
	}

	// Get and display digest
	digest, digestErr := img.Digest()
	if digestErr != nil {
		return fmt.Errorf("failed to get digest: %w", digestErr)
	}

	fmt.Printf("✓ Pulled bundle from %s\n", ref.String())
	fmt.Printf("  Digest: %s\n", digest.String())
	fmt.Printf("  Saved to: %s\n", outputPath)

	return nil
}

// fetchBundleImage fetches the bundle image from the registry
func (p *OCIPuller) fetchBundleImage() (name.Reference, v1.Image, error) {
	ref, parseErr := name.ParseReference(p.opts.Reference)
	if parseErr != nil {
		return nil, nil, WrapRegistryError(parseErr, p.opts.Reference, "pull")
	}

	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(p.opts.Keychain),
		remote.WithUserAgent(p.opts.UserAgent),
	}

	if p.opts.Insecure {
		remoteOpts = append(remoteOpts, remote.WithTransport(remote.DefaultTransport))
	}

	img, imgErr := remote.Image(ref, remoteOpts...)
	if imgErr != nil {
		return nil, nil, WrapRegistryError(imgErr, p.opts.Reference, "pull")
	}

	return ref, img, nil
}

// validateBundleManifest validates that the image is a valid Specular bundle
func (p *OCIPuller) validateBundleManifest(img v1.Image) error {
	manifest, manifestErr := img.Manifest()
	if manifestErr != nil {
		return fmt.Errorf("failed to get manifest: %w", manifestErr)
	}

	// Check artifact type annotation
	if err := p.validateArtifactType(manifest); err != nil {
		return err
	}

	// Check layer structure
	if err := p.validateLayerStructure(manifest); err != nil {
		return err
	}

	return nil
}

// validateArtifactType checks if the artifact type matches Specular bundle
func (p *OCIPuller) validateArtifactType(manifest *v1.Manifest) error {
	if manifest.Annotations == nil {
		return nil
	}

	artifactType, ok := manifest.Annotations["org.opencontainers.image.artifactType"]
	if !ok {
		return nil
	}

	if artifactType != BundleManifestArtifactType {
		return &RegistryError{
			Type:    ErrTypeInvalidBundle,
			Message: fmt.Sprintf("Not a Specular bundle: %s", p.opts.Reference),
			Suggestion: fmt.Sprintf(`The artifact has type %q but expected %q.

This appears to be a regular container image, not a Specular bundle.

To create a bundle:
  specular bundle build my-bundle.sbundle.tgz
  specular bundle push my-bundle.sbundle.tgz %s`, artifactType, BundleManifestArtifactType, p.opts.Reference),
			Reference: p.opts.Reference,
		}
	}

	return nil
}

// validateLayerStructure validates the layer count and media type
func (p *OCIPuller) validateLayerStructure(manifest *v1.Manifest) error {
	if len(manifest.Layers) != 1 {
		return &RegistryError{
			Type:    ErrTypeInvalidBundle,
			Message: fmt.Sprintf("Invalid bundle structure: expected 1 layer, got %d", len(manifest.Layers)),
			Suggestion: `Specular bundles must contain exactly one layer (the bundle tarball).

This artifact may have been created incorrectly or corrupted.`,
			Reference: p.opts.Reference,
		}
	}

	bundleLayer := manifest.Layers[0]
	if bundleLayer.MediaType != types.MediaType(BundleLayerMediaType) {
		return &RegistryError{
			Type:    ErrTypeInvalidBundle,
			Message: fmt.Sprintf("Invalid layer media type: expected %s, got %s", BundleLayerMediaType, bundleLayer.MediaType),
			Suggestion: `The layer media type doesn't match Specular bundle format.

This artifact may be a regular OCI artifact or container image.`,
			Reference: p.opts.Reference,
		}
	}

	return nil
}

// extractBundleToFile extracts the bundle layer to an output file
func (p *OCIPuller) extractBundleToFile(img v1.Image, outputPath string) error {
	layers, layersErr := img.Layers()
	if layersErr != nil {
		return fmt.Errorf("failed to get layers: %w", layersErr)
	}

	if len(layers) == 0 {
		return fmt.Errorf("no layers found in image")
	}

	layer := layers[0]

	layerReader, readerErr := layer.Compressed()
	if readerErr != nil {
		return fmt.Errorf("failed to get layer contents: %w", readerErr)
	}
	defer func() {
		if closeErr := layerReader.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close layer reader: %v\n", closeErr)
		}
	}()

	outputFile, createErr := os.Create(outputPath)
	if createErr != nil {
		return fmt.Errorf("failed to create output file: %w", createErr)
	}
	defer func() {
		if closeErr := outputFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close output file: %v\n", closeErr)
		}
	}()

	if _, copyErr := outputFile.ReadFrom(layerReader); copyErr != nil {
		return fmt.Errorf("failed to write bundle: %w", copyErr)
	}

	return nil
}

// GetRemoteBundleInfo retrieves bundle metadata from a registry without downloading
func GetRemoteBundleInfo(ref string, opts OCIOptions) (*BundleInfo, error) {
	if opts.Keychain == nil {
		opts.Keychain = authn.DefaultKeychain
	}

	// Parse reference
	parsedRef, parseErr := name.ParseReference(ref)
	if parseErr != nil {
		return nil, WrapRegistryError(parseErr, ref, "info")
	}

	// Configure remote options
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(opts.Keychain),
	}

	// Get image
	img, imgErr := remote.Image(parsedRef, remoteOpts...)
	if imgErr != nil {
		return nil, WrapRegistryError(imgErr, ref, "info")
	}

	// Get config
	configFile, configErr := img.ConfigFile()
	if configErr != nil {
		return nil, fmt.Errorf("failed to get config: %w", configErr)
	}

	// Extract bundle metadata from labels
	labels := configFile.Config.Labels
	info := &BundleInfo{
		ID:              labels["org.opencontainers.image.title"],
		Version:         labels["org.opencontainers.image.version"],
		Schema:          labels["dev.specular.bundle.schema"],
		GovernanceLevel: labels["dev.specular.bundle.governance-level"],
	}

	// Get digest
	digest, digestErr := img.Digest()
	if digestErr != nil {
		return nil, fmt.Errorf("failed to get digest: %w", digestErr)
	}
	info.IntegrityDigest = digest.String()

	return info, nil
}
