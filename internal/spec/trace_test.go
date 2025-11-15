package spec

import (
	"slices"
	"testing"
)

func TestNormalizeTraceAddsDefaultsAndSplitsRefs(t *testing.T) {
	feature := Feature{
		ID: "feat-010",
		Trace: []string{
			"docs/features/custom.md",
			"PRD section 4.1",
			".specular/tests/feat-010_test.go",
			"Tech Design 1.2",
		},
		Refs: []string{"existing-ref"},
		API: []API{
			{Method: "GET", Path: "/api/example"},
		},
	}

	NormalizeTrace(&feature)

	expectedTraceEntries := []string{
		"docs/features/custom.md",          // preserved
		"docs/features/feat-010.md",        // default doc artifact
		".specular/tests/feat-010_test.go", // default test artifact (deduped)
		".specular/openapi/feat-010.yaml",  // API artifact
	}

	for _, entry := range expectedTraceEntries {
		if !slices.Contains(feature.Trace, entry) {
			t.Fatalf("expected trace to contain %q, got %v", entry, feature.Trace)
		}
	}

	count := 0
	for _, trace := range feature.Trace {
		if trace == ".specular/tests/feat-010_test.go" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected single test trace entry, got %d occurrences", count)
	}

	expectedRefs := []string{
		"existing-ref",
		"PRD section 4.1",
		"Tech Design 1.2",
	}
	for _, ref := range expectedRefs {
		if !slices.Contains(feature.Refs, ref) {
			t.Fatalf("expected refs to contain %q, got %v", ref, feature.Refs)
		}
	}
}

func TestDefaultTraceArtifactsWithoutAPI(t *testing.T) {
	feature := Feature{ID: "feat-020"}

	artifacts := DefaultTraceArtifacts(feature)
	expected := []string{
		"docs/features/feat-020.md",
		".specular/tests/feat-020_test.go",
	}

	if len(artifacts) != len(expected) {
		t.Fatalf("expected %d artifacts, got %d (%v)", len(expected), len(artifacts), artifacts)
	}

	for _, entry := range expected {
		if !slices.Contains(artifacts, entry) {
			t.Fatalf("expected artifacts to contain %q, got %v", entry, artifacts)
		}
	}
}

func TestProductSpecNormalizePopulatesTrace(t *testing.T) {
	spec := ProductSpec{
		Features: []Feature{
			{ID: "feat-030"},
			{
				ID: "feat-031",
				API: []API{
					{Method: "GET", Path: "/api/resource"},
				},
			},
		},
	}

	spec.Normalize()

	first := spec.Features[0]
	if !slices.Contains(first.Trace, "docs/features/feat-030.md") ||
		!slices.Contains(first.Trace, ".specular/tests/feat-030_test.go") {
		t.Fatalf("expected default artifacts for feat-030, got %v", first.Trace)
	}

	second := spec.Features[1]
	if !slices.Contains(second.Trace, ".specular/openapi/feat-031.yaml") {
		t.Fatalf("expected openapi trace for feat-031, got %v", second.Trace)
	}
}
