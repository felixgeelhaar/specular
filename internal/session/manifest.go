package session

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ManifestEntry is one session declared in a fleet manifest.
type ManifestEntry struct {
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	Harness    string `json:"harness,omitempty" yaml:"harness,omitempty"`
	Goal       string `json:"goal" yaml:"goal"`
	Profile    string `json:"profile,omitempty" yaml:"profile,omitempty"`
	NoWorktree bool   `json:"noWorktree,omitempty" yaml:"noWorktree,omitempty"`
}

// ParseManifest decodes a JSON or YAML fleet manifest.
// Accepted shapes:
//   - array of entries
//   - object with a "sessions" array
func ParseManifest(data []byte) ([]ManifestEntry, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("session: empty manifest")
	}

	var entries []ManifestEntry
	if err := tryDecodeManifest(data, &entries); err == nil {
		return validateManifest(entries)
	}

	var wrapped struct {
		Sessions []ManifestEntry `json:"sessions" yaml:"sessions"`
	}
	if err := tryDecodeManifest(data, &wrapped); err != nil {
		return nil, fmt.Errorf("session: parse manifest: %w", err)
	}
	return validateManifest(wrapped.Sessions)
}

func tryDecodeManifest(data []byte, dest any) error {
	jsonErr := json.Unmarshal(data, dest)
	if jsonErr == nil {
		return nil
	}
	yamlErr := yaml.Unmarshal(data, dest)
	if yamlErr == nil {
		return nil
	}
	return fmt.Errorf("json: %v; yaml: %v", jsonErr, yamlErr)
}

func validateManifest(entries []ManifestEntry) ([]ManifestEntry, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("session: manifest has no sessions")
	}
	out := make([]ManifestEntry, 0, len(entries))
	seen := map[string]bool{}
	for i, e := range entries {
		goal := strings.TrimSpace(e.Goal)
		if goal == "" {
			return nil, fmt.Errorf("session: manifest entry %d missing goal", i)
		}
		name := strings.TrimSpace(e.Name)
		if name != "" {
			if seen[name] {
				return nil, fmt.Errorf("session: duplicate manifest name %q", name)
			}
			seen[name] = true
		}
		e.Goal = goal
		e.Name = name
		e.Harness = strings.TrimSpace(e.Harness)
		e.Profile = strings.TrimSpace(e.Profile)
		out = append(out, e)
	}
	return out, nil
}
