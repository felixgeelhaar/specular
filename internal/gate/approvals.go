package gate

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/approval"
)

// ApprovalsSection summarizes recent local approvals / exceptions for the
// gate board and evidence trail (PRODUCT_INTENT §17–§18). Unmatched
// exceptions stay advisory; bound exceptions may soft-ALLOW a DENY via
// Overrules (see exception_match.go).
type ApprovalsSection struct {
	Count      int                 `json:"count"`
	Recent     []ApprovalSummary   `json:"recent,omitempty"`
	Exceptions []ApprovalSummary   `json:"exceptions,omitempty"`
	Overrules  []ExceptionOverrule `json:"overrules,omitempty"`
	Note       string              `json:"note,omitempty"`
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
		if !rec.IsOpen(now) {
			continue
		}
		sec.Exceptions = append(sec.Exceptions, summaryFromRecord(rec, now))
		if len(sec.Exceptions) >= maxTrailApprovals {
			break
		}
	}
	switch {
	case len(sec.Exceptions) > 0:
		sec.Note = fmt.Sprintf("%d open exception(s); unmatched stay advisory — bind --policy/--scope to soft-ALLOW", len(sec.Exceptions))
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
			b.WriteString("  Hint           specular approve exception-<id> --reason \"...\" --scope \"...\" --policy drift|policy|<check>\n")
		}
		if sec.Note != "" {
			fmt.Fprintf(b, "  Note           %s\n", sec.Note)
		}
		return
	}
	fmt.Fprintf(b, "  Records        %d\n", sec.Count)
	writeOverrules(b, sec.Overrules)
	writeOpenExceptions(b, sec.Exceptions)
	writeRecentApprovals(b, sec.Recent)
	if verdict == Deny && len(sec.Exceptions) == 0 {
		b.WriteString("  Hint           record an exception: specular approve exception-<id> --reason \"...\" --scope \"...\" --policy …\n")
	}
	if sec.Note != "" {
		fmt.Fprintf(b, "  Note           %s\n", sec.Note)
	}
}

func writeOverrules(b *strings.Builder, overrules []ExceptionOverrule) {
	if len(overrules) == 0 {
		return
	}
	b.WriteString("  Overruled\n")
	for _, o := range overrules {
		fmt.Fprintf(b, "    ⚠ %s soft-ALLOW %s (%s)\n", o.ResourceID, o.Kind, o.Binding)
	}
}

func writeOpenExceptions(b *strings.Builder, exceptions []ApprovalSummary) {
	if len(exceptions) == 0 {
		return
	}
	b.WriteString("  OpenExceptions\n")
	for _, ex := range exceptions {
		fmt.Fprintf(b, "    ⚠ %s", ex.ResourceID)
		if ex.Reason != "" {
			fmt.Fprintf(b, " — %s", ex.Reason)
		}
		b.WriteString("\n")
		writeApprovalDetail(b, ex, "      ")
	}
}

func writeRecentApprovals(b *strings.Builder, recent []ApprovalSummary) {
	if len(recent) == 0 {
		return
	}
	b.WriteString("  Recent\n")
	for _, r := range recent {
		fmt.Fprintf(b, "    %s %s (%s)", approvalMark(r), r.ResourceID, r.Type)
		if r.ApprovedBy != "" {
			fmt.Fprintf(b, " by %s", r.ApprovedBy)
		}
		b.WriteString("\n")
	}
}

func approvalMark(r ApprovalSummary) string {
	switch {
	case r.Type == approval.TypeException && r.Expired:
		return "·"
	case r.Type == approval.TypeException:
		return "⚠"
	default:
		return "✓"
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
