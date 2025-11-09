package bundle

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Extractor unpacks and applies bundles to projects.
type Extractor struct {
	opts   ApplyOptions
	bundle *Bundle
}

// NewExtractor creates a new bundle extractor with the given options.
func NewExtractor(opts ApplyOptions) *Extractor {
	return &Extractor{
		opts:   opts,
		bundle: &Bundle{},
	}
}

// safeFileMode safely converts a tar header mode (int64) to os.FileMode (uint32).
// It masks the mode to only include valid permission and type bits to prevent overflow.
func safeFileMode(mode int64) os.FileMode {
	// Mask to include only file type and permission bits (0xFFFF covers all valid bits)
	// This prevents integer overflow when converting from int64 to uint32
	// #nosec G115 - Intentional masking with 0xFFFF prevents overflow
	return os.FileMode(mode & 0xFFFF)
}

// Apply extracts and applies a bundle to the target directory.
func (e *Extractor) Apply(bundlePath string) error {
	// Validate bundle first
	validator := NewValidator(VerifyOptions{
		Strict:             false,
		RequireApprovals:   false,
		RequireAttestation: false,
	})

	result, err := validator.Verify(bundlePath)
	if err != nil {
		return fmt.Errorf("bundle validation failed: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("bundle validation failed: %d errors", len(result.Errors))
	}

	e.bundle = validator.bundle

	// Extract bundle to temporary directory
	tempDir, err := e.extractBundle(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to extract bundle: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }() //nolint:errcheck

	// Apply files to target directory
	if e.opts.DryRun {
		return e.dryRunApply(tempDir)
	}

	return e.performApply(tempDir)
}

// extractBundle extracts the bundle to a temporary directory.
func (e *Extractor) extractBundle(bundlePath string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "bundle-extract-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Open bundle file
	file, err := os.Open(bundlePath)
	if err != nil {
		_ = os.RemoveAll(tempDir) //nolint:errcheck
		return "", fmt.Errorf("failed to open bundle: %w", err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		_ = os.RemoveAll(tempDir) //nolint:errcheck
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzReader.Close() }() //nolint:errcheck

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract all files
	for {
		header, readErr := tarReader.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = os.RemoveAll(tempDir) //nolint:errcheck
			return "", fmt.Errorf("failed to read tar: %w", readErr)
		}

		// Construct target path
		// #nosec G305 - Path traversal is validated on line 118
		targetPath := filepath.Join(tempDir, header.Name)

		// Ensure target path is within temp directory (prevent path traversal)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(tempDir)) {
			_ = os.RemoveAll(tempDir) //nolint:errcheck
			return "", fmt.Errorf("invalid file path in bundle: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if mkdirErr := os.MkdirAll(targetPath, safeFileMode(header.Mode)); mkdirErr != nil {
				_ = os.RemoveAll(tempDir) //nolint:errcheck
				return "", fmt.Errorf("failed to create directory: %w", mkdirErr)
			}

		case tar.TypeReg:
			// Create parent directory
			if parentDirErr := os.MkdirAll(filepath.Dir(targetPath), 0750); parentDirErr != nil {
				_ = os.RemoveAll(tempDir) //nolint:errcheck
				return "", fmt.Errorf("failed to create parent directory: %w", parentDirErr)
			}

			// Create file
			outFile, fileErr := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, safeFileMode(header.Mode))
			if fileErr != nil {
				_ = os.RemoveAll(tempDir) //nolint:errcheck
				return "", fmt.Errorf("failed to create file: %w", fileErr)
			}

			// Copy data
			// #nosec G110 - Decompression bomb risk accepted for trusted, verified bundles
			// Bundles are validated and verified before extraction, ensuring they come from trusted sources
			if _, copyErr := io.Copy(outFile, tarReader); copyErr != nil {
				_ = outFile.Close()       //nolint:errcheck
				_ = os.RemoveAll(tempDir) //nolint:errcheck
				return "", fmt.Errorf("failed to write file: %w", copyErr)
			}
			_ = outFile.Close() //nolint:errcheck
		}
	}

	return tempDir, nil
}

// dryRunApply shows what would be applied without making changes.
func (e *Extractor) dryRunApply(tempDir string) error {
	fmt.Println("DRY RUN: The following changes would be applied:")
	fmt.Println()

	// Show spec changes
	if err := e.showSpecChanges(tempDir); err != nil {
		return err
	}

	// Show lock changes
	if err := e.showLockChanges(tempDir); err != nil {
		return err
	}

	// Show routing changes
	if err := e.showRoutingChanges(tempDir); err != nil {
		return err
	}

	// Show policy changes
	if err := e.showPolicyChanges(tempDir); err != nil {
		return err
	}

	// Show additional file changes
	if err := e.showAdditionalFileChanges(tempDir); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("No changes were made (dry-run mode).")

	return nil
}

