package approval

import (
	"fmt"
	"strings"
	"time"
)

// ListSummary counts approval/exception records for dashboards.
type ListSummary struct {
	Total   int            `json:"total"`
	Open    int            `json:"open"`
	Closed  int            `json:"closed"`
	Expired int            `json:"expired"`
	ByType  map[string]int `json:"byType,omitempty"`
}

// ListRow is one trust-board line for `approvals list`.
type ListRow struct {
	ResourceID string     `json:"resourceId"`
	Type       string     `json:"type"`
	Status     string     `json:"status"` // open|closed|expired
	Policy     string     `json:"policy,omitempty"`
	Scope      string     `json:"scope,omitempty"`
	EvidenceID string     `json:"evidenceId,omitempty"`
	ApprovedBy string     `json:"approvedBy,omitempty"`
	ApprovedAt time.Time  `json:"approvedAt"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	Path       string     `json:"path,omitempty"`
}

// ListBoard is the JSON shape for `approvals list --json`.
type ListBoard struct {
	Summary ListSummary `json:"summary"`
	Records []ListRow   `json:"records"`
}

// BuildListBoard projects records into a trust board (order preserved).
func BuildListBoard(recs []Record, now time.Time) ListBoard {
	board := ListBoard{
		Records: make([]ListRow, 0, len(recs)),
		Summary: ListSummary{
			Total:  len(recs),
			ByType: make(map[string]int),
		},
	}
	for i := range recs {
		row := RowFromRecord(&recs[i], now)
		board.Records = append(board.Records, row)
		board.Summary.ByType[row.Type]++
		switch row.Status {
		case StatusOpen:
			board.Summary.Open++
		case StatusClosed:
			board.Summary.Closed++
		case StatusExpired:
			board.Summary.Expired++
		}
	}
	if len(board.Summary.ByType) == 0 {
		board.Summary.ByType = nil
	}
	return board
}

// RowFromRecord extracts board columns from one approval record.
func RowFromRecord(rec *Record, now time.Time) ListRow {
	var row ListRow
	if rec == nil {
		return row
	}
	row.ResourceID = rec.ResourceID
	row.Type = rec.Type
	row.Status = rec.Lifecycle(now)
	row.Policy = strings.TrimSpace(rec.Policy)
	row.Scope = strings.TrimSpace(rec.Scope)
	row.EvidenceID = strings.TrimSpace(rec.EvidenceID)
	row.ApprovedBy = strings.TrimSpace(rec.ApprovedBy)
	row.ApprovedAt = rec.ApprovedAt
	row.ExpiresAt = rec.ExpiresAt
	row.Path = rec.Path
	return row
}

// DashOr returns s or "-" when empty (human boards).
func DashOr(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// FormatEvidenceListHints returns human footer lines for rows with an EVID
// column (evidence show / explain / Soft trail pending+doctor). Empty when none bound.
func FormatEvidenceListHints(rows []ListRow) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	for _, row := range rows {
		evid := strings.TrimSpace(row.EvidenceID)
		if evid == "" {
			continue
		}
		id := strings.TrimSpace(row.ResourceID)
		if id == "" {
			id = evid
		}
		if b.Len() == 0 {
			b.WriteString("Evidence:\n")
		}
		fmt.Fprintf(&b, "  %s  specular evidence show %s\n", id, evid)
		fmt.Fprintf(&b, "       specular explain %s\n", evid)
	}
	if b.Len() == 0 {
		return ""
	}
	b.WriteString("  Trail  specular approvals pending\n")
	b.WriteString("         specular doctor\n")
	return b.String()
}

// FormatHollowSoftTrailHints returns Soft trail pending/doctor when no EVID Soft
// trail was printed (pending hollow Soft-trail parity).
func FormatHollowSoftTrailHints() string {
	return "Trail  specular approvals pending\n         specular doctor\n"
}

// FormatOpenExceptionHints returns human footer lines for open exceptions
// (approvals show / evidence show when bound / list --status open / Soft trail
// pending+doctor). Empty when none.
func FormatOpenExceptionHints(recs []Record) string {
	if len(recs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Open exceptions:\n")
	for _, rec := range recs {
		id := strings.TrimSpace(rec.ResourceID)
		if id == "" {
			continue
		}
		fmt.Fprintf(&b, "  %s  specular approvals show %s\n", id, id)
		if evid := strings.TrimSpace(rec.EvidenceID); evid != "" {
			fmt.Fprintf(&b, "       specular evidence show %s\n", evid)
			fmt.Fprintf(&b, "       specular explain %s\n", evid)
		}
	}
	b.WriteString("  List   specular approvals list --status open\n")
	b.WriteString("  Trail  specular approvals pending\n")
	b.WriteString("         specular doctor\n")
	return b.String()
}

// FormatExpires formats ExpiresAt for human boards.
func FormatExpires(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.UTC().Format("2006-01-02")
}
