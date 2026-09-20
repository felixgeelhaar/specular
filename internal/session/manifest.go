package session

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ManifestEntry is one session declared in a fleet manifest.
type ManifestEntry struct {
	Name       string   `json:"name,omitempty" yaml:"name,omitempty"`
	Harness    string   `json:"harness,omitempty" yaml:"harness,omitempty"`
	Goal       string   `json:"goal" yaml:"goal"`
	Profile    string   `json:"profile,omitempty" yaml:"profile,omitempty"`
	NoWorktree bool     `json:"noWorktree,omitempty" yaml:"noWorktree,omitempty"`
	DependsOn  []string `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`
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
		deps, depErr := normalizeDependsOn(e.DependsOn, i)
		if depErr != nil {
			return nil, depErr
		}
		e.Goal = goal
		e.Name = name
		e.Harness = strings.TrimSpace(e.Harness)
		e.Profile = strings.TrimSpace(e.Profile)
		e.DependsOn = deps
		out = append(out, e)
	}
	if err := validateDependencyGraph(out); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeDependsOn(raw []string, index int) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, d := range raw {
		name := strings.TrimSpace(d)
		if name == "" {
			return nil, fmt.Errorf("session: manifest entry %d has empty dependsOn name", index)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out, nil
}

func validateDependencyGraph(entries []ManifestEntry) error {
	byName := map[string]ManifestEntry{}
	for _, e := range entries {
		if e.Name != "" {
			byName[e.Name] = e
		}
	}
	for i, e := range entries {
		for _, dep := range e.DependsOn {
			if _, ok := byName[dep]; !ok {
				return fmt.Errorf("session: manifest entry %d (%s) depends on unknown %q", i, e.Name, dep)
			}
			if e.Name != "" && e.Name == dep {
				return fmt.Errorf("session: manifest entry %q depends on itself", e.Name)
			}
		}
	}
	return detectDependencyCycle(byName)
}

func detectDependencyCycle(byName map[string]ManifestEntry) error {
	const (
		unseen = 0
		active = 1
		done   = 2
	)
	state := map[string]int{}
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case active:
			return fmt.Errorf("session: dependency cycle involving %q", name)
		case done:
			return nil
		}
		state[name] = active
		for _, dep := range byName[name].DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[name] = done
		return nil
	}
	for name := range byName {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}
