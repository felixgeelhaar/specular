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

// SoftAllowPolicyHint returns the --policy token(s) to suggest on a DENY
// Approvals Hint. Prefer a specific bind when one section failed; otherwise
// list all soft-ALLOW kinds (including provenance).
func SoftAllowPolicyHint(res *Result) string {
	if res == nil {
		return "drift|policy|risk|provenance"
	}
	var fails []string
	if res.Drift.Status == StatusFail {
		fails = append(fails, "drift")
	}
	if res.Policy.Status == StatusFail {
		fails = append(fails, "policy")
	}
	if res.Risk.Enforced && len(res.Risk.Missing) > 0 {
		fails = append(fails, "risk")
	}
	if res.Provenance.Status == StatusFail {
		fails = append(fails, "provenance")
	}
	switch len(fails) {
	case 0:
		return "drift|policy|risk|provenance"
	case 1:
		return fails[0]
	default:
		return strings.Join(fails, "|")
	}
}

func writeApprovalsSection(b *strings.Builder, res *Result, evidenceID string) {
	sec := ApprovalsSection{}
	verdict := Verdict("")
	if res != nil {
		sec = res.Approvals
		verdict = res.Verdict
	}
	b.WriteString("Approvals\n")
	if sec.Count == 0 {
		b.WriteString("  Status         none recorded\n")
		if verdict == Deny {
			fmt.Fprintf(b, "  Hint           specular approve exception-<id> --reason \"...\" --scope \"...\" --policy %s\n",
				SoftAllowPolicyHint(res))
			writeDenyApprovalsSoftTrail(b, "  ")
		}
		if sec.Note != "" {
			fmt.Fprintf(b, "  Note           %s\n", sec.Note)
		}
		return
	}
	fmt.Fprintf(b, "  Records        %d\n", sec.Count)
	writeOverrules(b, sec.Overrules, evidenceID)
	writeOpenExceptions(b, sec.Exceptions)
	if len(sec.Overrules) == 0 && len(sec.Exceptions) > 0 {
		writeOpenExceptionSoftTrail(b, sec.Exceptions, "    ")
	}
	writeRecentApprovals(b, sec.Recent)
	if verdict == Deny && len(sec.Exceptions) == 0 && len(sec.Overrules) == 0 {
		fmt.Fprintf(b, "  Hint           record an exception: specular approve exception-<id> --reason \"...\" --scope \"...\" --policy %s\n",
			SoftAllowPolicyHint(res))
		writeDenyApprovalsSoftTrail(b, "  ")
	}
	if sec.Note != "" {
		fmt.Fprintf(b, "  Note           %s\n", sec.Note)
	}
}

// writeDenyApprovalsSoftTrail jumps DENY SoftAllow Hint paths to pending/doctor
// when no Soft Overruled / OpenExceptions Soft trail was printed.
func writeDenyApprovalsSoftTrail(b *strings.Builder, indent string) {
	fmt.Fprintf(b, "%sPending        %s\n", indent, SoftAllowPendingHint)
	fmt.Fprintf(b, "%sDoctor         %s\n", indent, SoftAllowDoctorHint)
}

func writeOverrules(b *strings.Builder, overrules []ExceptionOverrule, evidenceID string) {
	if len(overrules) == 0 {
		return
	}
	b.WriteString("  Overruled\n")
	for _, o := range overrules {
		fmt.Fprintf(b, "    ⚠ %s soft-ALLOW %s (%s)\n", o.ResourceID, o.Kind, o.Binding)
	}
	writeSoftAllowBoardHints(b, overrules, evidenceID, "    ")
}

// SoftAllowResourceIDs returns deduped soft-ALLOW exception ids (explain/gate board jumps).
func SoftAllowResourceIDs(overrules []ExceptionOverrule) []string {
	if len(overrules) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(overrules))
	out := make([]string, 0, len(overrules))
	for _, o := range overrules {
		id := strings.TrimSpace(o.ResourceID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// SoftAllowListHint returns the approvals list command for soft-ALLOW jumps.
// Prefer --evidence when a Change Evidence Graph id is known (#154).
func SoftAllowListHint(evidenceID string) string {
	if id := strings.TrimSpace(evidenceID); id != "" {
		return "specular approvals list --evidence " + id
	}
	return "specular approvals list --status open"
}

// SoftAllowPendingHint is the Soft-trail jump to approvals pending.
const SoftAllowPendingHint = "specular approvals pending"

// SoftAllowDoctorHint is the Soft-trail jump to doctor open_exceptions.
const SoftAllowDoctorHint = "specular doctor"

// writeSoftAllowBoardHints jumps soft-ALLOW ResourceIDs to approvals show/list
// plus pending/doctor Soft-trail (approve create/close parity).
func writeSoftAllowBoardHints(b *strings.Builder, overrules []ExceptionOverrule, evidenceID, indent string) {
	ids := SoftAllowResourceIDs(overrules)
	for _, id := range ids {
		fmt.Fprintf(b, "%sShow           specular approvals show %s\n", indent, id)
	}
	if len(ids) > 0 {
		fmt.Fprintf(b, "%sList           %s\n", indent, SoftAllowListHint(evidenceID))
		fmt.Fprintf(b, "%sPending        %s\n", indent, SoftAllowPendingHint)
		fmt.Fprintf(b, "%sDoctor         %s\n", indent, SoftAllowDoctorHint)
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

// writeOpenExceptionSoftTrail jumps advisory open exceptions to show/list/
// pending/doctor when Soft Overruled Soft trail was not already printed.
func writeOpenExceptionSoftTrail(b *strings.Builder, exceptions []ApprovalSummary, indent string) {
	seen := make(map[string]struct{}, len(exceptions))
	for _, ex := range exceptions {
		id := strings.TrimSpace(ex.ResourceID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		fmt.Fprintf(b, "%sShow           specular approvals show %s\n", indent, id)
	}
	if len(seen) == 0 {
		return
	}
	fmt.Fprintf(b, "%sList           specular approvals list --status open\n", indent)
	fmt.Fprintf(b, "%sPending        %s\n", indent, SoftAllowPendingHint)
	fmt.Fprintf(b, "%sDoctor         %s\n", indent, SoftAllowDoctorHint)
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
