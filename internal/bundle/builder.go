package bundle

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
	"gopkg.in/yaml.v3"
)

const (
	// BundleSchemaVersion is the current bundle schema version
	BundleSchemaVersion = "specular.bundle/v1"

	// DefaultChecksumAlgorithm is the default hash algorithm for checksums
	DefaultChecksumAlgorithm = "sha256"

	// ManifestFileName is the name of the manifest file in bundles
	ManifestFileName = "manifest.yaml"

	// ChecksumsFileName is the name of the checksums file in bundles
	ChecksumsFileName = "checksums.txt"
)

// Builder creates governance bundles from project files.
type Builder struct {
	opts   BundleOptions
	bundle *Bundle
}

// NewBuilder creates a new bundle builder with the given options.
func NewBuilder(opts BundleOptions) (*Builder, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid bundle options: %w", err)
	}

	return &Builder{
		opts: opts,
		bundle: &Bundle{
			Manifest:        &Manifest{},
			Checksums:       make(map[string]string),
			AdditionalFiles: make(map[string][]byte),
		},
	}, nil
}

// Build creates a bundle and writes it to the specified output path.
func (b *Builder) Build(outputPath string) error {
	// Load project files
	if err := b.loadProjectFiles(); err != nil {
		return fmt.Errorf("failed to load project files: %w", err)
	}

	// Create manifest
	if err := b.createManifest(); err != nil {
		return fmt.Errorf("failed to create manifest: %w", err)
	}

	// Calculate checksums for all files
	if err := b.calculateChecksums(); err != nil {
		return fmt.Errorf("failed to calculate checksums: %w", err)
	}

	// Create tarball
	if err := b.createTarball(outputPath); err != nil {
		return fmt.Errorf("failed to create bundle tarball: %w", err)
	}

	return nil
}

// loadProjectFiles loads all project files into the bundle.
func (b *Builder) loadProjectFiles() error {
	// Load spec.yaml
	if b.opts.SpecPath != "" {
		if err := b.loadSpec(); err != nil {
			return fmt.Errorf("failed to load spec: %w", err)
		}
	}

	// Load spec.lock.json
	if b.opts.LockPath != "" {
		if err := b.loadSpecLock(); err != nil {
			return fmt.Errorf("failed to load spec lock: %w", err)
		}
	}

	// Load routing.yaml
	if b.opts.RoutingPath != "" {
		if err := b.loadRouting(); err != nil {
			return fmt.Errorf("failed to load routing: %w", err)
		}
	}

	// Load policies
	if len(b.opts.PolicyPaths) > 0 {
		if err := b.loadPolicies(); err != nil {
			return fmt.Errorf("failed to load policies: %w", err)
		}
	}

	// Load additional files
	if len(b.opts.IncludePaths) > 0 {
		if err := b.loadAdditionalFiles(); err != nil {
			return fmt.Errorf("failed to load additional files: %w", err)
		}
	}

	return nil
}

// loadSpec loads the product specification.
func (b *Builder) loadSpec() error {
	data, err := os.ReadFile(b.opts.SpecPath)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	var productSpec spec.ProductSpec
	if unmarshalErr := yaml.Unmarshal(data, &productSpec); unmarshalErr != nil {
		return fmt.Errorf("failed to parse spec file: %w", unmarshalErr)
	}

	b.bundle.Spec = &productSpec
	return nil
}

// loadSpecLock loads the specification lock file.
func (b *Builder) loadSpecLock() error {
	data, err := os.ReadFile(b.opts.LockPath)
	if err != nil {
		return fmt.Errorf("failed to read spec lock file: %w", err)
	}

	var specLock spec.SpecLock
	if unmarshalErr := json.Unmarshal(data, &specLock); unmarshalErr != nil {
		return fmt.Errorf("failed to parse spec lock file: %w", unmarshalErr)
	}

	b.bundle.SpecLock = &specLock
	return nil
}

// loadRouting loads the routing configuration.
func (b *Builder) loadRouting() error {
	data, err := os.ReadFile(b.opts.RoutingPath)
	if err != nil {
		return fmt.Errorf("failed to read routing file: %w", err)
	}

	var routerConfig router.Router
	if unmarshalErr := yaml.Unmarshal(data, &routerConfig); unmarshalErr != nil {
		return fmt.Errorf("failed to parse routing file: %w", unmarshalErr)
	}

	b.bundle.Routing = &routerConfig
	return nil
}

