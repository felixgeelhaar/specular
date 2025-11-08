package bundle

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AddAttestationToBundle adds an attestation to an existing bundle.
// This re-packs the bundle with the attestation included.
func AddAttestationToBundle(bundlePath string, attestation *Attestation) error {
	// Extract bundle using security-hardened extraction with decompression bomb protection
	tempDir, extractErr := extractBundle(bundlePath)
	if extractErr != nil {
		return fmt.Errorf("failed to extract bundle: %w", extractErr)
	}
	defer cleanupOnError(tempDir)

	// Create attestations directory with secure permissions
	attestDir := filepath.Join(tempDir, "attestations")
	if mkdirErr := os.MkdirAll(attestDir, 0750); mkdirErr != nil {
		return fmt.Errorf("failed to create attestations directory: %w", mkdirErr)
	}

	// Marshal attestation to YAML
	attestYAML, marshalErr := yaml.Marshal(attestation)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal attestation: %w", marshalErr)
	}

	// Write attestation file with secure permissions (0600)
	attestPath := filepath.Join(attestDir, "attestation.yaml")
	if writeErr := os.WriteFile(attestPath, attestYAML, 0600); writeErr != nil {
		return fmt.Errorf("failed to write attestation: %w", writeErr)
	}

	// Re-create bundle with attestation
	if repackErr := repackBundle(tempDir, bundlePath); repackErr != nil {
		return fmt.Errorf("failed to repack bundle: %w", repackErr)
	}

	return nil
}

// repackBundle re-creates a bundle tarball from a directory.
func repackBundle(sourceDir, bundlePath string) error {
	// Create output file
	outFile, err := os.Create(bundlePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			// Log but don't override primary error
			fmt.Fprintf(os.Stderr, "warning: failed to close output file: %v\n", closeErr)
		}
	}()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer func() {
		if closeErr := gzWriter.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close gzip writer: %v\n", closeErr)
		}
	}()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer func() {
		if closeErr := tarWriter.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close tar writer: %v\n", closeErr)
		}
	}()

	// Walk directory and add all files
	walkErr := filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Get relative path
		relPath, relErr := filepath.Rel(sourceDir, path)
		if relErr != nil {
			return relErr
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, headerErr := tar.FileInfoHeader(info, "")
		if headerErr != nil {
			return headerErr
		}
		header.Name = relPath

		// Write header
		if writeHeaderErr := tarWriter.WriteHeader(header); writeHeaderErr != nil {
			return writeHeaderErr
		}

		// Write file content if regular file
		if info.Mode().IsRegular() {
			fileHandle, openErr := os.Open(path)
			if openErr != nil {
				return openErr
			}
			defer func() {
				if closeErr := fileHandle.Close(); closeErr != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to close file %s: %v\n", path, closeErr)
				}
			}()

			if _, copyErr := io.Copy(tarWriter, fileHandle); copyErr != nil {
				return copyErr
			}
		}

		return nil
	})

	return walkErr
}
