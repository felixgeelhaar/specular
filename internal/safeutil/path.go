package safeutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// JoinInsideBase joins path segments and ensures the resulting path stays within baseDir.
// This prevents directory traversal vulnerabilities when paths are derived from user input.
func JoinInsideBase(baseDir string, segments ...string) (string, error) {
	if baseDir == "" {
		return "", fmt.Errorf("safeutil: base directory is empty")
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("safeutil: resolve base directory: %w", err)
	}

	parts := append([]string{baseAbs}, segments...)
	target := filepath.Join(parts...)
	clean := filepath.Clean(target)

	basePrefix := baseAbs
	if runtime.GOOS == "windows" {
		basePrefix = strings.TrimSuffix(basePrefix, string(os.PathSeparator))
	}

	if !strings.HasPrefix(clean, basePrefix+string(os.PathSeparator)) && clean != basePrefix {
		return "", fmt.Errorf("safeutil: path %q escapes base %q", clean, baseAbs)
	}

	return clean, nil
}
