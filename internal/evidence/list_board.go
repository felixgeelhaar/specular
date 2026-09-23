package evidence

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/gate"
)

// ListSummary counts evidence records by gate trust bucket for dashboards.
type ListSummary struct {
	Total     int `json:"total"`
	Allow     int `json:"allow"`
	Deny      int `json:"deny"`
	SoftAllow int `json:"softAllow"`
	Other     int `json:"other,omitempty"`
}

// ListRow is one trust-board line for `evidence list` (session status parity).
type ListRow struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	Verdict      string    `json:"verdict,omitempty"`
	SoftAllow    bool      `json:"softAllow,omitempty"`
	SoftAllowIDs []string  `json:"softAllowIds,omitempty"` // soft-ALLOW ResourceIDs
	Risk         string    `json:"risk,omitempty"`
	Attested     bool      `json:"attested"`
	Governed     bool      `json:"governed"`
	Protocol     bool      `json:"protocol"`
	Commit       string    `json:"commit,omitempty"`
	Sessions     []string  `json:"sessions,omitempty"`
	Harnesses    []string  `json:"harnesses,omitempty"`
}

// ListBoard is the JSON shape for `evidence list --json`.
type ListBoard struct {
	Summary ListSummary `json:"summary"`
	Records []ListRow   `json:"records"`
}

// BuildListBoard projects records into a trust board (order preserved).
func BuildListBoard(recs []*Record) ListBoard {
	board := ListBoard{
		Records: make([]ListRow, 0, len(recs)),
		Summary: ListSummary{Total: len(recs)},
	}
	for _, rec := range recs {
		row := RowFromRecord(rec)
		board.Records = append(board.Records, row)
		switch strings.ToUpper(row.Verdict) {
		case string(gate.Allow):
			board.Summary.Allow++
		case string(gate.Deny):
			board.Summary.Deny++
		default:
			board.Summary.Other++
		}
		if row.SoftAllow {
			board.Summary.SoftAllow++
		}
	}
	return board
}

// RowFromRecord extracts trust columns from one evidence record.
func RowFromRecord(rec *Record) ListRow {
	var row ListRow
	if rec == nil {
		return row
	}
	row.ID = rec.ID
	row.CreatedAt = rec.CreatedAt
	row.Commit = strings.TrimSpace(rec.Commit)
	if rec.Gate == nil {
		return row
	}
	g := rec.Gate
	row.Verdict = strings.TrimSpace(string(g.Verdict))
	row.SoftAllow = len(g.Approvals.Overrules) > 0
	row.SoftAllowIDs = gate.SoftAllowResourceIDs(g.Approvals.Overrules)
	row.Risk = strings.ToUpper(strings.TrimSpace(g.Risk.Level))
	if row.Risk == "" && row.Verdict != "" {
		row.Risk = "NONE"
	}
	row.Attested = g.Provenance.Attested
	row.Governed = g.Provenance.Governed
	row.Protocol = protocolDocsOK(rec)
	if row.Commit == "" {
		row.Commit = strings.TrimSpace(g.Change.Commit)
	}
	row.Sessions = append([]string(nil), g.Provenance.Sessions...)
	row.Harnesses = append([]string(nil), g.Provenance.Harnesses...)
	return row
}

// FormatSoftAllowListHints returns human footer lines for Soft=yes evidence
// rows (approvals list --evidence / approvals show SoftAllowIDs / evidence show /
// explain / Soft trail pending+doctor). Empty when none.
func FormatSoftAllowListHints(rows []ListRow) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	for _, row := range rows {
		if !row.SoftAllow {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("Soft-ALLOW:\n")
		}
		id := strings.TrimSpace(row.ID)
		if id == "" {
			continue
		}
		fmt.Fprintf(&b, "  %s  specular approvals list --evidence %s\n", id, id)
		for _, aid := range row.SoftAllowIDs {
			aid = strings.TrimSpace(aid)
			if aid == "" {
				continue
			}
			fmt.Fprintf(&b, "       specular approvals show %s\n", aid)
		}
		fmt.Fprintf(&b, "       specular evidence show %s\n", id)
		fmt.Fprintf(&b, "       specular explain %s\n", id)
	}
	if b.Len() == 0 {
		return ""
	}
	fmt.Fprintf(&b, "  Trail  %s\n", gate.SoftAllowPendingHint)
	fmt.Fprintf(&b, "         %s\n", gate.SoftAllowDoctorHint)
	return b.String()
}

// FormatDenySoftTrailHints returns human footer lines for DENY Soft=no evidence
// rows (evidence show / explain / Soft trail pending+doctor). Soft-ALLOW Soft
// trail is covered by FormatSoftAllowListHints; empty when none.
func FormatDenySoftTrailHints(rows []ListRow) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	for _, row := range rows {
		if row.SoftAllow || !strings.EqualFold(strings.TrimSpace(row.Verdict), "DENY") {
			continue
		}
		id := strings.TrimSpace(row.ID)
		if id == "" {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("DENY Soft trail:\n")
		}
		fmt.Fprintf(&b, "  %s  specular evidence show %s\n", id, id)
		fmt.Fprintf(&b, "       specular explain %s\n", id)
	}
	if b.Len() == 0 {
		return ""
	}
	fmt.Fprintf(&b, "  Trail  %s\n", gate.SoftAllowPendingHint)
	fmt.Fprintf(&b, "         %s\n", gate.SoftAllowDoctorHint)
	return b.String()
}

// ShortCommit truncates a SHA for human boards (empty → "-").
func ShortCommit(sha string) string {
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return "-"
	}
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// JoinDash joins strings with ", " or returns "-" when empty.
func JoinDash(parts []string) string {
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return "-"
	}
	return strings.Join(out, ",")
}