// loadPolicies loads all policy files.
func (b *Builder) loadPolicies() error {
	policies := make([]*policy.Policy, 0, len(b.opts.PolicyPaths))

	for _, policyPath := range b.opts.PolicyPaths {
		data, err := os.ReadFile(policyPath)
		if err != nil {
			return fmt.Errorf("failed to read policy file %s: %w", policyPath, err)
		}

		var pol policy.Policy
		if unmarshalErr := yaml.Unmarshal(data, &pol); unmarshalErr != nil {
			return fmt.Errorf("failed to parse policy file %s: %w", policyPath, unmarshalErr)
		}

		policies = append(policies, &pol)
	}

	b.bundle.Policies = policies
	return nil
}

// loadAdditionalFiles loads additional files specified in include paths.
func (b *Builder) loadAdditionalFiles() error {
	for _, includePath := range b.opts.IncludePaths {
		info, err := os.Stat(includePath)
		if err != nil {
			return fmt.Errorf("failed to stat include path %s: %w", includePath, err)
		}

		if info.IsDir() {
			if loadErr := b.loadDirectory(includePath); loadErr != nil {
				return fmt.Errorf("failed to load directory %s: %w", includePath, loadErr)
			}
		} else {
			if loadErr := b.loadFile(includePath); loadErr != nil {
				return fmt.Errorf("failed to load file %s: %w", includePath, loadErr)
			}
		}
	}

	return nil
}

// loadDirectory recursively loads all files from a directory.
func (b *Builder) loadDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return b.loadFile(path)
		}

		return nil
	})
}

// loadFile loads a single file into additional files.
func (b *Builder) loadFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Use relative path as key
	relPath, err := filepath.Rel(".", filePath)
	if err != nil {
		relPath = filePath
	}

	b.bundle.AdditionalFiles[relPath] = data
	return nil
}

// createManifest creates the bundle manifest with metadata.
func (b *Builder) createManifest() error {
	// Set bundle ID and version from spec
	bundleID := "unknown/bundle"
	bundleVersion := "0.0.0"

	if b.bundle.Spec != nil {
		if b.bundle.Spec.Product != "" {
			bundleID = b.bundle.Spec.Product
		}
	}

	if b.bundle.SpecLock != nil {
		if b.bundle.SpecLock.Version != "" {
			bundleVersion = b.bundle.SpecLock.Version
		}
	}

	b.bundle.Manifest = &Manifest{
		Schema:            BundleSchemaVersion,
		ID:                bundleID,
		Version:           bundleVersion,
		Created:           time.Now(),
		GovernanceLevel:   b.opts.GovernanceLevel,
		RequiredApprovals: b.opts.RequireApprovals,
		Metadata:          b.opts.Metadata,
		Files:             []FileEntry{},
	}

	return nil
}

