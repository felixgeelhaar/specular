package approval

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestBindEvidence(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	exp := now.Add(24 * time.Hour)
	path, err := Write(root, &Record{
		Type: TypeException, ResourceID: "exception-drift",
		ApprovedBy: "alice", ApprovedAt: now, ExpiresAt: &exp,
		Reason: "hotfix", Scope: "a.go", Policy: "drift",
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := BindEvidence(root, []string{"exception-drift", "exception-missing"}, "ev_soft_1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("bound=%d", n)
	}
	found, err := FindByResourceID(root, "exception-drift")
	if err != nil || len(found) != 1 || found[0].EvidenceID != "ev_soft_1" {
		t.Fatalf("found=%+v err=%v", found, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "evidence_id: ev_soft_1") {
		t.Fatalf("yaml missing evidence_id:\n%s", data)
	}
	n2, err := BindEvidence(root, []string{"exception-drift"}, "ev_soft_1")
	if err != nil || n2 != 0 {
		t.Fatalf("idempotent n=%d err=%v", n2, err)
	}
	if n, err := BindEvidence(root, nil, "ev_x"); err != nil || n != 0 {
		t.Fatalf("empty ids n=%d err=%v", n, err)
	}
}

func TestFilterByEvidenceSoftResourceIDs(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	recs := []Record{
		{Type: TypeException, ResourceID: "exception-drift", ApprovedAt: now},
		{Type: TypeException, ResourceID: "exception-other", ApprovedAt: now, EvidenceID: "ev_other"},
	}
	got := FilterByEvidence(recs, "ev_join", "exception-drift")
	if len(got) != 1 || got[0].ResourceID != "exception-drift" {
		t.Fatalf("soft join=%v", idsOfApprovals(got))
	}
	got = FilterByEvidence(recs, "ev_other")
	if len(got) != 1 || got[0].ResourceID != "exception-other" {
		t.Fatalf("field=%v", idsOfApprovals(got))
	}
}
