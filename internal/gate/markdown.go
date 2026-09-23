package gate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/felixgeelhaar/specular/internal/provenance"
)

// MarkdownMarker is the stable heading used to upsert PR comments.
const MarkdownMarker = "## Specular Change Control"

// FormatMarkdownOptions configures FormatMarkdownWith.
type FormatMarkdownOptions struct {
	EvidenceID  string
	MaxFindings int // default 8
}

// FormatMarkdown renders a PR / GITHUB_STEP_SUMMARY body for a gate result.
func FormatMarkdown(res *Result) string {
	return FormatMarkdownWith(res, FormatMarkdownOptions{})
}

// FormatMarkdownWith renders markdown with optional evidence id and finding cap.
func FormatMarkdownWith(res *Result, opts FormatMarkdownOptions) string {
	if res == nil {
		return MarkdownMarker + "\n\n_No gate result._\n"
	}
	maxFindings := opts.MaxFindings
	if maxFindings <= 0 {
		maxFindings = 8
	}

	var b strings.Builder
	b.WriteString(MarkdownMarker + "\n\n")
	writeMarkdownVerdict(&b, res)
	writeMarkdownTable(&b, res)
	writeMarkdownNotes(&b, res, opts.EvidenceID)
	writeMarkdownFindings(&b, res.Drift.Findings, maxFindings)
	writeMarkdownDenyNextSteps(&b, res)
	if opts.EvidenceID != "" {
		fmt.Fprintf(&b, "**Evidence:** `%s` — `specular explain %s`\n\n", opts.EvidenceID, opts.EvidenceID)
	} else {
		b.WriteString("Re-run locally: `specular gate` · explain: `specular explain`\n")
	}
	return b.String()
}

func writeMarkdownVerdict(b *strings.Builder, res *Result) {
	verdictIcon := "✅"
	statusWord := "PASSED"
	if res.Verdict == Deny {
		verdictIcon = "❌"
		statusWord = "DENIED"
	}
	fmt.Fprintf(b, "**%s %s** — `%s`\n\n", verdictIcon, statusWord, res.Verdict)
	if res.Reason != "" {
		fmt.Fprintf(b, "**Reason:** %s\n\n", res.Reason)
	}
}

func writeMarkdownTable(b *strings.Builder, res *Result) {
	b.WriteString("| Section | Status |\n")
	b.WriteString("|---|---|\n")
	fmt.Fprintf(b, "| Change | dirty=`%v` files=`%d` branch=`%s` |\n",
		res.Change.Dirty, res.Change.Files, res.Change.Branch)
	fmt.Fprintf(b, "| Provenance | `%s` |\n", res.Provenance.Status)
	fmt.Fprintf(b, "| Drift | `%s` |\n", res.Drift.Status)
	fmt.Fprintf(b, "| Policy | `%s` |\n", res.Policy.Status)
	riskLevel := res.Risk.Level
	if riskLevel == "" {
		riskLevel = "NONE"
	}
	riskMode := "advisory"
	if res.Risk.Enforced {
		riskMode = "enforced"
	}
	fmt.Fprintf(b, "| Risk | `%s` (%s) |\n", riskLevel, riskMode)
	approvalStatus := "none"
	if res.Approvals.Count > 0 {
		approvalStatus = fmt.Sprintf("%d recorded", res.Approvals.Count)
		if n := len(res.Approvals.Exceptions); n > 0 {
			approvalStatus += fmt.Sprintf(", %d open exception(s)", n)
		}
	}
	approvalMode := "advisory"
	if res.Risk.Enforced {
		approvalMode = "risk-adaptive"
	}
	fmt.Fprintf(b, "| Approvals | `%s` (%s) |\n\n", approvalStatus, approvalMode)
}

func writeMarkdownNotes(b *strings.Builder, res *Result, evidenceID string) {
	if res.Provenance.Note != "" {
		fmt.Fprintf(b, "_Provenance:_ %s\n\n", res.Provenance.Note)
	}
	writeMarkdownSessionProvenance(b, res)
	if res.Drift.Note != "" {
		fmt.Fprintf(b, "_Drift:_ %s\n\n", res.Drift.Note)
	}
	if res.Policy.Note != "" {
		fmt.Fprintf(b, "_Policy:_ %s\n\n", res.Policy.Note)
	}
	writeMarkdownRiskNotes(b, res)
	writeMarkdownApprovals(b, res, evidenceID)
}

