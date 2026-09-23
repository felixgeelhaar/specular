package approval

import (
	"strings"
	"testing"
	"time"
)

func TestBuildListBoard(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	expFuture := now.Add(24 * time.Hour)
	expPast := now.Add(-time.Hour)
	closedAt := now.Add(-30 * time.Minute)
	recs := []Record{
		{
			Type: TypeException, ResourceID: "exception-open",
			ApprovedBy: "alice", ApprovedAt: now.Add(-2 * time.Hour),
			Policy: "SOC2", Scope: "auth", EvidenceID: "ev_1",
			ExpiresAt: &expFuture,
		},
		{
			Type: TypeException, ResourceID: "exception-closed",
			ApprovedBy: "bob", ApprovedAt: now.Add(-3 * time.Hour),
			ClosedAt: &closedAt, ClosedBy: "bob",
			ExpiresAt: &expFuture,
		},
		{
			Type: TypeException, ResourceID: "exception-expired",
			ApprovedBy: "carol", ApprovedAt: now.Add(-48 * time.Hour),
			ExpiresAt: &expPast,
		},
		{
			Type: TypePolicy, ResourceID: "policy-x",
			ApprovedBy: "dave", ApprovedAt: now,
		},
	}
	board := BuildListBoard(recs, now)
	if board.Summary.Total != 4 || board.Summary.Closed != 1 || board.Summary.Expired != 1 {
		t.Fatalf("summary=%+v", board.Summary)
	}
	// Non-exception policy counts as open (Lifecycle default).
	if board.Summary.Open != 2 {
		t.Fatalf("open=%d want 2", board.Summary.Open)
	}
	if board.Records[0].Status != StatusOpen || board.Records[0].EvidenceID != "ev_1" {
		t.Fatalf("open=%+v", board.Records[0])
	}
	if board.Records[1].Status != StatusClosed {
		t.Fatalf("closed=%+v", board.Records[1])
	}
	if board.Records[2].Status != StatusExpired {
		t.Fatalf("expired=%+v", board.Records[2])
	}
	if board.Summary.ByType[TypeException] != 3 || board.Summary.ByType[TypePolicy] != 1 {
		t.Fatalf("byType=%v", board.Summary.ByType)
	}
	if DashOr("") != "-" || DashOr("x") != "x" {
		t.Fatal(DashOr(""), DashOr("x"))
	}
	if FormatExpires(nil) != "-" || FormatExpires(&expFuture) != "2026-09-24" {
		t.Fatal(FormatExpires(&expFuture))
	}

	hints := FormatEvidenceListHints(board.Records)
	for _, want := range []string{
		"Evidence:",
		"exception-open  specular evidence show ev_1",
		"specular explain ev_1",
	} {
		if !strings.Contains(hints, want) {
			t.Fatalf("missing %q:\n%s", want, hints)
		}
	}
	if strings.Contains(hints, "exception-closed") || strings.Contains(hints, "exception-expired") {
		t.Fatalf("unexpected unbound row in evidence hints:\n%s", hints)
	}
	if FormatEvidenceListHints(nil) != "" {
		t.Fatal("expected empty hints")
	}
	if FormatEvidenceListHints([]ListRow{{ResourceID: "x"}}) != "" {
		t.Fatal("expected empty hints without EvidenceID")
	}

	openHints := FormatOpenExceptionHints([]Record{
		{ResourceID: "exception-open", EvidenceID: "ev_1"},
		{ResourceID: "exception-bare"},
	})
	for _, want := range []string{
		"Open exceptions:",
		"exception-open  specular approvals show exception-open",
		"specular evidence show ev_1",
		"specular explain ev_1",
		"exception-bare  specular approvals show exception-bare",
		"List   specular approvals list --status open",
		"Trail  specular approvals pending",
		"specular doctor",
	} {
		if !strings.Contains(openHints, want) {
			t.Fatalf("missing %q:\n%s", want, openHints)
		}
	}
	bareIdx := strings.Index(openHints, "exception-bare")
	if bareIdx < 0 {
		t.Fatal("missing bare exception")
	}
	if strings.Contains(openHints[bareIdx:], "evidence show") {
		t.Fatalf("bare exception should not invent evidence show:\n%s", openHints[bareIdx:])
	}
	if FormatOpenExceptionHints(nil) != "" {
		t.Fatal("expected empty open hints")
	}
}
