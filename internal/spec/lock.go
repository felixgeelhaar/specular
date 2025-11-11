package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/specular/internal/domain"
)

// GenerateSpecLock creates a SpecLock from a ProductSpec
func GenerateSpecLock(spec ProductSpec, version string) (*SpecLock, error) {
	lock := &SpecLock{
		Version:  version,
		Features: make(map[domain.FeatureID]LockedFeature),
	}

	for _, feature := range spec.Features {
		hash, err := Hash(feature)
		if err != nil {
			return nil, fmt.Errorf("hash feature %s: %w", feature.ID, err)
		}

		lock.Features[feature.ID] = LockedFeature{
			Hash:        hash,
			OpenAPIPath: fmt.Sprintf(".specular/openapi/%s.yaml", feature.ID),
			TestPaths: []string{
				fmt.Sprintf(".specular/tests/%s_test.go", feature.ID),
			},
		}
	}

	return lock, nil
}

// SaveSpecLock writes a SpecLock to disk
func SaveSpecLock(lock *SpecLock, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal spec lock: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write spec lock: %w", err)
	}

	return nil
}

// LoadSpecLock reads a SpecLock from disk
func LoadSpecLock(path string) (*SpecLock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec lock: %w", err)
	}

	var lock SpecLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("unmarshal spec lock: %w", err)
	}

	return &lock, nil
}