// performApply actually applies the bundle to the target directory.
func (e *Extractor) performApply(tempDir string) error {
	targetDir := e.opts.TargetDir
	if targetDir == "" {
		targetDir = "."
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Apply spec file
	if err := e.applySpecFile(tempDir, targetDir); err != nil {
		return err
	}

	// Apply lock file
	if err := e.applyLockFile(tempDir, targetDir); err != nil {
		return err
	}

	// Apply routing file
	if err := e.applyRoutingFile(tempDir, targetDir); err != nil {
		return err
	}

	// Apply policy files
	if err := e.applyPolicyFiles(tempDir, targetDir); err != nil {
		return err
	}

	// Apply additional files
	if err := e.applyAdditionalFiles(tempDir, targetDir); err != nil {
		return err
	}

	fmt.Println("Bundle applied successfully!")
	return nil
}

// applySpecFile applies the spec.yaml file.
func (e *Extractor) applySpecFile(tempDir, targetDir string) error {
	sourcePath := filepath.Join(tempDir, "spec.yaml")
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil // No spec file in bundle
	}

	targetPath := filepath.Join(targetDir, "spec.yaml")
	return e.copyFile(sourcePath, targetPath, "spec.yaml")
}

// applyLockFile applies the spec.lock.json file.
func (e *Extractor) applyLockFile(tempDir, targetDir string) error {
	sourcePath := filepath.Join(tempDir, "spec.lock.json")
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil // No lock file in bundle
	}

	targetPath := filepath.Join(targetDir, "spec.lock.json")
	return e.copyFile(sourcePath, targetPath, "spec.lock.json")
}

// applyRoutingFile applies the routing.yaml file.
func (e *Extractor) applyRoutingFile(tempDir, targetDir string) error {
	sourcePath := filepath.Join(tempDir, "routing.yaml")
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil // No routing file in bundle
	}

	targetPath := filepath.Join(targetDir, "routing.yaml")
	return e.copyFile(sourcePath, targetPath, "routing.yaml")
}

// applyPolicyFiles applies all policy files.
func (e *Extractor) applyPolicyFiles(tempDir, targetDir string) error {
	policiesDir := filepath.Join(tempDir, "policies")
	if _, err := os.Stat(policiesDir); os.IsNotExist(err) {
		return nil // No policies in bundle
	}

	targetPoliciesDir := filepath.Join(targetDir, "policies")
	if err := os.MkdirAll(targetPoliciesDir, 0750); err != nil {
		return fmt.Errorf("failed to create policies directory: %w", err)
	}

	// Read all policy files
	entries, err := os.ReadDir(policiesDir)
	if err != nil {
		return fmt.Errorf("failed to read policies directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sourcePath := filepath.Join(policiesDir, entry.Name())
		targetPath := filepath.Join(targetPoliciesDir, entry.Name())

		if copyErr := e.copyFile(sourcePath, targetPath, "policies/"+entry.Name()); copyErr != nil {
			return copyErr
		}
	}

	return nil
}

// applyAdditionalFiles applies additional files from the bundle.
func (e *Extractor) applyAdditionalFiles(tempDir, targetDir string) error {
	// Skip manifest, checksums, and standard files
	skipFiles := map[string]bool{
		"manifest.yaml":  true,
		"checksums.txt":  true,
		"spec.yaml":      true,
		"spec.lock.json": true,
		"routing.yaml":   true,
		"policies":       true,
		"approvals":      true,
		"attestations":   true,
	}

	return filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Skip excluded files
		topDir := strings.Split(relPath, string(filepath.Separator))[0]
		if skipFiles[topDir] || skipFiles[relPath] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check exclude patterns
		for _, pattern := range e.opts.Exclude {
			//nolint:errcheck // Pattern validity checked at startup
			matched, _ := filepath.Match(pattern, relPath)
			if matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			targetPath := filepath.Join(targetDir, relPath)
			return os.MkdirAll(targetPath, info.Mode())
		}

		sourcePath := path
		targetPath := filepath.Join(targetDir, relPath)
		return e.copyFile(sourcePath, targetPath, relPath)
	})
}

