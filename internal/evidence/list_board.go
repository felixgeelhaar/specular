package evidence

import (
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
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Verdict   string    `json:"verdict,omitempty"`
	SoftAllow bool      `json:"softAllow,omitempty"`
	Risk      string    `json:"risk,omitempty"`
	Attested  bool      `json:"attested"`
	Governed  bool      `json:"governed"`
	Protocol  bool      `json:"protocol"`
	Commit    string    `json:"commit,omitempty"`
	Sessions  []string  `json:"sessions,omitempty"`
	Harnesses []string  `json:"harnesses,omitempty"`
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
