package gate

import (
	"strings"
	"testing"
)

func TestFormatMarkdownAllow(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict: Allow,
		Reason:  "drift skipped; policy skipped; provenance unattested",
		Change:  ChangeSection{Dirty: true, Files: 2, Branch: "feat/x"},
		Provenance: ProvenanceSection{
			Status: StatusSkipped,
			Note:   "unattested",
		},
		Drift:  DriftSection{Status: StatusSkipped, Note: "brownfield"},
		Policy: PolicySection{Status: StatusSkipped},
	}
	md := FormatMarkdownWith(res, FormatMarkdownOptions{EvidenceID: "ev_abc"})
	for _, want := range []string{
		MarkdownMarker,
		"PASSED",
		"`ALLOW`",
		"| Provenance |",
		"**Evidence:** `ev_abc`",
		"specular explain ev_abc",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q:\n%s", want, md)
		}
	}
}

func TestFormatMarkdownDenyFindings(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict: Deny,
		Reason:  "hash mismatch",
		Drift: DriftSection{
			Status: StatusFail,
			Findings: []FindingDetail{{
				Category: "code",
				Code:     "HASH_MISMATCH",
				Severity: "error",
				Message:  "file hash mismatch",
				Location: "src/auth.go:12",
				Path:     "src/auth.go",
				Line:     12,
			}},
		},
		Policy: PolicySection{Status: StatusSkipped},
	}
	md := FormatMarkdown(res)
	for _, want := range []string{
		"DENIED",
		"HASH_MISMATCH",
		"src/auth.go:12",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q:\n%s", want, md)
		}
	}
}

func TestParseFindingLocation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		wantPath string
		wantLine int
		wantOK   bool
	}{
		{"src/a.go", "src/a.go", 0, true},
		{"src/a.go:12", "src/a.go", 12, true},
		{"src/a.go:12:3", "src/a.go", 12, true},
		{"task:foo", "", 0, false},
		{"", "", 0, false},
		{"https://example.com/x", "", 0, false},
	}
	for _, tc := range cases {
		path, line, ok := ParseFindingLocation(tc.in)
		if ok != tc.wantOK || path != tc.wantPath || line != tc.wantLine {
			t.Fatalf("%q => (%q,%d,%v) want (%q,%d,%v)",
				tc.in, path, line, ok, tc.wantPath, tc.wantLine, tc.wantOK)
		}
	}
}

func TestFormatGitHubAnnotations(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict: Deny,
		Drift: DriftSection{
			Findings: []FindingDetail{{
				Code:     "HASH_MISMATCH",
				Severity: "error",
				Message:  "mismatch",
				Path:     "src/a.go",
				Line:     4,
			}},
		},
	}
	out := FormatGitHubAnnotations(AnnotationsFromResult(res))
	if !strings.Contains(out, "::error ") {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(out, "file=src/a.go") || !strings.Contains(out, "line=4") {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(out, "title=HASH_MISMATCH") {
		t.Fatalf("got %q", out)
	}
}

func TestAnnotationsDenyFallback(t *testing.T) {
	t.Parallel()
	res := &Result{Verdict: Deny, Reason: "policy verification failed"}
	anns := AnnotationsFromResult(res)
	if len(anns) != 1 || anns[0].Level != "error" {
		t.Fatalf("%+v", anns)
	}
}
