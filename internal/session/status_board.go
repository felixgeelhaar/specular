package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
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
// worktree HEAD tip, and newest gate verdict / soft-ALLOW / risk / protocol
// from the Change Evidence Graph (fleet board → explain / evidence show).
type SessionEvidenceFlags struct {
	Attested     bool     `json:"attested"`
	App          bool     `json:"app"`                    // sibling .provenance.json present
	Commit       string   `json:"commit,omitempty"`       // short worktree HEAD SHA
	Verdict      string   `json:"verdict,omitempty"`      // ALLOW | DENY from newest evidence
	EvidenceID   string   `json:"evidenceId,omitempty"`   // newest matching evidence record id
	SoftAllow    bool     `json:"softAllow,omitempty"`    // newest evidence has exception overrules
	SoftAllowIDs []string `json:"softAllowIds,omitempty"` // soft-ALLOW ResourceIDs → approvals show
	Risk         string   `json:"risk,omitempty"`         // NONE|LOW|MEDIUM|HIGH|CRITICAL
	Protocol     bool     `json:"protocol,omitempty"`     // newest evidence APP docs schema+bound
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
// verdict / soft-ALLOW / risk / protocol when repoRoot has matching records.
func EvidenceFlagsFor(sessionsDir, repoRoot string, rec Record) SessionEvidenceFlags {
	f := EvidenceFlags(sessionsDir, rec.ID)
	f.Commit = WorktreeHEADShort(rec.WorktreePath)
	if g, ok := NewestGateBySession(repoRoot)[rec.ID]; ok {
		applySessionGate(&f, g)
	}
	return f
}

// SessionGate is the newest gate verdict for a session id.
type SessionGate struct {
	Verdict      string
	EvidenceID   string
	SoftAllow    bool
	SoftAllowIDs []string
	Risk         string
	Protocol     bool
}

func applySessionGate(f *SessionEvidenceFlags, g SessionGate) {
	f.Verdict = g.Verdict
	f.EvidenceID = g.EvidenceID
	f.SoftAllow = g.SoftAllow
	f.SoftAllowIDs = append([]string(nil), g.SoftAllowIDs...)
	f.Risk = g.Risk
	f.Protocol = g.Protocol
}

// NewestGateBySession returns ALLOW/DENY (evidence id, soft-ALLOW, risk,
// protocol) for each session from the newest matching Change Evidence Graph
// record under repoRoot. Sessions on older records only keep the newest hit.
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
		soft := len(rec.Gate.Approvals.Overrules) > 0
		softIDs := gate.SoftAllowResourceIDs(rec.Gate.Approvals.Overrules)
		risk := strings.ToUpper(strings.TrimSpace(rec.Gate.Risk.Level))
		if risk == "" {
			risk = "NONE"
		}
		protocol := protocolDocsOK(rec.Gate)
		for _, sid := range rec.Gate.Provenance.Sessions {
			sid = strings.TrimSpace(sid)
			if sid == "" {
				continue
			}
			if _, exists := out[sid]; exists {
				continue // newer already recorded (List is newest-first)
			}
			out[sid] = SessionGate{
				Verdict:      verdict,
				EvidenceID:   rec.ID,
				SoftAllow:    soft,
				SoftAllowIDs: softIDs,
				Risk:         risk,
				Protocol:     protocol,
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// protocolDocsOK mirrors evidence list --protocol: APP docs present with
// schema+bound counts (ProtocolOK >= ProtocolDocs > 0).
func protocolDocsOK(g *gate.Result) bool {
	if g == nil {
		return false
	}
	docs := g.Provenance.ProtocolDocs
	ok := g.Provenance.ProtocolOK
	return docs > 0 && ok >= docs
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

// GateDetails is SessionEvidenceFlags plus DENY Next steps for session show.
type GateDetails struct {
	SessionEvidenceFlags
	NextSteps []string `json:"nextSteps,omitempty"`
}

// GateDetailsFor is EvidenceFlagsFor plus DenyNextSteps from the loaded
// newest matching evidence record. SoftAllowIDs come from SessionEvidenceFlags
// (NewestGateBySession); refill from the evidence record if missing.
func GateDetailsFor(sessionsDir, repoRoot string, rec Record) GateDetails {
	f := EvidenceFlagsFor(sessionsDir, repoRoot, rec)
	d := GateDetails{SessionEvidenceFlags: f}
	if strings.TrimSpace(f.EvidenceID) == "" || strings.TrimSpace(repoRoot) == "" {
		return d
	}
	erec, err := evidence.Load(repoRoot, f.EvidenceID)
	if err != nil || erec == nil || erec.Gate == nil {
		return d
	}
	d.NextSteps = gate.DenyNextSteps(erec.Gate)
	if len(d.SoftAllowIDs) == 0 {
		d.SoftAllowIDs = gate.SoftAllowResourceIDs(erec.Gate.Approvals.Overrules)
	}
	return d
}

// HasSurface reports whether any board/show evidence field is populated.
func (d GateDetails) HasSurface() bool {
	return d.Attested || d.App || d.Commit != "" || d.Verdict != "" ||
		d.EvidenceID != "" || d.SoftAllow || d.Risk != "" || d.Protocol ||
		len(d.NextSteps) > 0 || len(d.SoftAllowIDs) > 0
}

// FormatSoftAllowBoardHints returns human footer lines for Soft=yes rows
// (approvals list --evidence / approvals show SoftAllowIDs / evidence show /
// explain --session / session show / Soft trail pending+doctor). Empty when none.
func FormatSoftAllowBoardHints(sessions []Record, evidence map[string]SessionEvidenceFlags) string {
	if len(sessions) == 0 || len(evidence) == 0 {
		return ""
	}
	var b strings.Builder
	for _, s := range sessions {
		ev := evidence[s.ID]
		if !ev.SoftAllow {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("Soft-ALLOW:\n")
		}
		if id := strings.TrimSpace(ev.EvidenceID); id != "" {
			fmt.Fprintf(&b, "  %s  specular approvals list --evidence %s\n", s.ID, id)
		} else {
			fmt.Fprintf(&b, "  %s  specular approvals list --status open\n", s.ID)
		}
		for _, aid := range ev.SoftAllowIDs {
			aid = strings.TrimSpace(aid)
			if aid == "" {
				continue
			}
			fmt.Fprintf(&b, "       specular approvals show %s\n", aid)
		}
		if id := strings.TrimSpace(ev.EvidenceID); id != "" {
			fmt.Fprintf(&b, "       specular evidence show %s\n", id)
			fmt.Fprintf(&b, "       specular explain --session %s\n", s.ID)
		}
		fmt.Fprintf(&b, "       specular session show %s\n", s.ID)
	}
	if b.Len() == 0 {
		return ""
	}
	fmt.Fprintf(&b, "  Trail  %s\n", gate.SoftAllowPendingHint)
	fmt.Fprintf(&b, "         %s\n", gate.SoftAllowDoctorHint)
	return b.String()
}

// FormatDenySoftTrailHints returns human footer lines for DENY Soft=no rows
// (explain --session / evidence show / session show / Soft trail pending+doctor).
// Soft=yes Soft-ALLOW footer already covers soft-ALLOW Soft trail; empty when none.
func FormatDenySoftTrailHints(sessions []Record, evidence map[string]SessionEvidenceFlags) string {
	if len(sessions) == 0 || len(evidence) == 0 {
		return ""
	}
	var b strings.Builder
	for _, s := range sessions {
		ev := evidence[s.ID]
		if ev.SoftAllow || !strings.EqualFold(strings.TrimSpace(ev.Verdict), "DENY") {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("DENY Soft trail:\n")
		}
		if id := strings.TrimSpace(ev.EvidenceID); id != "" {
			fmt.Fprintf(&b, "  %s  specular evidence show %s\n", s.ID, id)
			fmt.Fprintf(&b, "       specular explain --session %s\n", s.ID)
		} else {
			fmt.Fprintf(&b, "  %s  specular explain --session %s\n", s.ID, s.ID)
		}
		fmt.Fprintf(&b, "       specular session show %s\n", s.ID)
	}
	if b.Len() == 0 {
		return ""
	}
	fmt.Fprintf(&b, "  Trail  %s\n", gate.SoftAllowPendingHint)
	fmt.Fprintf(&b, "         %s\n", gate.SoftAllowDoctorHint)
	return b.String()
}

// BuildStatusBoard aggregates a session list into a dashboard board.
func BuildStatusBoard(list []Record) StatusBoard {
	return BuildStatusBoardWithEvidence(list, "", "")
}

// BuildStatusBoardWithEvidence is BuildStatusBoard plus optional
// attest/APP/commit/gate/soft-ALLOW/risk/protocol flags. sessionsDir is Manager.Store().Dir();
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
				applySessionGate(&f, g)
			}
			board.Evidence[s.ID] = f
		}
	}
	return board
}

