package evidence

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/gate"
)

func TestBuildListBoard(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	recs := []*Record{
		{
			ID:        "ev_allow",
			CreatedAt: now,
			Commit:    "abcdef0123456789",
			Gate: &gate.Result{
				Verdict: gate.Allow,
				Risk:    gate.RiskSection{Level: "LOW"},
				Provenance: gate.ProvenanceSection{
					Attested:     true,
					Governed:     true,
					Sessions:     []string{"auth"},
					Harnesses:    []string{"claude-code"},
					ProtocolDocs: 1,
					ProtocolOK:   1,
				},
				Approvals: gate.ApprovalsSection{
					Overrules: []gate.ExceptionOverrule{{Kind: "drift"}},
				},
			},
		},
		{
			ID:        "ev_deny",
			CreatedAt: now.Add(-time.Hour),
			Gate: &gate.Result{
				Verdict: gate.Deny,
				Risk:    gate.RiskSection{Level: "HIGH"},
				Provenance: gate.ProvenanceSection{
					Attested:     false,
					Governed:     false,
					Sessions:     []string{"migrate"},
					ProtocolDocs: 1,
					ProtocolOK:   0,
				},
				Change: gate.ChangeSection{Commit: "deadbeef"},
			},
		},
		{ID: "ev_empty", CreatedAt: now.Add(-2 * time.Hour)},
	}

	board := BuildListBoard(recs)
	if board.Summary.Total != 3 || board.Summary.Allow != 1 || board.Summary.Deny != 1 || board.Summary.Other != 1 {
		t.Fatalf("summary=%+v", board.Summary)
	}
	if board.Summary.SoftAllow != 1 {
		t.Fatalf("softAllow=%d", board.Summary.SoftAllow)
	}
	allow := board.Records[0]
	if allow.Verdict != "ALLOW" || !allow.SoftAllow || allow.Risk != "LOW" || !allow.Attested || !allow.Governed || !allow.Protocol {
		t.Fatalf("allow=%+v", allow)
	}
	if allow.Commit != "abcdef0123456789" || JoinDash(allow.Sessions) != "auth" {
		t.Fatalf("allow commit/session=%q %v", allow.Commit, allow.Sessions)
	}
	deny := board.Records[1]
	if deny.Verdict != "DENY" || deny.SoftAllow || deny.Risk != "HIGH" || deny.Attested || deny.Governed || deny.Protocol {
		t.Fatalf("deny=%+v", deny)
	}
	if deny.Commit != "deadbeef" {
		t.Fatalf("deny commit=%q", deny.Commit)
	}
	empty := board.Records[2]
	if empty.Verdict != "" || empty.Risk != "" || empty.Protocol {
		t.Fatalf("empty=%+v", empty)
	}
	if ShortCommit("abcdef0") != "abcdef0" || ShortCommit("abcdef0123") != "abcdef0" || ShortCommit("") != "-" {
		t.Fatal(ShortCommit("abcdef0123"), ShortCommit(""))
	}
	if JoinDash(nil) != "-" || JoinDash([]string{"a", "b"}) != "a,b" {
		t.Fatal(JoinDash([]string{"a", "b"}))
	}
}