func writeMarkdownSessionProvenance(b *strings.Builder, res *Result) {
	if !res.Provenance.Attested {
		return
	}
	var bits []string
	if len(res.Provenance.WorktreePaths) > 0 {
		bits = append(bits, "worktree=`"+strings.Join(res.Provenance.WorktreePaths, ", ")+"`")
	}
	if len(res.Provenance.WorktreeBranches) > 0 {
		bits = append(bits, "branch=`"+strings.Join(res.Provenance.WorktreeBranches, ", ")+"`")
	}
	bits = append(bits, fmt.Sprintf("governed=`%v`", res.Provenance.Governed))
	if res.Provenance.ProtocolDocs > 0 {
		bits = append(bits, fmt.Sprintf("APP %s",
			provenance.FormatProtocolDocsOK(res.Provenance.ProtocolOK, res.Provenance.ProtocolDocs)))
	} else if res.ProvenanceProtocol != nil {
		bits = append(bits, "protocol=`"+res.ProvenanceProtocol.Schema+"`")
	}
	fmt.Fprintf(b, "_Session provenance:_ %s\n\n", strings.Join(bits, " · "))
}

func writeMarkdownRiskNotes(b *strings.Builder, res *Result) {
	if len(res.Risk.Factors) > 0 {
		label := "_Risk factors (advisory):_"
		if res.Risk.Enforced {
			label = "_Risk factors (enforced):_"
		}
		b.WriteString(label + "\n")
		for _, f := range res.Risk.Factors {
			fmt.Fprintf(b, "- %s\n", f)
		}
		b.WriteString("\n")
	} else if res.Risk.Note != "" {
		fmt.Fprintf(b, "_Risk:_ %s\n\n", res.Risk.Note)
	}
	if !res.Risk.Enforced || len(res.Risk.Required) == 0 {
		return
	}
	fmt.Fprintf(b, "_Risk required:_ `%s`", strings.Join(res.Risk.Required, "`, `"))
	if len(res.Risk.Missing) > 0 {
		fmt.Fprintf(b, " — **missing** `%s`", strings.Join(res.Risk.Missing, "`, `"))
	} else if len(res.Risk.Observed) > 0 {
		fmt.Fprintf(b, " — observed `%s`", strings.Join(res.Risk.Observed, "`, `"))
	}
	b.WriteString("\n\n")
}

func writeMarkdownApprovals(b *strings.Builder, res *Result, evidenceID string) {
	if len(res.Approvals.Overrules) > 0 {
		b.WriteString("_Exception soft-ALLOW overruled:_\n")
		for _, o := range res.Approvals.Overrules {
			fmt.Fprintf(b, "- ⚠ `%s` → `%s` (%s)\n", o.ResourceID, o.Kind, o.Binding)
		}
		for _, id := range SoftAllowResourceIDs(res.Approvals.Overrules) {
			fmt.Fprintf(b, "- Show: `specular approvals show %s`\n", id)
		}
		fmt.Fprintf(b, "- List: `%s`\n", SoftAllowListHint(evidenceID))
		b.WriteString("\n")
	}
	if len(res.Approvals.Exceptions) > 0 {
		label := "_Open exceptions:_"
		if len(res.Approvals.Overrules) == 0 {
			label = "_Open exceptions (advisory until bound):_"
		}
		b.WriteString(label + "\n")
		for _, ex := range res.Approvals.Exceptions {
			line := "- ⚠ `" + ex.ResourceID + "`"
			if ex.Reason != "" {
				line += " — " + ex.Reason
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
		return
	}
	if res.Verdict == Deny {
		fmt.Fprintf(b, "_Approvals:_ none open — record with `specular approve exception-<id> --reason \"...\" --scope \"...\" --policy %s`\n\n",
			SoftAllowPolicyHint(res))
		return
	}
	if res.Approvals.Note != "" && res.Approvals.Count == 0 {
		fmt.Fprintf(b, "_Approvals:_ %s\n\n", res.Approvals.Note)
	}
}

func writeMarkdownFindings(b *strings.Builder, findings []FindingDetail, maxFindings int) {
	if len(findings) == 0 {
		return
	}
	b.WriteString("### Drift findings\n\n")
	limit := len(findings)
	if limit > maxFindings {
		limit = maxFindings
	}
	for i := 0; i < limit; i++ {
		f := findings[i]
		fmt.Fprintf(b, "- **[%s] %s**", f.Severity, f.Code)
		if f.Category != "" {
			fmt.Fprintf(b, " (%s)", f.Category)
		}
		if f.Message != "" {
			fmt.Fprintf(b, ": %s", f.Message)
		}
		loc := f.Location
		if f.Path != "" {
			loc = f.Path
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.Path, f.Line)
			}
		}
		if loc != "" {
			fmt.Fprintf(b, " _at `%s`_", loc)
		}
		b.WriteString("\n")
	}
	if len(findings) > maxFindings {
		fmt.Fprintf(b, "\n_…and %d more (see SARIF / `specular explain`)._\n",
			len(findings)-maxFindings)
	}
	b.WriteString("\n")
}

