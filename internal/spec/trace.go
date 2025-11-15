package spec

import (
	"fmt"
	"strings"
)

// NormalizeTrace ensures feature.trace points to implementation artifacts and
// requirement references are stored in feature.refs.
func NormalizeTrace(feature *Feature) {
	if feature == nil {
		return
	}

	var artifactTraces []string
	var refsToAdd []string

	for _, entry := range feature.Trace {
		value := strings.TrimSpace(entry)
		if value == "" {
			continue
		}
		if isLikelyTracePath(value) {
			artifactTraces = appendUnique(artifactTraces, value)
		} else {
			refsToAdd = append(refsToAdd, value)
		}
	}

	if len(refsToAdd) > 0 {
		feature.Refs = appendUnique(feature.Refs, refsToAdd...)
	}

	defaults := DefaultTraceArtifacts(*feature)
	feature.Trace = appendUnique(artifactTraces, defaults...)
}

// DefaultTraceArtifacts returns the canonical artifact paths for a feature.
func DefaultTraceArtifacts(feature Feature) []string {
	id := strings.TrimSpace(feature.ID.String())
	if id == "" {
		id = "feature"
	}

	traces := []string{
		fmt.Sprintf("docs/features/%s.md", id),
		fmt.Sprintf(".specular/tests/%s_test.go", id),
	}

	if len(feature.API) > 0 {
		traces = append(traces, fmt.Sprintf(".specular/openapi/%s.yaml", id))
	}

	return appendUnique(nil, traces...)
}

func isLikelyTracePath(value string) bool {
	if value == "" {
		return false
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}

	if strings.HasPrefix(trimmed, "./") || strings.HasPrefix(trimmed, "../") {
		return true
	}

	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return true
	}

	if strings.HasPrefix(trimmed, ".") {
		return true
	}

	// Strings with file extensions (e.g., main.go) but no spaces
	if strings.Contains(trimmed, ".") && !strings.Contains(trimmed, " ") {
		return true
	}

	return false
}

func appendUnique(list []string, items ...string) []string {
	if len(items) == 0 {
		return list
	}

	seen := make(map[string]struct{}, len(list))
	for _, existing := range list {
		seen[existing] = struct{}{}
	}

	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		list = append(list, item)
		seen[item] = struct{}{}
	}

	return list
}

func removeEntries(list []string, targets ...string) []string {
	if len(list) == 0 || len(targets) == 0 {
		return list
	}

	remove := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if target == "" {
			continue
		}
		remove[target] = struct{}{}
	}

	if len(remove) == 0 {
		return list
	}

	filtered := make([]string, 0, len(list))
	for _, item := range list {
		if _, ok := remove[item]; ok {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered
}

// Normalize ensures all derived data within the product spec is consistent.
func (p *ProductSpec) Normalize() {
	if p == nil {
		return
	}

	for i := range p.Features {
		NormalizeTrace(&p.Features[i])
	}
}