// calculateChecksums calculates SHA-256 checksums for all files in the bundle.
func (b *Builder) calculateChecksums() error {
	fileEntries := []FileEntry{}

	// Checksum spec file
	if b.opts.SpecPath != "" {
		entry, err := b.checksumFile(b.opts.SpecPath, "spec.yaml")
		if err != nil {
			return err
		}
		fileEntries = append(fileEntries, *entry)
		b.bundle.Checksums["spec.yaml"] = entry.Checksum
	}

	// Checksum lock file
	if b.opts.LockPath != "" {
		entry, err := b.checksumFile(b.opts.LockPath, "spec.lock.json")
		if err != nil {
			return err
		}
		fileEntries = append(fileEntries, *entry)
		b.bundle.Checksums["spec.lock.json"] = entry.Checksum
	}

	// Checksum routing file
	if b.opts.RoutingPath != "" {
		entry, err := b.checksumFile(b.opts.RoutingPath, "routing.yaml")
		if err != nil {
			return err
		}
		fileEntries = append(fileEntries, *entry)
		b.bundle.Checksums["routing.yaml"] = entry.Checksum
	}

	// Checksum policy files
	for i, policyPath := range b.opts.PolicyPaths {
		bundlePath := fmt.Sprintf("policies/policy_%d.yaml", i)
		entry, err := b.checksumFile(policyPath, bundlePath)
		if err != nil {
			return err
		}
		fileEntries = append(fileEntries, *entry)
		b.bundle.Checksums[bundlePath] = entry.Checksum
	}

	// Checksum additional files
	for path, data := range b.bundle.AdditionalFiles {
		checksum := sha256.Sum256(data)
		checksumHex := hex.EncodeToString(checksum[:])

		fileEntries = append(fileEntries, FileEntry{
			Path:     path,
			Size:     int64(len(data)),
			Checksum: checksumHex,
		})
		b.bundle.Checksums[path] = checksumHex
	}

	b.bundle.Manifest.Files = fileEntries

	// Calculate manifest integrity digest
	manifestData, err := yaml.Marshal(b.bundle.Manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestDigest := sha256.Sum256(manifestData)
	digestHex := hex.EncodeToString(manifestDigest[:])

	b.bundle.Manifest.Integrity = IntegrityInfo{
		Algorithm:      DefaultChecksumAlgorithm,
		Digest:         fmt.Sprintf("%s:%s", DefaultChecksumAlgorithm, digestHex),
		ManifestDigest: digestHex,
	}

	return nil
}

// checksumFile calculates the checksum for a file.
func (b *Builder) checksumFile(filePath, bundlePath string) (*FileEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close file %s: %w", filePath, closeErr)
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	hash := sha256.New()
	if _, copyErr := io.Copy(hash, file); copyErr != nil {
		return nil, fmt.Errorf("failed to hash file %s: %w", filePath, copyErr)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	return &FileEntry{
		Path:     bundlePath,
		Size:     info.Size(),
		Checksum: checksum,
		Mode:     uint32(info.Mode()),
	}, nil
}

// createTarball creates the final .sbundle.tgz archive.
func (b *Builder) createTarball(outputPath string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close output file: %w", closeErr)
		}
	}()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer func() {
		if closeErr := gzWriter.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close gzip writer: %w", closeErr)
		}
	}()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer func() {
		if closeErr := tarWriter.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close tar writer: %w", closeErr)
		}
	}()

	// Write manifest
	if writeErr := b.writeManifestToTar(tarWriter); writeErr != nil {
		return fmt.Errorf("failed to write manifest: %w", writeErr)
	}

	// Write spec file
	if b.opts.SpecPath != "" {
		if writeErr := b.writeFileToTar(tarWriter, b.opts.SpecPath, "spec.yaml"); writeErr != nil {
			return fmt.Errorf("failed to write spec: %w", writeErr)
		}
	}

	// Write lock file
	if b.opts.LockPath != "" {
		if writeErr := b.writeFileToTar(tarWriter, b.opts.LockPath, "spec.lock.json"); writeErr != nil {
			return fmt.Errorf("failed to write lock: %w", writeErr)
		}
	}

	// Write routing file
	if b.opts.RoutingPath != "" {
		if writeErr := b.writeFileToTar(tarWriter, b.opts.RoutingPath, "routing.yaml"); writeErr != nil {
			return fmt.Errorf("failed to write routing: %w", writeErr)
		}
	}

	// Write policy files
	for i, policyPath := range b.opts.PolicyPaths {
		bundlePath := fmt.Sprintf("policies/policy_%d.yaml", i)
		if writeErr := b.writeFileToTar(tarWriter, policyPath, bundlePath); writeErr != nil {
			return fmt.Errorf("failed to write policy: %w", writeErr)
		}
	}

	// Write additional files
	for path, data := range b.bundle.AdditionalFiles {
		if writeErr := b.writeBytesToTar(tarWriter, data, path); writeErr != nil {
			return fmt.Errorf("failed to write additional file %s: %w", path, writeErr)
		}
	}

	// Write checksums file
	if writeErr := b.writeChecksumsToTar(tarWriter); writeErr != nil {
		return fmt.Errorf("failed to write checksums: %w", writeErr)
	}

	return nil
}

// writeManifestToTar writes the manifest to the tar archive.
func (b *Builder) writeManifestToTar(tw *tar.Writer) error {
	manifestData, err := yaml.Marshal(b.bundle.Manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	header := &tar.Header{
		Name:    ManifestFileName,
		Size:    int64(len(manifestData)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := tw.Write(manifestData); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// writeFileToTar writes a file from the filesystem to the tar archive.
func (b *Builder) writeFileToTar(tw *tar.Writer, sourcePath, bundlePath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	header := &tar.Header{
		Name:    bundlePath,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := io.Copy(tw, file); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// writeBytesToTar writes byte data to the tar archive.
func (b *Builder) writeBytesToTar(tw *tar.Writer, data []byte, bundlePath string) error {
	header := &tar.Header{
		Name:    bundlePath,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// writeChecksumsToTar writes the checksums file to the tar archive.
func (b *Builder) writeChecksumsToTar(tw *tar.Writer) error {
	var checksumData string
	for path, checksum := range b.bundle.Checksums {
		checksumData += fmt.Sprintf("%s  %s\n", checksum, path)
	}

	header := &tar.Header{
		Name:    ChecksumsFileName,
		Size:    int64(len(checksumData)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := tw.Write([]byte(checksumData)); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// Validate validates bundle options.
func (opts *BundleOptions) Validate() error {
	// At least one input file is required
	if opts.SpecPath == "" && opts.LockPath == "" && opts.RoutingPath == "" &&
		len(opts.PolicyPaths) == 0 && len(opts.IncludePaths) == 0 {
		return fmt.Errorf("at least one input file must be specified")
	}

	return nil
}
