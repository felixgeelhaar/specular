package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func containsSlice(list []string, needle string) bool {
	for _, item := range list {
		if item == needle {
			return true
		}
	}
	return false
}

func TestSlugifyAndTrimHelpers(t *testing.T) {
	if slugify("Feature One") != "feature-one" {
		t.Fatalf("expected slugify to convert to feature-one")
	}

	featureID, err := types.NewFeatureID("feature")
	if err != nil {
		t.Fatalf("unexpected error creating feature ID: %v", err)
	}
	if slugs := featureSlugs(Feature{ID: featureID, Title: "Feature Title"}); len(slugs) != 2 {
		t.Fatalf("expected two slugs, got %d", len(slugs))
	}

	if trimmed := trimTestAffixes("Test_Feature_test"); trimmed != "feature" {
		t.Fatalf("expected trimTestAffixes to remove affixes, got %s", trimmed)
	}

	if !hasExtension("file.md", docExtensions) {
		t.Fatal("expected hasExtension to detect .md")
	}
	if hasExtension("file.exe", docExtensions) {
		t.Fatal("did not expect .exe to be accepted")
	}
}

func TestTraceResolverEnhancesSpec(t *testing.T) {
	tmp := t.TempDir()
	docsDir := filepath.Join(tmp, "docs")
	testsDir := filepath.Join(tmp, "tests")
	apiDir := filepath.Join(tmp, ".specular", "openapi")

	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("failed to create docs dir: %v", err)
	}
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(docsDir, "feature-one.md"), []byte("doc"), 0o600); err != nil {
		t.Fatalf("failed to write doc file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "feature-one_test.go"), []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "feature-one.yaml"), []byte("api"), 0o600); err != nil {
		t.Fatalf("failed to write api file: %v", err)
	}

	featureID, err := types.NewFeatureID("feature-one")
	if err != nil {
		t.Fatalf("failed to create feature ID: %v", err)
	}
	spec := &ProductSpec{
		Features: []Feature{
			{
				ID:    featureID,
				Title: "Feature One",
				Trace: []string{"placeholder"},
				API: []API{
					{Path: "/feature-one"},
				},
			},
		},
	}

	resolver := NewTraceResolver(tmp)
	if resolver == nil {
		t.Fatal("TraceResolver should not be nil for valid workspace")
	}

	resolver.EnhanceSpec(spec)
	trace := spec.Features[0].Trace
	if !containsSlice(trace, "docs/feature-one.md") {
		t.Fatalf("expected docs path added, got %v", trace)
	}
	if !containsSlice(trace, "tests/feature-one_test.go") {
		t.Fatalf("expected test path added, got %v", trace)
	}
	if !containsSlice(trace, ".specular/openapi/feature-one.yaml") {
		t.Fatalf("expected api path added, got %v", trace)
	}

	// Ensure EnhanceTraceArtifacts gracefully handles nil spec / invalid workspace
	EnhanceTraceArtifacts(nil, tmp)
	EnhanceTraceArtifacts(spec, filepath.Join(tmp, "non-existent"))
}