// BoardFilter selects sessions on the fleet board. Zero-valued fields mean no
// constraint. Gate-scoped fields (verdict/soft-allow/protocol/risk) require a
// newest evidence hit; session-scoped fields use Record / sibling flags.
type BoardFilter struct {
	Verdict   string // ALLOW or DENY
	Harness   string // case-insensitive substring on Record.Harness
	SoftAllow *bool
	Attested  *bool
	Governed  *bool // Record.Governed (GOV column)
	Protocol  *bool
	RiskLevel string // NONE|LOW|MEDIUM|HIGH|CRITICAL
}

// Active reports whether any board filter constraint is set.
func (f BoardFilter) Active() bool {
	return f.Verdict != "" || strings.TrimSpace(f.Harness) != "" ||
		f.SoftAllow != nil || f.Attested != nil || f.Governed != nil ||
		f.Protocol != nil || strings.TrimSpace(f.RiskLevel) != ""
}

// Validate checks verdict / risk tokens.
func (f BoardFilter) Validate() error {
	if f.Verdict != "" {
		switch strings.ToUpper(strings.TrimSpace(f.Verdict)) {
		case "ALLOW", "DENY":
		default:
			return fmt.Errorf("session: invalid verdict %q (want ALLOW or DENY)", f.Verdict)
		}
	}
	if level := strings.TrimSpace(f.RiskLevel); level != "" {
		switch strings.ToUpper(level) {
		case "NONE", "LOW", "MEDIUM", "HIGH", "CRITICAL":
		default:
			return fmt.Errorf("session: invalid risk %q (want NONE|LOW|MEDIUM|HIGH|CRITICAL)", level)
		}
	}
	return nil
}

