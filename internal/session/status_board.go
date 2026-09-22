package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/specular/internal/evidence"
)

// StatusSummary counts sessions by lifecycle bucket for dashboards.
type StatusSummary struct {
	Working   int `json:"working"`
	Queued    int `json:"queued"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Stopped   int `json:"stopped"`
	Other     int `json:"other,omitempty"`
	Total     int `json:"total"`
}

// SessionEvidenceFlags reports sibling attestation / APP doc presence,
// worktree HEAD tip, and newest gate verdict from the Change Evidence Graph
// (fleet board → explain / evidence show).
type SessionEvidenceFlags struct {
	Attested   bool   `json:"attested"`
	App        bool   `json:"app"`                  // sibling .provenance.json present
	Commit     string `json:"commit,omitempty"`     // short worktree HEAD SHA
	Verdict    string `json:"verdict,omitempty"`    // ALLOW | DENY from newest evidence
	EvidenceID string `json:"evidenceId,omitempty"` // newest matching evidence record id
}

// StatusBoard is the JSON shape for `session status --json`: summary counts
// plus the full session list (Xirp-style board for CI/dashboards).
type StatusBoard struct {
	Summary  StatusSummary                   `json:"summary"`
	Sessions []Record                        `json:"sessions"`
	Evidence map[string]SessionEvidenceFlags `json:"evidence,omitempty"`
}

// EvidenceFlags reports whether attestation / APP docs exist for id under dir
// (typically Manager store Dir()).
func EvidenceFlags(sessionsDir, id string) SessionEvidenceFlags {
	id = filepath.Base(id)
	var f SessionEvidenceFlags
	if id == "" || id == "." || id == string(filepath.Separator) {
		return f
	}
	if _, err := os.Stat(filepath.Join(sessionsDir, id+".attestation.json")); err == nil {
		f.Attested = true
	}
	if _, err := os.Stat(filepath.Join(sessionsDir, id+".provenance.json")); err == nil {
		f.App = true
	}
	return f
}

// EvidenceFlagsFor is EvidenceFlags plus short worktree HEAD and newest gate
// verdict when repoRoot has matching Change Evidence Graph records.
func EvidenceFlagsFor(sessionsDir, repoRoot string, rec Record) SessionEvidenceFlags {
	f := EvidenceFlags(sessionsDir, rec.ID)
	f.Commit = WorktreeHEADShort(rec.WorktreePath)
	if g, ok := NewestGateBySession(repoRoot)[rec.ID]; ok {
		f.Verdict = g.Verdict
		f.EvidenceID = g.EvidenceID
	}
	return f
}

// SessionGate is the newest gate verdict for a session id.
type SessionGate struct {
	Verdict    string
	EvidenceID string
}

// NewestGateBySession returns ALLOW/DENY (and evidence id) for each session
// from the newest matching Change Evidence Graph record under repoRoot.
// Sessions appearing on older records only keep the newest hit (List order).
func NewestGateBySession(repoRoot string) map[string]SessionGate {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return nil
	}
	recs, err := evidence.List(repoRoot, evidence.ListFilter{})
	if err != nil || len(recs) == 0 {
		return nil
	}
	out := make(map[string]SessionGate)
	for _, rec := range recs {
		if rec == nil || rec.Gate == nil {
			continue
		}
		verdict := strings.TrimSpace(string(rec.Gate.Verdict))
		if verdict == "" {
			continue
		}
		for _, sid := range rec.Gate.Provenance.Sessions {
			sid = strings.TrimSpace(sid)
			if sid == "" {
				continue
			}
			if _, exists := out[sid]; exists {
				continue // newer already recorded (List is newest-first)
			}
			out[sid] = SessionGate{Verdict: verdict, EvidenceID: rec.ID}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// WorktreeHEADShort returns `git rev-parse --short HEAD` for a worktree path,
// or "" when missing / not a git dir.
func WorktreeHEADShort(worktreePath string) string {
	worktreePath = strings.TrimSpace(worktreePath)
	if worktreePath == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// YesDash renders a boolean as "yes" or "-" for human boards.
func YesDash(v bool) string {
	if v {
		return "yes"
	}
	return "-"
}

// DashOr returns s or "-" when empty (human boards).
func DashOr(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// BuildStatusBoard aggregates a session list into a dashboard board.
func BuildStatusBoard(list []Record) StatusBoard {
	return BuildStatusBoardWithEvidence(list, "", "")
}

// BuildStatusBoardWithEvidence is BuildStatusBoard plus optional
// attest/APP/commit/gate flags. sessionsDir is Manager.Store().Dir();
// repoRoot is the git repository root for .specular/evidence.
func BuildStatusBoardWithEvidence(list []Record, sessionsDir, repoRoot string) StatusBoard {
	board := StatusBoard{
		Sessions: append([]Record(nil), list...),
		Summary:  StatusSummary{Total: len(list)},
	}
	if sessionsDir != "" && len(list) > 0 {
		board.Evidence = make(map[string]SessionEvidenceFlags, len(list))
	}
	gates := NewestGateBySession(repoRoot)
	for _, s := range list {
		switch s.Status {
		case StatusWorking, StatusIdle, StatusWaiting:
			board.Summary.Working++
		case StatusQueued:
			board.Summary.Queued++
		case StatusCompleted:
			board.Summary.Completed++
		case StatusFailed:
			board.Summary.Failed++
		case StatusStopped:
			board.Summary.Stopped++
		default:
			board.Summary.Other++
		}
		if board.Evidence != nil {
			f := EvidenceFlags(sessionsDir, s.ID)
			f.Commit = WorktreeHEADShort(s.WorktreePath)
			if g, ok := gates[s.ID]; ok {
				f.Verdict = g.Verdict
				f.EvidenceID = g.EvidenceID
			}
			board.Evidence[s.ID] = f
		}
	}
	return board
}