// Annotation is one GitHub Actions workflow command annotation.
type Annotation struct {
	Level   string // error | warning | notice
	Path    string
	Line    int
	Title   string
	Message string
}

// AnnotationsFromResult maps drift findings (and deny reason fallback) to annotations.
func AnnotationsFromResult(res *Result) []Annotation {
	if res == nil {
		return nil
	}
	var out []Annotation
	for _, f := range res.Drift.Findings {
		level := severityToLevel(f.Severity)
		path := f.Path
		line := f.Line
		if path == "" {
			path, line, _ = ParseFindingLocation(f.Location)
		}
		msg := f.Message
		if msg == "" {
			msg = f.Code
		}
		out = append(out, Annotation{
			Level:   level,
			Path:    path,
			Line:    line,
			Title:   f.Code,
			Message: msg,
		})
	}
	if res.Verdict == Deny && len(out) == 0 && res.Reason != "" {
		out = append(out, Annotation{
			Level:   "error",
			Title:   "gate",
			Message: res.Reason,
		})
	}
	return out
}

func severityToLevel(sev string) string {
	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "error":
		return "error"
	case "warning":
		return "warning"
	default:
		return "notice"
	}
}

// FormatGitHubAnnotations emits ::error/::warning/::notice workflow commands.
func FormatGitHubAnnotations(anns []Annotation) string {
	if len(anns) == 0 {
		return ""
	}
	var b strings.Builder
	for _, a := range anns {
		level := a.Level
		if level == "" {
			level = "notice"
		}
		props := []string{}
		if a.Path != "" {
			props = append(props, "file="+escapeProp(a.Path))
		}
		if a.Line > 0 {
			props = append(props, "line="+strconv.Itoa(a.Line))
		}
		if a.Title != "" {
			props = append(props, "title="+escapeProp(a.Title))
		}
		msg := escapeMsg(a.Message)
		if len(props) == 0 {
			fmt.Fprintf(&b, "::%s::%s\n", level, msg)
			continue
		}
		fmt.Fprintf(&b, "::%s %s::%s\n", level, strings.Join(props, ","), msg)
	}
	return b.String()
}

func escapeProp(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, ",", "%2C")
	return s
}

func escapeMsg(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	return s
}

// ParseFindingLocation best-effort parses "path", "path:line", or "path:line:col".
func ParseFindingLocation(loc string) (path string, line int, ok bool) {
	loc = strings.TrimSpace(loc)
	if loc == "" || strings.Contains(loc, "://") {
		return "", 0, false
	}
	if i := strings.LastIndex(loc, ":"); i > 0 {
		if n, err := strconv.Atoi(loc[i+1:]); err == nil {
			rest := loc[:i]
			if j := strings.LastIndex(rest, ":"); j > 0 {
				if lineNum, lineErr := strconv.Atoi(rest[j+1:]); lineErr == nil && looksLikePath(rest[:j]) {
					return rest[:j], lineNum, true
				}
			}
			if looksLikePath(rest) {
				return rest, n, true
			}
		}
	}
	if looksLikePath(loc) {
		return loc, 0, true
	}
	return "", 0, false
}

func looksLikePath(s string) bool {
	if s == "" || strings.ContainsAny(s, " \t") {
		return false
	}
	// Require a path separator or a file extension-ish suffix.
	if strings.Contains(s, "/") || strings.Contains(s, `\`) {
		return true
	}
	if i := strings.LastIndex(s, "."); i > 0 && i < len(s)-1 {
		ext := s[i+1:]
		if len(ext) <= 8 {
			return true
		}
	}
	return false
}
