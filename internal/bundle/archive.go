package bundle

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Security limits for bundle extraction to prevent decompression bombs (G110)
const (
	// MaxBundleSize is the maximum total extracted size (1 GB)
	MaxBundleSize = 1 * 1024 * 1024 * 1024
	// MaxFileSize is the maximum individual file size (100 MB)
	MaxFileSize = 100 * 1024 * 1024
	// MaxFileCount is the maximum number of files in a bundle
	MaxFileCount = 10000
)

// cleanupOnError attempts to remove the directory and logs any cleanup failures.
// This is used in error paths where cleanup failure is secondary to the primary error.
func cleanupOnError(dir string) {
	if rmErr := os.RemoveAll(dir); rmErr != nil {
		log.Printf("warning: failed to cleanup directory %s: %v", dir, rmErr)
	}
}

// extractBundle extracts a .sbundle.tgz file to a temporary directory with comprehensive security checks.
// This function protects against:
// - G305: Path traversal attacks
// - G110: Decompression bomb attacks
// - G115: Integer overflow in file mode conversion
// - G306/G301: Unsafe file/directory permissions
func extractBundle(bundlePath string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "bundle-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Open and create readers
	tarReader, cleanup, err := openBundleReaders(bundlePath, tempDir)
	if err != nil {
		return "", err
	}
	defer cleanup()

	// Extract all files with security checks
	if extractErr := extractAllFiles(tarReader, tempDir); extractErr != nil {
		cleanupOnError(tempDir)
		return "", extractErr
	}

	return tempDir, nil
}

// openBundleReaders opens the bundle file and creates gzip and tar readers
func openBundleReaders(bundlePath, tempDir string) (*tar.Reader, func(), error) {
	file, err := os.Open(bundlePath)
	if err != nil {
		cleanupOnError(tempDir)
		return nil, nil, fmt.Errorf("failed to open bundle: %w", err)
	}

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("warning: failed to close bundle file: %v", closeErr)
		}
		cleanupOnError(tempDir)
		return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	tarReader := tar.NewReader(gzReader)

	cleanup := func() {
		if closeErr := gzReader.Close(); closeErr != nil {
			log.Printf("warning: failed to close gzip reader: %v", closeErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("warning: failed to close bundle file: %v", closeErr)
		}
	}

	return tarReader, cleanup, nil
}

// extractAllFiles extracts all files from the tar archive with security checks
func extractAllFiles(tarReader *tar.Reader, tempDir string) error {
	var totalSize int64
	var fileCount int

	for {
		header, readErr := tarReader.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read tar: %w", readErr)
		}

		// Validate security constraints
		if err := validateExtractionSecurity(header, &fileCount, &totalSize); err != nil {
			return err
		}

		// Extract the file
		if err := extractSingleFile(tarReader, header, tempDir); err != nil {
			return err
		}
	}

	return nil
}

// validateExtractionSecurity performs all security validations for extraction
func validateExtractionSecurity(header *tar.Header, fileCount *int, totalSize *int64) error {
	// Check file count limit (G110: decompression bomb protection)
	*fileCount++
	if *fileCount > MaxFileCount {
		return fmt.Errorf("bundle exceeds maximum file count (%d)", MaxFileCount)
	}

	// Check individual file size (G110: decompression bomb protection)
	if header.Size > MaxFileSize {
		return fmt.Errorf("file %s exceeds maximum size (%d bytes)", header.Name, MaxFileSize)
	}

	// Check total extracted size (G110: decompression bomb protection)
	*totalSize += header.Size
	if *totalSize > MaxBundleSize {
		return fmt.Errorf("bundle exceeds maximum total size (%d bytes)", MaxBundleSize)
	}

	// Validate path to prevent traversal attacks (G305)
	if validateErr := validateBundlePath(header.Name); validateErr != nil {
		return fmt.Errorf("invalid path in bundle: %w", validateErr)
	}

	return nil
}

// extractSingleFile extracts a single file or directory from the tar archive
func extractSingleFile(tarReader *tar.Reader, header *tar.Header, tempDir string) error {
	// #nosec G305 - Path traversal is validated on the next line
	targetPath := filepath.Join(tempDir, header.Name)

	// Additional path traversal check: ensure target is within temp directory
	if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(tempDir)) {
		return fmt.Errorf("path traversal attempt detected: %s", header.Name)
	}

	switch header.Typeflag {
	case tar.TypeDir:
		return extractDirectory(header, targetPath)
	case tar.TypeReg:
		return extractRegularFile(tarReader, header, targetPath)
	}

	return nil
}

// extractDirectory creates a directory with secure permissions
func extractDirectory(header *tar.Header, targetPath string) error {
	mode := sanitizeFileMode(header.Mode, header.Typeflag)
	if mkdirErr := os.MkdirAll(targetPath, mode); mkdirErr != nil {
		return fmt.Errorf("failed to create directory: %w", mkdirErr)
	}
	return nil
}

// extractRegularFile creates a file and copies data with size validation
func extractRegularFile(tarReader *tar.Reader, header *tar.Header, targetPath string) error {
	// Create parent directory with secure permissions (G306)
	if mkdirErr := os.MkdirAll(filepath.Dir(targetPath), 0750); mkdirErr != nil {
		return fmt.Errorf("failed to create parent directory: %w", mkdirErr)
	}

	// Create file with secure permissions (G306/G301, G115)
	mode := sanitizeFileMode(header.Mode, header.Typeflag)
	outFile, openErr := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, mode)
	if openErr != nil {
		return fmt.Errorf("failed to create file: %w", openErr)
	}

	// Copy data with size validation
	written, copyErr := io.Copy(outFile, io.LimitReader(tarReader, header.Size))
	closeErr := outFile.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to write file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close file: %w", closeErr)
	}
	if written != header.Size {
		return fmt.Errorf("file size mismatch for %s: expected %d, got %d", header.Name, header.Size, written)
	}

	return nil
}

// validateBundlePath checks for path traversal attempts in bundle paths (G305).
func validateBundlePath(path string) error {
	// Check for absolute paths
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}

	// Check for parent directory references
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, string(filepath.Separator)+"..") {
		return fmt.Errorf("parent directory references not allowed: %s", path)
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("null bytes not allowed in path: %s", path)
	}

	return nil
}

// sanitizeFileMode validates and sanitizes file permissions from tar headers.
// This prevents G115 (integer overflow) and G306 (unsafe permissions).
func sanitizeFileMode(mode int64, typeflag byte) os.FileMode {
	// Cap mode to valid FileMode range (0-0777 octal)
	const maxMode = 0777
	if mode < 0 || mode > maxMode {
		mode = maxMode
	}

	// Convert with explicit range check to prevent G115 integer overflow
	// os.FileMode is uint32, so we ensure mode is in valid range before conversion
	var fileMode os.FileMode
	if mode >= 0 && mode <= 0777 {
		fileMode = os.FileMode(mode)
	} else {
		fileMode = 0777
	}

	// Apply secure defaults based on type
	if typeflag == tar.TypeDir {
		// Directories: cap at 0750 (owner: rwx, group: r-x, other: none)
		return fileMode & 0750
	}

	// Regular files: cap at 0600 (owner: rw, group: none, other: none)
	return fileMode & 0600
}