// Match reports whether rec + evidence flags satisfy the filter.
func (f BoardFilter) Match(rec Record, ev SessionEvidenceFlags) bool {
	return f.matchSession(rec, ev) && f.matchGate(ev)
}

func (f BoardFilter) matchSession(rec Record, ev SessionEvidenceFlags) bool {
	if sub := strings.TrimSpace(f.Harness); sub != "" {
		if !strings.Contains(strings.ToLower(rec.Harness), strings.ToLower(sub)) {
			return false
		}
	}
	if f.Governed != nil && rec.Governed != *f.Governed {
		return false
	}
	if f.Attested != nil && ev.Attested != *f.Attested {
		return false
	}
	return true
}

func (f BoardFilter) matchGate(ev SessionEvidenceFlags) bool {
	needsGate := f.Verdict != "" || f.SoftAllow != nil || f.Protocol != nil || strings.TrimSpace(f.RiskLevel) != ""
	if !needsGate {
		return true
	}
	hasGate := ev.Verdict != "" || ev.EvidenceID != ""
	if !hasGate {
		return false
	}
	if f.Verdict != "" && !strings.EqualFold(ev.Verdict, strings.TrimSpace(f.Verdict)) {
		return false
	}
	if f.SoftAllow != nil && ev.SoftAllow != *f.SoftAllow {
		return false
	}
	if f.Protocol != nil && ev.Protocol != *f.Protocol {
		return false
	}
	return f.matchRisk(ev.Risk)
}

func (f BoardFilter) matchRisk(got string) bool {
	level := strings.TrimSpace(f.RiskLevel)
	if level == "" {
		return true
	}
	if got == "" {
		got = "NONE"
	}
	return strings.EqualFold(got, level)
}

// FilterSessions returns sessions matching filter (order preserved).
// evidence may be nil when filter is inactive; when active, missing map
// entries are treated as empty flags.
func FilterSessions(list []Record, evidence map[string]SessionEvidenceFlags, filter BoardFilter) []Record {
	if !filter.Active() {
		return append([]Record(nil), list...)
	}
	out := make([]Record, 0, len(list))
	for _, s := range list {
		var ev SessionEvidenceFlags
		if evidence != nil {
			ev = evidence[s.ID]
		}
		if filter.Match(s, ev) {
			out = append(out, s)
		}
	}
	return out
}

// EvidenceMapFor builds per-session flags (attest/APP/commit + newest gate).
func EvidenceMapFor(list []Record, sessionsDir, repoRoot string) map[string]SessionEvidenceFlags {
	if len(list) == 0 {
		return nil
	}
	out := make(map[string]SessionEvidenceFlags, len(list))
	gates := NewestGateBySession(repoRoot)
	for _, s := range list {
		f := EvidenceFlags(sessionsDir, s.ID)
		f.Commit = WorktreeHEADShort(s.WorktreePath)
		if g, ok := gates[s.ID]; ok {
			applySessionGate(&f, g)
		}
		out[s.ID] = f
	}
	return out
}
