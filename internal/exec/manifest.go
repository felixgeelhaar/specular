package exec

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CreateManifest creates a run manifest for audit purposes
func CreateManifest(step Step, result *Result) *RunManifest {
	return &RunManifest{
		Timestamp:    time.Now(),
		StepID:       step.ID,
		Runner:       step.Runner,
		Image:        step.Image,
		Command:      step.Cmd,
		Env:          step.Env,
		ExitCode:     result.ExitCode,
		Duration:     result.Duration.String(),
		InputHashes:  make(map[string]string),
		OutputHashes: make(map[string]string),
	}
}

// SaveManifest writes a run manifest to disk
func SaveManifest(manifest *RunManifest, dir string) error {
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("%s_%s.json",
		manifest.Timestamp.Format("20060102_150405"),
		manifest.StepID)
	path := filepath.Join(dir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// HashFile computes the SHA-256 hash of a file
func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// AddInputHash adds an input file hash to the manifest
func (m *RunManifest) AddInputHash(name, path string) error {
	hash, err := HashFile(path)
	if err != nil {
		return err
	}
	m.InputHashes[name] = hash
	return nil
}

// AddOutputHash adds an output file hash to the manifest
func (m *RunManifest) AddOutputHash(name, path string) error {
	hash, err := HashFile(path)
	if err != nil {
		return err
	}
	m.OutputHashes[name] = hash
	return nil
}
