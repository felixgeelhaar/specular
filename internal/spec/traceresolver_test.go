package spec

import (
	"path/filepath"
	"testing"
)

func TestTraceResolverEnhancesFeatureTraces(t *testing.T) {
	workspace := filepath.Join("testdata", "traceresolver")
	resolver := NewTraceResolver(workspace)
	if resolver == nil {
		t.Fatalf("expected resolver for workspace %s", workspace)
	}

	feature := Feature{
		ID:    "feat-001",
		Title: "Todo REST Endpoints",
		API:   []API{{Method: "GET", Path: "/api/todo"}},
		Trace: []string{
			"docs/features/feat-001.md",
			".specular/tests/feat-001_test.go",
			".specular/openapi/feat-001.yaml",
		},
	}

	resolver.enhanceFeature(&feature)

	assertContains(t, feature.Trace, "docs/features/todo-rest-endpoints.md")
	assertContains(t, feature.Trace, "internal/todo/todo_rest_endpoints_test.go")
	assertContains(t, feature.Trace, "src/features/todo/todo-rest-endpoints.spec.ts")
	assertContains(t, feature.Trace, ".specular/openapi/todo-rest-endpoints.yaml")

	if sliceContainsString(feature.Trace, "docs/features/feat-001.md") {
		t.Fatalf("expected placeholder doc path to be removed, got %v", feature.Trace)
	}
	if sliceContainsString(feature.Trace, ".specular/tests/feat-001_test.go") {
		t.Fatalf("expected placeholder test path to be removed, got %v", feature.Trace)
	}
}

func TestTraceResolverHandlesDifferentSlugPatterns(t *testing.T) {
	workspace := filepath.Join("testdata", "traceresolver")
	resolver := NewTraceResolver(workspace)
	if resolver == nil {
		t.Fatalf("expected resolver for workspace %s", workspace)
	}

	feature := Feature{
		ID:    "feat-010",
		Title: "Reporting Dashboard",
		API:   []API{{Method: "GET", Path: "/api/reports"}},
		Trace: []string{
			"docs/features/feat-010.md",
			".specular/tests/feat-010_test.go",
			".specular/openapi/feat-010.yaml",
		},
	}

	resolver.enhanceFeature(&feature)

	assertContains(t, feature.Trace, "docs/design/reporting-dashboard.adoc")
	assertContains(t, feature.Trace, "tests/python/test_reporting_dashboard.py")
	assertContains(t, feature.Trace, "api/reporting-dashboard.yaml")
}

func TestTraceResolverCaching(t *testing.T) {
	workspace := filepath.Join("testdata", "traceresolver")

	resolverCacheMu.Lock()
	resolverCache = make(map[string]*TraceResolver)
	resolverCacheMu.Unlock()

	first := getTraceResolver(workspace)
	if first == nil {
		t.Fatalf("expected resolver for %s", workspace)
	}
	second := getTraceResolver(workspace)
	if second == nil {
		t.Fatalf("expected cached resolver for %s", workspace)
	}
	if first != second {
		t.Fatalf("expected cached resolver to be reused")
	}
}

func assertContains(t *testing.T, list []string, target string) {
	t.Helper()
	if !sliceContainsString(list, target) {
		t.Fatalf("expected list %v to contain %q", list, target)
	}
}

func sliceContainsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
