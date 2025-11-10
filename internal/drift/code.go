package drift

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/specular/internal/spec"
)

// CodeDriftOptions configures code drift detection
type CodeDriftOptions struct {
	ProjectRoot string   // Root directory of the project
	TracePaths  []string // File paths to track for drift
	APISpecPath string   // Path to OpenAPI spec (if applicable)
	IgnoreGlobs []string // Patterns to ignore (e.g., "*.test.js")
}

// DetectCodeDrift checks for code drift against the specification
func DetectCodeDrift(s *spec.ProductSpec, lock *spec.SpecLock, opts CodeDriftOptions) []Finding {
	var findings []Finding

	// Check file hashes for tracked files (always check test files and trace files)
	hashFindings := checkFileHashes(s, lock, opts)
	findings = append(findings, hashFindings...)

	// Check API implementations against spec
	apiFindings := checkAPIImplementations(s, opts)
	findings = append(findings, apiFindings...)

	// Check test coverage for features
	testFindings := checkTestCoverage(s, opts)
	findings = append(findings, testFindings...)

	return findings
}

// checkFileHashes verifies that tracked files haven't changed unexpectedly
func checkFileHashes(s *spec.ProductSpec, lock *spec.SpecLock, opts CodeDriftOptions) []Finding {
	var findings []Finding

	for _, feature := range s.Features {
		lockedFeature, exists := lock.Features[feature.ID]
		if !exists {
			continue
		}

		// Check test files
		for _, testPath := range lockedFeature.TestPaths {
			if !pathExists(filepath.Join(opts.ProjectRoot, testPath)) {
				findings = append(findings, Finding{
					Code:      "MISSING_TEST",
					FeatureID: feature.ID,
					Message:   fmt.Sprintf("Test file missing: %s", testPath),
					Severity:  "error",
					Location:  testPath,
				})
				continue
			}

			// Compute current hash
			currentHash, err := hashFile(filepath.Join(opts.ProjectRoot, testPath))
			if err != nil {
				findings = append(findings, Finding{
					Code:      "HASH_ERROR",
					FeatureID: feature.ID,
					Message:   fmt.Sprintf("Cannot hash file %s: %v", testPath, err),
					Severity:  "warning",
					Location:  testPath,
				})
				continue
			}

			// For now, we just check existence
			// Future: Store test hashes in SpecLock and compare
			_ = currentHash
		}

		// Check trace files
		for _, tracePath := range feature.Trace {
			// Check if file is ignored first
			if shouldIgnore(tracePath, opts.IgnoreGlobs) {
				continue
			}

			fullPath := filepath.Join(opts.ProjectRoot, tracePath)
			if !pathExists(fullPath) {
				findings = append(findings, Finding{
					Code:      "MISSING_TRACE",
					FeatureID: feature.ID,
					Message:   fmt.Sprintf("Traced file missing: %s", tracePath),
					Severity:  "error",
					Location:  tracePath,
				})
				continue
			}

			// Verify file exists and is readable
			if _, err := os.Stat(fullPath); err != nil {
				findings = append(findings, Finding{
					Code:      "TRACE_ERROR",
					FeatureID: feature.ID,
					Message:   fmt.Sprintf("Cannot access traced file %s: %v", tracePath, err),
					Severity:  "warning",
					Location:  tracePath,
				})
			}
		}
	}

	return findings
}

// checkAPIImplementations verifies API endpoints are implemented
func checkAPIImplementations(s *spec.ProductSpec, opts CodeDriftOptions) []Finding {
	var findings []Finding

	// Check if any features have APIs defined
	hasAPIs := false
	for _, feature := range s.Features {
		if len(feature.API) > 0 {
			hasAPIs = true
			break
		}
	}

	if !hasAPIs {
		return findings // No APIs to check
	}

	// If no API spec path provided, we can't validate
	if opts.APISpecPath == "" {
		return findings
	}

	// Validate API spec and endpoints
	apiFindings := ValidateAPISpec(opts.APISpecPath, opts.ProjectRoot, s.Features)
	findings = append(findings, apiFindings...)

	return findings
}

// checkTestCoverage verifies that features have associated tests
func checkTestCoverage(s *spec.ProductSpec, opts CodeDriftOptions) []Finding {
	var findings []Finding

	for _, feature := range s.Features {
		// Check if feature has test paths defined
		testCount := 0
		testPaths := []string{}

		// Count test files in project
		if opts.ProjectRoot != "" {
			// Look for test files related to this feature
			// Heuristic: feature ID in filename or feature title in test file
			featureName := strings.ToLower(strings.ReplaceAll(feature.Title, " ", "_"))

			// Walk the directory tree to find matching test files
			//nolint:errcheck,gosec // Walk errors handled inline
			filepath.Walk(opts.ProjectRoot, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip files with errors
				}
				if info.IsDir() {
					return nil // Skip directories themselves
				}

				basename := strings.ToLower(filepath.Base(path))
				// Only check files that look like test files
				if strings.Contains(basename, "test") {
					if strings.Contains(basename, featureName) || strings.Contains(basename, feature.ID.String()) {
						testCount++
						testPaths = append(testPaths, path)
					}
				}
				return nil
			})
		}

		// Check if feature has minimal test coverage
		if testCount == 0 && feature.Priority == "P0" {
			findings = append(findings, Finding{
				Code:      "NO_TESTS",
				FeatureID: feature.ID,
				Message:   fmt.Sprintf("P0 feature '%s' has no associated tests", feature.Title),
				Severity:  "error",
				Location:  feature.ID.String(),
			})
		} else if testCount == 0 && feature.Priority == "P1" {
			findings = append(findings, Finding{
				Code:      "NO_TESTS",
				FeatureID: feature.ID,
				Message:   fmt.Sprintf("P1 feature '%s' has no associated tests", feature.Title),
				Severity:  "warning",
				Location:  feature.ID.String(),
			})
		}
	}

	return findings
}

// pathExists checks if a file or directory exists
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// hashFile computes SHA-256 hash of a file
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck // Deferred close is best effort

	h := sha256.New()
	if _, copyErr := io.Copy(h, f); copyErr != nil {
		return "", copyErr
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// shouldIgnore checks if a path should be ignored
func shouldIgnore(path string, ignoreGlobs []string) bool {
	for _, pattern := range ignoreGlobs {
		matched, _ := filepath.Match(pattern, filepath.Base(path)) //nolint:errcheck // Match error only on malformed pattern, not possible here
		if matched {
			return true
		}
	}
	return false
}
