// Package policylibrary ships open framework→evidence control mappings.
// Seeds are embedded so `specular policy library` works without a checkout.
package policylibrary

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed seed/*.yaml
var seedFS embed.FS

// Artifact maps a control requirement to a Specular evidence path.
type Artifact struct {
	Kind string `yaml:"kind"`
	Path string `yaml:"path"`
	Why  string `yaml:"why"`
}

// Entry is one library fragment (control mapping).
type Entry struct {
	ID               string                 `yaml:"id"`
	Framework        string                 `yaml:"framework"`
	Control          string                 `yaml:"control"`
	Title            string                 `yaml:"title"`
	Summary          string                 `yaml:"summary"`
	Version          string                 `yaml:"version"`
	Artifacts        []Artifact             `yaml:"artifacts"`
	EvidenceCommands []string               `yaml:"evidence_commands"`
	Compliance       map[string]interface{} `yaml:"compliance,omitempty"`
	// Raw is the original YAML bytes (for install).
	Raw []byte `yaml:"-"`
}

// List returns all embedded library entries, sorted by id.
func List() ([]Entry, error) {
	names, err := fs.Glob(seedFS, "seed/*.yaml")
	if err != nil {
		return nil, fmt.Errorf("policylibrary: list seed: %w", err)
	}
	sort.Strings(names)
	out := make([]Entry, 0, len(names))
	for _, name := range names {
		e, loadErr := loadSeed(name)
		if loadErr != nil {
			return nil, loadErr
		}
		out = append(out, e)
	}
	return out, nil
}

// Get returns one entry by id (e.g. "soc2-cc8.1").
func Get(id string) (Entry, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Entry{}, fmt.Errorf("policylibrary: empty id")
	}
	entries, err := List()
	if err != nil {
		return Entry{}, err
	}
	for _, e := range entries {
		if e.ID == id {
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("policylibrary: unknown id %q (run: specular policy library list)", id)
}

// DefaultInstallPath returns .specular/policies/<id>.yaml under root.
func DefaultInstallPath(root, id string) string {
	if root == "" {
		root = "."
	}
	return filepath.Join(root, ".specular", "policies", id+".yaml")
}

// Install writes the seed YAML to dest. Refuses overwrite unless force.
func Install(id, dest string, force bool) error {
	e, err := Get(id)
	if err != nil {
		return err
	}
	if valErr := Validate(e); valErr != nil {
		return valErr
	}
	if dest == "" {
		dest = DefaultInstallPath(".", id)
	}
	if !force {
		if _, statErr := os.Stat(dest); statErr == nil {
			return fmt.Errorf("policylibrary: %s already exists (use --force to overwrite)", dest)
		}
	}
	if mkErr := os.MkdirAll(filepath.Dir(dest), 0o750); mkErr != nil {
		return fmt.Errorf("policylibrary: mkdir: %w", mkErr)
	}
	if writeErr := os.WriteFile(dest, e.Raw, 0o600); writeErr != nil {
		return fmt.Errorf("policylibrary: write %s: %w", dest, writeErr)
	}
	return nil
}

// Validate checks required fields on a library entry.
func Validate(e Entry) error {
	if e.ID == "" {
		return fmt.Errorf("policylibrary: missing id")
	}
	if e.Framework == "" {
		return fmt.Errorf("policylibrary: %s: missing framework", e.ID)
	}
	if e.Control == "" {
		return fmt.Errorf("policylibrary: %s: missing control", e.ID)
	}
	if e.Title == "" {
		return fmt.Errorf("policylibrary: %s: missing title", e.ID)
	}
	if len(e.Artifacts) == 0 {
		return fmt.Errorf("policylibrary: %s: need at least one artifact", e.ID)
	}
	for i, a := range e.Artifacts {
		if a.Kind == "" || a.Path == "" {
			return fmt.Errorf("policylibrary: %s: artifact[%d] needs kind and path", e.ID, i)
		}
	}
	return nil
}

func loadSeed(name string) (Entry, error) {
	raw, err := seedFS.ReadFile(name)
	if err != nil {
		return Entry{}, fmt.Errorf("policylibrary: read %s: %w", name, err)
	}
	var e Entry
	if parseErr := yaml.Unmarshal(raw, &e); parseErr != nil {
		return Entry{}, fmt.Errorf("policylibrary: parse %s: %w", name, parseErr)
	}
	e.Raw = raw
	if e.ID == "" {
		base := filepath.Base(name)
		e.ID = strings.TrimSuffix(base, filepath.Ext(base))
	}
	if valErr := Validate(e); valErr != nil {
		return Entry{}, valErr
	}
	return e, nil
}
