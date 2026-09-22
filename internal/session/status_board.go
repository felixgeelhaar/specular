package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// SessionEvidenceFlags reports sibling attestation / APP doc presence and
// worktree HEAD tip for a session (fleet board → explain <sha>).
type SessionEvidenceFlags struct {
	Attested bool   `json:"attested"`
	App      bool   `json:"app"`              // sibling .provenance.json present
	Commit   string `json:"commit,omitempty"` // short worktree HEAD SHA
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

// EvidenceFlagsFor is EvidenceFlags plus short worktree HEAD when present.
func EvidenceFlagsFor(sessionsDir string, rec Record) SessionEvidenceFlags {
	f := EvidenceFlags(sessionsDir, rec.ID)
	f.Commit = WorktreeHEADShort(rec.WorktreePath)
	return f
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
	return BuildStatusBoardWithEvidence(list, "")
}

// BuildStatusBoardWithEvidence is BuildStatusBoard plus optional attest/APP/commit flags.
func BuildStatusBoardWithEvidence(list []Record, sessionsDir string) StatusBoard {
	board := StatusBoard{
		Sessions: append([]Record(nil), list...),
		Summary:  StatusSummary{Total: len(list)},
	}
	if sessionsDir != "" && len(list) > 0 {
		board.Evidence = make(map[string]SessionEvidenceFlags, len(list))
	}
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
			board.Evidence[s.ID] = EvidenceFlagsFor(sessionsDir, s)
		}
	}
	return board
}
