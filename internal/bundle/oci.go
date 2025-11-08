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
	// OCI media types for Specular bundles
	BundleLayerMediaType      = "application/vnd.specular.bundle.layer.v1.tar+gzip"
	BundleConfigMediaType     = "application/vnd.specular.bundle.config.v1+json"
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
	ref, err := name.ParseReference(p.opts.Reference)
	if err != nil {
		return WrapRegistryError(err, p.opts.Reference, "push")
	}

	// Get bundle info for metadata
	info, err := GetBundleInfo(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to get bundle info: %w", err)
	}

	// Create layer from bundle tarball
	layer, err := tarball.LayerFromFile(bundlePath, tarball.WithMediaType(BundleLayerMediaType))
	if err != nil {
		return fmt.Errorf("failed to create layer from bundle: %w", err)
	}

	// Start with empty image
	img := empty.Image

	// Add the bundle layer
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		return fmt.Errorf("failed to append layer: %w", err)
	}

	// Get the current config to preserve DiffIDs
	currentConfig, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
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

	img, err = mutate.ConfigFile(img, configFile)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// Set artifact type annotation
	img = mutate.Annotations(img, map[string]string{
		"org.opencontainers.image.artifactType": BundleManifestArtifactType,
	}).(v1.Image)

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
	if err := remote.Write(ref, img, remoteOpts...); err != nil {
		return WrapRegistryError(err, p.opts.Reference, "push")
	}

	// Get the digest
	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("failed to get digest: %w", err)
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
	// Parse the reference
	ref, err := name.ParseReference(p.opts.Reference)
	if err != nil {
		return WrapRegistryError(err, p.opts.Reference, "pull")
	}

	// Configure remote options
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(p.opts.Keychain),
		remote.WithUserAgent(p.opts.UserAgent),
	}

	if p.opts.Insecure {
		remoteOpts = append(remoteOpts, remote.WithTransport(remote.DefaultTransport))
	}

	// Pull the image
	img, err := remote.Image(ref, remoteOpts...)
	if err != nil {
		return WrapRegistryError(err, p.opts.Reference, "pull")
	}

	// Verify it's a bundle artifact
	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	// Check artifact type annotation
	if manifest.Annotations != nil {
		if artifactType, ok := manifest.Annotations["org.opencontainers.image.artifactType"]; ok {
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
		}
	}

	// Check media type of layers
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

	// Get the layers
	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("failed to get layers: %w", err)
	}

	if len(layers) == 0 {
		return fmt.Errorf("no layers found in image")
	}

	// Extract the first layer (the bundle tarball)
	layer := layers[0]

	// Get layer contents
	layerReader, err := layer.Compressed()
	if err != nil {
		return fmt.Errorf("failed to get layer contents: %w", err)
	}
	defer layerReader.Close()

	// Write to output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Copy layer contents to output file
	if _, err := outputFile.ReadFrom(layerReader); err != nil {
		return fmt.Errorf("failed to write bundle: %w", err)
	}

	// Get the digest
	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("failed to get digest: %w", err)
	}

	fmt.Printf("✓ Pulled bundle from %s\n", ref.String())
	fmt.Printf("  Digest: %s\n", digest.String())
	fmt.Printf("  Saved to: %s\n", outputPath)

	return nil
}

// GetRemoteBundleInfo retrieves bundle metadata from a registry without downloading
func GetRemoteBundleInfo(ref string, opts OCIOptions) (*BundleInfo, error) {
	if opts.Keychain == nil {
		opts.Keychain = authn.DefaultKeychain
	}

	// Parse reference
	parsedRef, err := name.ParseReference(ref)
	if err != nil {
		return nil, WrapRegistryError(err, ref, "info")
	}

	// Configure remote options
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(opts.Keychain),
	}

	// Get image
	img, err := remote.Image(parsedRef, remoteOpts...)
	if err != nil {
		return nil, WrapRegistryError(err, ref, "info")
	}

	// Get config
	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
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
	digest, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("failed to get digest: %w", err)
	}
	info.IntegrityDigest = digest.String()

	return info, nil
}
