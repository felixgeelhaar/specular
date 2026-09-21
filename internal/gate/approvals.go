package gate

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/approval"
)

// ApprovalsSection summarizes recent local approvals / exceptions for the
// gate board and evidence trail (PRODUCT_INTENT §17–§18). Advisory only —
// never flips ALLOW↔DENY by itself.
type ApprovalsSection struct {
	Count      int               `json:"count"`
	Recent     []ApprovalSummary `json:"recent,omitempty"`
	Exceptions []ApprovalSummary `json:"exceptions,omitempty"`
	Note       string            `json:"note,omitempty"`
}

// ApprovalSummary is a compact approval/exception trail entry.
type ApprovalSummary struct {
	Type       string `json:"type"`
	ResourceID string `json:"resource_id"`
	ApprovedBy string `json:"approved_by,omitempty"`
	ApprovedAt string `json:"approved_at,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Policy     string `json:"policy,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	Path       string `json:"path,omitempty"`
	Expired    bool   `json:"expired,omitempty"`
}

const maxTrailApprovals = 5

func discoverApprovals(root string) ApprovalsSection {
	now := time.Now().UTC()
	recs, err := approval.List(root)
	if err != nil || len(recs) == 0 {
		return ApprovalsSection{
			Note: "no local approval/exception records under .specular/approvals/",
		}
	}

	sec := ApprovalsSection{Count: len(recs)}
	limit := maxTrailApprovals
	if limit > len(recs) {
		limit = len(recs)
	}
	for i := 0; i < limit; i++ {
		sec.Recent = append(sec.Recent, summaryFromRecord(recs[i], now))
	}
	for _, rec := range recs {
		if rec.Type != approval.TypeException {
			continue
		}
		if rec.IsExpired(now) {
			continue
		}
		sec.Exceptions = append(sec.Exceptions, summaryFromRecord(rec, now))
		if len(sec.Exceptions) >= maxTrailApprovals {
			break
		}
	}
	switch {
	case len(sec.Exceptions) > 0:
		sec.Note = fmt.Sprintf("%d open exception(s); trail is advisory — does not change gate verdict", len(sec.Exceptions))
	default:
		sec.Note = "recent approvals listed; no open exceptions"
	}
	return sec
}

func summaryFromRecord(rec approval.Record, now time.Time) ApprovalSummary {
	s := ApprovalSummary{
		Type:       rec.Type,
		ResourceID: rec.ResourceID,
		ApprovedBy: rec.ApprovedBy,
		Reason:     rec.Reason,
		Scope:      rec.Scope,
		Policy:     rec.Policy,
		Path:       rec.Path,
		Expired:    rec.IsExpired(now),
	}
	if !rec.ApprovedAt.IsZero() {
		s.ApprovedAt = rec.ApprovedAt.UTC().Format(time.RFC3339)
	}
	if rec.ExpiresAt != nil {
		s.ExpiresAt = rec.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return s
}

func writeApprovalsSection(b *strings.Builder, sec ApprovalsSection, verdict Verdict) {
	b.WriteString("Approvals\n")
	if sec.Count == 0 {
		b.WriteString("  Status         none recorded\n")
		if verdict == Deny {
			b.WriteString("  Hint           specular approve exception-<id> --reason \"...\" --scope \"...\"\n")
		}
		if sec.Note != "" {
			fmt.Fprintf(b, "  Note           %s\n", sec.Note)
		}
		return
	}
	fmt.Fprintf(b, "  Records        %d\n", sec.Count)
	if len(sec.Exceptions) > 0 {
		b.WriteString("  OpenExceptions\n")
		for _, ex := range sec.Exceptions {
			fmt.Fprintf(b, "    ⚠ %s", ex.ResourceID)
			if ex.Reason != "" {
				fmt.Fprintf(b, " — %s", ex.Reason)
			}
			b.WriteString("\n")
			writeApprovalDetail(b, ex, "      ")
		}
	}
	if len(sec.Recent) > 0 {
		b.WriteString("  Recent\n")
		for _, r := range sec.Recent {
			mark := "•"
			if r.Type == approval.TypeException {
				if r.Expired {
					mark = "·"
				} else {
					mark = "⚠"
				}
			} else {
				mark = "✓"
			}
			fmt.Fprintf(b, "    %s %s (%s)", mark, r.ResourceID, r.Type)
			if r.ApprovedBy != "" {
				fmt.Fprintf(b, " by %s", r.ApprovedBy)
			}
			b.WriteString("\n")
		}
	}
	if verdict == Deny && len(sec.Exceptions) == 0 {
		b.WriteString("  Hint           record an exception: specular approve exception-<id> --reason \"...\" --scope \"...\"\n")
	}
	if sec.Note != "" {
		fmt.Fprintf(b, "  Note           %s\n", sec.Note)
	}
}

func writeApprovalDetail(b *strings.Builder, ex ApprovalSummary, indent string) {
	if ex.Scope != "" {
		fmt.Fprintf(b, "%sscope=%s\n", indent, ex.Scope)
	}
	if ex.Policy != "" {
		fmt.Fprintf(b, "%spolicy=%s\n", indent, ex.Policy)
	}
	if ex.ApprovedBy != "" {
		fmt.Fprintf(b, "%sby %s\n", indent, ex.ApprovedBy)
	}
	if ex.ExpiresAt != "" {
		fmt.Fprintf(b, "%sexpires %s\n", indent, ex.ExpiresAt)
	}
	if ex.Path != "" {
		fmt.Fprintf(b, "%s%s\n", indent, ex.Path)
	}
}
