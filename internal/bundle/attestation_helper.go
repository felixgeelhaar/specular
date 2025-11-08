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

// AddAttestationToBundle adds an attestation to an existing bundle.
// This re-packs the bundle with the attestation included.
func AddAttestationToBundle(bundlePath string, attestation *Attestation) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "bundle-attest-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract existing bundle
	if err := extractBundleToDir(bundlePath, tempDir); err != nil {
		return fmt.Errorf("failed to extract bundle: %w", err)
	}

	// Create attestations directory
	attestDir := filepath.Join(tempDir, "attestations")
	if err := os.MkdirAll(attestDir, 0755); err != nil {
		return fmt.Errorf("failed to create attestations directory: %w", err)
	}

	// Marshal attestation to YAML
	attestYAML, err := yaml.Marshal(attestation)
	if err != nil {
		return fmt.Errorf("failed to marshal attestation: %w", err)
	}

	// Write attestation file
	attestPath := filepath.Join(attestDir, "attestation.yaml")
	if err := os.WriteFile(attestPath, attestYAML, 0644); err != nil {
		return fmt.Errorf("failed to write attestation: %w", err)
	}

	// Re-create bundle with attestation
	if err := repackBundle(tempDir, bundlePath); err != nil {
		return fmt.Errorf("failed to repack bundle: %w", err)
	}

	return nil
}

// extractBundleToDir extracts a bundle to a directory.
func extractBundleToDir(bundlePath, targetDir string) error {
	file, err := os.Open(bundlePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, header.Name)

		// Ensure target path is within temp directory
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(targetDir)) {
			return fmt.Errorf("invalid file path in bundle: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
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
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk directory and add all files
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file content if regular file
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
