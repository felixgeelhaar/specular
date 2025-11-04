package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadSpec reads a ProductSpec from a YAML file
func LoadSpec(path string) (*ProductSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec file: %w", err)
	}

	var spec ProductSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal spec: %w", err)
	}

	return &spec, nil
}

// SaveSpec writes a ProductSpec to a YAML file
func SaveSpec(spec *ProductSpec, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write spec file: %w", err)
	}

	return nil
}
