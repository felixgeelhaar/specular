package approval

import (
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

// FormatExpires formats ExpiresAt for human boards.
func FormatExpires(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.UTC().Format("2006-01-02")
}
