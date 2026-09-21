package session

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

// StatusBoard is the JSON shape for `session status --json`: summary counts
// plus the full session list (Xirp-style board for CI/dashboards).
type StatusBoard struct {
	Summary  StatusSummary `json:"summary"`
	Sessions []Record      `json:"sessions"`
}

// BuildStatusBoard aggregates a session list into a dashboard board.
func BuildStatusBoard(list []Record) StatusBoard {
	board := StatusBoard{
		Sessions: append([]Record(nil), list...),
		Summary:  StatusSummary{Total: len(list)},
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
	}
	return board
}