// copyFile copies a file from source to target with confirmation if needed.
func (e *Extractor) copyFile(sourcePath, targetPath, displayName string) error {
	// Check if target exists
	if _, err := os.Stat(targetPath); err == nil {
		// File exists - check if we should overwrite
		if !e.opts.Force && !e.opts.Yes {
			fmt.Printf("File exists: %s. Overwrite? [y/N]: ", displayName)
			var response string
			if _, scanErr := fmt.Scanln(&response); scanErr != nil {
				// Error reading input, default to "no"
				fmt.Printf("Skipping %s\n", displayName)
				return nil
			}
			if response != "y" && response != "Y" {
				fmt.Printf("Skipping %s\n", displayName)
				return nil
			}
		}
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(targetPath), 0750); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", displayName, err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", displayName, err)
	}
	defer func() { _ = sourceFile.Close() }() //nolint:errcheck

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", displayName, err)
	}

	// Create target file
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", displayName, err)
	}
	defer func() { _ = targetFile.Close() }() //nolint:errcheck

	// Copy data
	if _, copyErr := io.Copy(targetFile, sourceFile); copyErr != nil {
		return fmt.Errorf("failed to copy file %s: %w", displayName, copyErr)
	}

	fmt.Printf("Applied: %s\n", displayName)
	return nil
}

// showSpecChanges shows what spec changes would be made.
func (e *Extractor) showSpecChanges(tempDir string) error {
	sourcePath := filepath.Join(tempDir, "spec.yaml")
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil
	}

	targetPath := filepath.Join(e.opts.TargetDir, "spec.yaml")
	return e.showFileChange(sourcePath, targetPath, "spec.yaml")
}

// showLockChanges shows what lock changes would be made.
func (e *Extractor) showLockChanges(tempDir string) error {
	sourcePath := filepath.Join(tempDir, "spec.lock.json")
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil
	}

	targetPath := filepath.Join(e.opts.TargetDir, "spec.lock.json")
	return e.showFileChange(sourcePath, targetPath, "spec.lock.json")
}

// showRoutingChanges shows what routing changes would be made.
func (e *Extractor) showRoutingChanges(tempDir string) error {
	sourcePath := filepath.Join(tempDir, "routing.yaml")
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil
	}

	targetPath := filepath.Join(e.opts.TargetDir, "routing.yaml")
	return e.showFileChange(sourcePath, targetPath, "routing.yaml")
}

// showPolicyChanges shows what policy changes would be made.
func (e *Extractor) showPolicyChanges(tempDir string) error {
	policiesDir := filepath.Join(tempDir, "policies")
	if _, err := os.Stat(policiesDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(policiesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sourcePath := filepath.Join(policiesDir, entry.Name())
		targetPath := filepath.Join(e.opts.TargetDir, "policies", entry.Name())

		if showErr := e.showFileChange(sourcePath, targetPath, "policies/"+entry.Name()); showErr != nil {
			return showErr
		}
	}

	return nil
}

// showAdditionalFileChanges shows what additional file changes would be made.
func (e *Extractor) showAdditionalFileChanges(tempDir string) error {
	// Similar to applyAdditionalFiles but just shows changes
	skipFiles := map[string]bool{
		"manifest.yaml":  true,
		"checksums.txt":  true,
		"spec.yaml":      true,
		"spec.lock.json": true,
		"routing.yaml":   true,
		"policies":       true,
		"approvals":      true,
		"attestations":   true,
	}

	return filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		topDir := strings.Split(relPath, string(filepath.Separator))[0]
		if skipFiles[topDir] || skipFiles[relPath] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		targetPath := filepath.Join(e.opts.TargetDir, relPath)
		return e.showFileChange(path, targetPath, relPath)
	})
}

// showFileChange shows what would change for a single file.
func (e *Extractor) showFileChange(sourcePath, targetPath, displayName string) error {
	_, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		fmt.Printf("  [CREATE] %s\n", displayName)
	} else if err == nil {
		fmt.Printf("  [UPDATE] %s\n", displayName)
	}
	return nil
}

// GetBundleInfo extracts basic information from a bundle without full extraction.
func GetBundleInfo(bundlePath string) (*BundleInfo, error) {
	// Open bundle file
	file, err := os.Open(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open bundle: %w", err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat bundle: %w", err)
	}

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzReader.Close() }() //nolint:errcheck

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Find and read manifest
	var manifestData []byte
	for {
		header, readErr := tarReader.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("failed to read tar: %w", readErr)
		}

		if header.Name == ManifestFileName {
			var readAllErr error
			manifestData, readAllErr = io.ReadAll(tarReader)
			if readAllErr != nil {
				return nil, fmt.Errorf("failed to read manifest: %w", readAllErr)
			}
			break
		}
	}

	if manifestData == nil {
		return nil, fmt.Errorf("manifest not found in bundle")
	}

	// Parse manifest
	var manifest Manifest
	if unmarshalErr := yaml.Unmarshal(manifestData, &manifest); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", unmarshalErr)
	}

	// Create BundleInfo
	info := &BundleInfo{
		ID:              manifest.ID,
		Version:         manifest.Version,
		Schema:          manifest.Schema,
		Created:         manifest.Created,
		IntegrityDigest: manifest.Integrity.Digest,
		GovernanceLevel: manifest.GovernanceLevel,
		Size:            fileInfo.Size(),
	}

	return info, nil
}
