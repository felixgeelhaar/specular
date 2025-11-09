package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SpecRepository defines the interface for loading and saving ProductSpec files.
// This interface enables dependency injection and makes testing easier.
type SpecRepository interface {
	// Load reads a ProductSpec from a file
	Load(path string) (*ProductSpec, error)

	// Save writes a ProductSpec to a file
	Save(spec *ProductSpec, path string) error
}

// FileSpecRepository implements SpecRepository for file-based storage
type FileSpecRepository struct{}

// NewFileSpecRepository creates a new file-based spec repository
func NewFileSpecRepository() *FileSpecRepository {
	return &FileSpecRepository{}
}

// Load reads a ProductSpec from a YAML file
func (r *FileSpecRepository) Load(path string) (*ProductSpec, error) {
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

// Save writes a ProductSpec to a YAML file
func (r *FileSpecRepository) Save(spec *ProductSpec, path string) error {
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

// Default instance for package-level functions
var defaultRepository = NewFileSpecRepository()

// LoadSpec reads a ProductSpec from a YAML file using the default repository.
// This is a convenience wrapper that maintains backwards compatibility.
func LoadSpec(path string) (*ProductSpec, error) {
	return defaultRepository.Load(path)
}

// SaveSpec writes a ProductSpec to a YAML file using the default repository.
// This is a convenience wrapper that maintains backwards compatibility.
func SaveSpec(spec *ProductSpec, path string) error {
	return defaultRepository.Save(spec, path)
}

// Compile-time verification that FileSpecRepository implements SpecRepository
var _ SpecRepository = (*FileSpecRepository)(nil)
