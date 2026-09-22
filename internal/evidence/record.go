// Package evidence implements Specular's Change Evidence Graph store (v1).
//
// A Record captures a gate evaluation as a durable, content-addressable
// evidence artifact under .specular/evidence/. Later graph edges (intent →
// session → commit → approval) will link to these records. See
// docs/PRODUCT_INTENT.md §7 and §20.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/gate"
	"github.com/felixgeelhaar/specular/internal/provenance"
)

// Schema identifies the evidence record format.
const Schema = "specular.evidence/v1"

// DirName is the directory under .specular for evidence records.
const DirName = "evidence"

// LatestName is the pointer file holding the most recent evidence ID.
const LatestName = "latest"

// Record is one node in the Change Evidence Graph: a gate decision plus
// repository coordinates. Relationships (intent, session, approval) land in
// later schema versions.
type Record struct {
	Schema    string       `json:"schema"`
	ID        string       `json:"id"`
	CreatedAt time.Time    `json:"createdAt"`
	Root      string       `json:"root,omitempty"`
	Branch    string       `json:"branch,omitempty"`
	Commit    string       `json:"commit,omitempty"`
	Gate      *gate.Result `json:"gate"`
}

// NewFromGate builds a Record from a gate evaluation result.
func NewFromGate(root string, res *gate.Result) (*Record, error) {
	if res == nil {
		return nil, fmt.Errorf("evidence: nil gate result")
	}
	rec := &Record{
		Schema:    Schema,
		CreatedAt: time.Now().UTC(),
		Root:      root,
		Gate:      res,
	}
	if res.Change.Branch != "" {
		rec.Branch = res.Change.Branch
	}
	id, err := contentID(rec)
	if err != nil {
		return nil, err
	}
	rec.ID = id
	return rec, nil
}

func contentID(rec *Record) (string, error) {
	// Hash stable fields only (exclude CreatedAt jitter by hashing gate payload).
	payload := struct {
		Schema string       `json:"schema"`
		Root   string       `json:"root,omitempty"`
		Branch string       `json:"branch,omitempty"`
		Commit string       `json:"commit,omitempty"`
		Gate   *gate.Result `json:"gate"`
	}{
		Schema: rec.Schema,
		Root:   rec.Root,
		Branch: rec.Branch,
		Commit: rec.Commit,
		Gate:   rec.Gate,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("evidence: marshal for id: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "ev_" + hex.EncodeToString(sum[:12]), nil
}

// Dir returns .specular/evidence under root.
func Dir(root string) string {
	return filepath.Join(root, ".specular", DirName)
}

// Write persists the record and updates the latest pointer.
func Write(root string, rec *Record) error {
	if rec == nil || rec.ID == "" {
		return fmt.Errorf("evidence: missing record id")
	}
	dir := Dir(root)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("evidence: mkdir: %w", err)
	}
	path := filepath.Join(dir, rec.ID+".json")
	raw, marshalErr := json.MarshalIndent(rec, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("evidence: marshal: %w", marshalErr)
	}
	raw = append(raw, '\n')
	if writeErr := os.WriteFile(path, raw, 0o600); writeErr != nil {
		return fmt.Errorf("evidence: write: %w", writeErr)
	}
	latest := filepath.Join(dir, LatestName)
	if latestErr := os.WriteFile(latest, []byte(rec.ID+"\n"), 0o600); latestErr != nil {
		return fmt.Errorf("evidence: latest: %w", latestErr)
	}
	return nil
}

// Load reads a record by ID.
func Load(root, id string) (*Record, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("evidence: empty id")
	}
	path := filepath.Join(Dir(root), id+".json")
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("evidence: load %s: %w", id, readErr)
	}
	var rec Record
	if parseErr := json.Unmarshal(raw, &rec); parseErr != nil {
		return nil, fmt.Errorf("evidence: parse %s: %w", id, parseErr)
	}
	return &rec, nil
}

// LoadLatest reads the record referenced by the latest pointer.
func LoadLatest(root string) (*Record, error) {
	raw, err := os.ReadFile(filepath.Join(Dir(root), LatestName))
	if err != nil {
		return nil, fmt.Errorf("evidence: no latest record (run specular gate first): %w", err)
	}
	return Load(root, strings.TrimSpace(string(raw)))
}

// ListIDs returns evidence IDs newest-first by CreatedAt (ID tiebreak).
func ListIDs(root string) ([]string, error) {
	recs, err := List(root, ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]string, len(recs))
	for i, rec := range recs {
		out[i] = rec.ID
	}
	return out, nil
}

const changeRecordRule = "──────────────────────────────────────"

// FormatExplain renders an auditor-facing AI CHANGE RECORD (PRODUCT_INTENT §19).
// Machine --json output is unchanged; this is the human text layout only.
func FormatExplain(rec *Record) string {
	if rec == nil || rec.Gate == nil {
		return "No evidence record.\n"
	}
	g := rec.Gate
	var b strings.Builder

	b.WriteString("AI CHANGE RECORD\n")
	b.WriteString(changeRecordRule + "\n")

	fmt.Fprintf(&b, "Decision     %s\n", g.Verdict)
	if g.Reason != "" {
		fmt.Fprintf(&b, "Reason       %s\n", g.Reason)
	}
	if !rec.CreatedAt.IsZero() {
		fmt.Fprintf(&b, "Timestamp    %s\n", rec.CreatedAt.UTC().Format(time.RFC3339))
	}
	if rec.ID != "" {
		fmt.Fprintf(&b, "Evidence     %s\n", rec.ID)
	}

	writeChangeSummary(&b, rec)
	writeProvenanceBlock(&b, g)
	writePolicyBlock(&b, g)
	writeDriftBlock(&b, g)
	writeRiskBlock(&b, g)
	writeApprovalsBlock(&b, g)

	b.WriteString("Why\n")
	writeWhy(&b, g)

	b.WriteString(changeRecordRule + "\n")
	b.WriteString("Refs\n")
	fmt.Fprintf(&b, "Evidence     %s\n", evidenceRef(rec))
	if g.Drift.SARIF != "" {
		fmt.Fprintf(&b, "SARIF        %s\n", g.Drift.SARIF)
	}
	b.WriteString("Store        .specular/evidence/\n")
	b.WriteString("Re-run       specular gate\n")
	return b.String()
}

func writeChangeSummary(b *strings.Builder, rec *Record) {
	g := rec.Gate
	b.WriteString("Change\n")
	branch := rec.Branch
	if branch == "" {
		branch = g.Change.Branch
	}
	if branch != "" {
		fmt.Fprintf(b, "Branch       %s\n", branch)
	}
	if rec.Commit != "" {
		fmt.Fprintf(b, "Commit       %s\n", rec.Commit)
	}
	repo := repositoryLabel(rec)
	if repo != "" {
		fmt.Fprintf(b, "Repository   %s\n", repo)
	}
	switch {
	case g.Change.Dirty:
		fmt.Fprintf(b, "Files        %d uncommitted\n", g.Change.Files)
	case g.Change.Files > 0:
		fmt.Fprintf(b, "Files        %d\n", g.Change.Files)
	default:
		b.WriteString("Files        clean working tree\n")
	}
}

func repositoryLabel(rec *Record) string {
	root := rec.Root
	if root == "" && rec.Gate != nil {
		root = rec.Gate.Change.Root
	}
	if root == "" {
		return ""
	}
	base := filepath.Base(root)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return root
	}
	return base
}

func writeProvenanceBlock(b *strings.Builder, g *gate.Result) {
	b.WriteString("Provenance\n")
	if g.Provenance.Attested {
		b.WriteString("Status       ATTESTED\n")
	} else {
		b.WriteString("Status       UNATTESTED\n")
	}
	if len(g.Provenance.Harnesses) > 0 {
		fmt.Fprintf(b, "Harness      %s\n", strings.Join(g.Provenance.Harnesses, ", "))
	}
	if len(g.Provenance.Sessions) > 0 {
		fmt.Fprintf(b, "Session      %s\n", strings.Join(g.Provenance.Sessions, ", "))
	}
	if len(g.Provenance.WorktreePaths) > 0 {
		fmt.Fprintf(b, "Worktree     %s\n", strings.Join(g.Provenance.WorktreePaths, ", "))
	}
	if len(g.Provenance.WorktreeBranches) > 0 {
		fmt.Fprintf(b, "WtBranch     %s\n", strings.Join(g.Provenance.WorktreeBranches, ", "))
	}
	if len(g.Provenance.WorktreeNames) > 0 {
		fmt.Fprintf(b, "WtName       %s\n", strings.Join(g.Provenance.WorktreeNames, ", "))
	}
	if g.Provenance.Attested {
		fmt.Fprintf(b, "Governed     %v\n", g.Provenance.Governed)
	}
	if g.Provenance.ProtocolDocs > 0 {
		schema := g.Provenance.ProtocolSchema
		if schema == "" {
			schema = "specular.provenance/v1"
		}
		fmt.Fprintf(b, "Protocol     %s (%s)\n", schema,
			provenance.FormatProtocolDocsOK(g.Provenance.ProtocolOK, g.Provenance.ProtocolDocs))
	}
	if !g.Provenance.Attested {
		note := g.Provenance.Note
		if note == "" {
			note = "not treated as verified"
		}
		fmt.Fprintf(b, "Note         %s\n", note)
	} else if g.Provenance.Note != "" {
		fmt.Fprintf(b, "Note         %s\n", g.Provenance.Note)
	}
}

func writePolicyBlock(b *strings.Builder, g *gate.Result) {
	b.WriteString("Policy\n")
	fmt.Fprintf(b, "Status       %s\n", sectionMark(g.Policy.Status))
	if g.Policy.Status != gate.StatusSkipped {
		fmt.Fprintf(b, "Checks       passed=%d failed=%d skipped=%d\n",
			g.Policy.Passed, g.Policy.Failed, g.Policy.Skipped)
	}
	if g.Policy.Note != "" {
		fmt.Fprintf(b, "Note         %s\n", g.Policy.Note)
	}
}

func writeDriftBlock(b *strings.Builder, g *gate.Result) {
	b.WriteString("Drift\n")
	fmt.Fprintf(b, "Status       %s\n", sectionMark(g.Drift.Status))
	switch {
	case g.Drift.Status == gate.StatusPass && len(g.Drift.Findings) == 0:
		b.WriteString("Summary      None detected\n")
	case g.Drift.Status == gate.StatusSkipped:
		if g.Drift.Note != "" {
			fmt.Fprintf(b, "Summary      %s\n", g.Drift.Note)
		} else {
			b.WriteString("Summary      Skipped (brownfield / missing spec)\n")
		}
	default:
		fmt.Fprintf(b, "Summary      errors=%d warnings=%d info=%d\n",
			g.Drift.Errors, g.Drift.Warnings, g.Drift.Info)
		if g.Drift.Note != "" {
			fmt.Fprintf(b, "Note         %s\n", g.Drift.Note)
		}
	}
	writeDriftFindings(b, g.Drift.Findings)
}

func writeDriftFindings(b *strings.Builder, findings []gate.FindingDetail) {
	if len(findings) == 0 {
		return
	}
	limit := len(findings)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		f := findings[i]
		mark := findingMark(f.Severity)
		fmt.Fprintf(b, "%s %-12s %s", mark, f.Code, f.Severity)
		if f.Category != "" {
			fmt.Fprintf(b, " (%s)", f.Category)
		}
		b.WriteString("\n")
		if f.Message != "" {
			fmt.Fprintf(b, "  %s\n", f.Message)
		}
		if f.Location != "" {
			fmt.Fprintf(b, "  at %s\n", f.Location)
		}
	}
	if len(findings) > 8 {
		fmt.Fprintf(b, "  … and %d more (see SARIF / --json)\n", len(findings)-8)
	}
}

func writeRiskBlock(b *strings.Builder, g *gate.Result) {
	b.WriteString("Risk\n")
	level := strings.TrimSpace(g.Risk.Level)
	if level == "" {
		level = "NONE"
	}
	mode := "advisory"
	if g.Risk.Enforced {
		mode = "enforced"
	}
	fmt.Fprintf(b, "Level        %s (%s)\n", level, mode)
	for _, f := range g.Risk.Factors {
		fmt.Fprintf(b, "  + %s\n", f)
	}
	if len(g.Risk.Required) > 0 {
		fmt.Fprintf(b, "Required     %s\n", strings.Join(g.Risk.Required, ", "))
	}
	if len(g.Risk.Observed) > 0 {
		fmt.Fprintf(b, "Observed     %s\n", strings.Join(g.Risk.Observed, ", "))
	}
	if len(g.Risk.Missing) > 0 {
		fmt.Fprintf(b, "Missing      %s\n", strings.Join(g.Risk.Missing, ", "))
	}
	if g.Risk.Note != "" {
		fmt.Fprintf(b, "Note         %s\n", g.Risk.Note)
	}
}

func writeApprovalsBlock(b *strings.Builder, g *gate.Result) {
	b.WriteString("Approvals\n")
	sec := g.Approvals
	if len(sec.Overrules) > 0 {
		b.WriteString("Overruled\n")
		for _, o := range sec.Overrules {
			fmt.Fprintf(b, "⚠ soft-ALLOW %-6s %s (%s)\n", o.Kind, o.ResourceID, o.Binding)
		}
	}
	if sec.Count == 0 {
		b.WriteString("Status       none recorded\n")
		if g.Verdict == gate.Deny {
			b.WriteString("Hint         specular approve exception-<id> --reason \"...\" --scope \"...\" --policy …\n")
		}
		return
	}
	fmt.Fprintf(b, "Records      %d\n", sec.Count)
	writeApprovalExceptions(b, sec.Exceptions)
	writeApprovalRecent(b, sec.Recent)
	if g.Verdict == gate.Deny && len(sec.Exceptions) == 0 {
		b.WriteString("Hint         record an exception: specular approve exception-<id> --reason \"...\" --scope \"...\" --policy …\n")
	}
	if sec.Note != "" {
		fmt.Fprintf(b, "Note         %s\n", sec.Note)
	}
}

func writeApprovalExceptions(b *strings.Builder, exceptions []gate.ApprovalSummary) {
	for _, ex := range exceptions {
		fmt.Fprintf(b, "⚠ %-12s %s", "exception", ex.ResourceID)
		if ex.Reason != "" {
			fmt.Fprintf(b, " — %s", ex.Reason)
		}
		b.WriteString("\n")
		if ex.Scope != "" {
			fmt.Fprintf(b, "  scope=%s\n", ex.Scope)
		}
		if ex.Policy != "" {
			fmt.Fprintf(b, "  policy=%s\n", ex.Policy)
		}
		if ex.ApprovedBy != "" {
			fmt.Fprintf(b, "  by %s\n", ex.ApprovedBy)
		}
		if ex.ExpiresAt != "" {
			fmt.Fprintf(b, "  expires %s\n", ex.ExpiresAt)
		}
	}
}

func writeApprovalRecent(b *strings.Builder, recent []gate.ApprovalSummary) {
	shown := 0
	for _, r := range recent {
		if r.Type == "exception" && !r.Expired {
			continue // already listed above
		}
		mark := "✓"
		if r.Type == "exception" {
			mark = "·"
		}
		fmt.Fprintf(b, "%s %-12s %s", mark, r.Type, r.ResourceID)
		if r.ApprovedBy != "" {
			fmt.Fprintf(b, " by %s", r.ApprovedBy)
		}
		b.WriteString("\n")
		shown++
		if shown >= 5 {
			break
		}
	}
}

func sectionMark(status gate.SectionStatus) string {
	switch status {
	case gate.StatusPass:
		return "✓ PASS"
	case gate.StatusFail:
		return "✗ FAIL"
	case gate.StatusSkipped:
		return "· SKIPPED"
	default:
		return string(status)
	}
}

func findingMark(severity string) string {
	switch strings.ToLower(severity) {
	case "error":
		return "✗"
	case "warning":
		return "⚠"
	default:
		return "·"
	}
}

func evidenceRef(rec *Record) string {
	if rec.ID == "" {
		return ".specular/evidence/"
	}
	return ".specular/evidence/" + rec.ID + ".json"
}

func writeWhy(b *strings.Builder, g *gate.Result) {
	writeSectionWhy(b, "Drift", g.Drift.Status, g.Drift.Note, map[gate.SectionStatus]string{
		gate.StatusSkipped: "Drift skipped (brownfield / missing spec)",
	})
	writeSectionWhy(b, "Policy", g.Policy.Status, g.Policy.Note, nil)
	writeWhyProvenance(b, g)
	writeWhyApprovals(b, g)
	writeWhyRisk(b, g)
	writeWhyVerdict(b, g)
}

func writeWhyProvenance(b *strings.Builder, g *gate.Result) {
	if !g.Provenance.Attested {
		writeWhyUnattested(b, g)
		return
	}
	b.WriteString("  • Provenance attested")
	if len(g.Provenance.Harnesses) > 0 {
		fmt.Fprintf(b, " (%s)", strings.Join(g.Provenance.Harnesses, ", "))
	}
	b.WriteString("\n")
	writeWhyWorktree(b, g)
	fmt.Fprintf(b, "  • Governed %v\n", g.Provenance.Governed)
	writeWhyProvenanceEnforce(b, g)
}

func writeWhyUnattested(b *strings.Builder, g *gate.Result) {
	if g.Provenance.Enforced && g.Provenance.Status == gate.StatusFail {
		note := g.Provenance.Note
		if note == "" {
			note = "attested provenance required"
		}
		fmt.Fprintf(b, "  • Provenance unattested — enforce FAIL (%s)\n", note)
		return
	}
	b.WriteString("  • Provenance unattested — not treated as verified\n")
}

func writeWhyWorktree(b *strings.Builder, g *gate.Result) {
	if len(g.Provenance.WorktreePaths) == 0 && len(g.Provenance.WorktreeBranches) == 0 {
		return
	}
	b.WriteString("  • Worktree ")
	parts := make([]string, 0, 2)
	if len(g.Provenance.WorktreePaths) > 0 {
		parts = append(parts, strings.Join(g.Provenance.WorktreePaths, ", "))
	}
	if len(g.Provenance.WorktreeBranches) > 0 {
		parts = append(parts, "branch="+strings.Join(g.Provenance.WorktreeBranches, ", "))
	}
	b.WriteString(strings.Join(parts, " · "))
	b.WriteString("\n")
}

func writeWhyProvenanceEnforce(b *strings.Builder, g *gate.Result) {
	if g.Provenance.Enforced && g.Provenance.Status == gate.StatusFail {
		note := g.Provenance.Note
		if note == "" {
			note = "provenance verification failed"
		}
		label := "Provenance enforce FAIL"
		switch {
		case strings.Contains(note, "APP protocol"):
			label = "APP protocol enforce FAIL"
		case strings.Contains(note, "governed provenance"):
			label = "Governed provenance enforce FAIL"
		case strings.Contains(note, "attested provenance"):
			label = "Attested provenance enforce FAIL"
		}
		fmt.Fprintf(b, "  • %s — %s\n", label, note)
		return
	}
	if g.Provenance.ProtocolDocs > 0 {
		mode := "advisory"
		if g.Provenance.Enforced {
			mode = "enforced"
		}
		fmt.Fprintf(b, "  • APP protocol %s: %s\n",
			mode, provenance.FormatProtocolDocsOK(g.Provenance.ProtocolOK, g.Provenance.ProtocolDocs))
		return
	}
	if g.Provenance.Enforced {
		switch {
		case strings.Contains(g.Provenance.Note, "governed provenance"):
			b.WriteString("  • Governed provenance enforce active\n")
		case strings.Contains(g.Provenance.Note, "attested provenance"):
			b.WriteString("  • Attested provenance enforce active\n")
		default:
			b.WriteString("  • APP protocol enforce active\n")
		}
	}
}

func writeWhyApprovals(b *strings.Builder, g *gate.Result) {
	switch {
	case len(g.Approvals.Overrules) > 0:
		fmt.Fprintf(b, "  • %d exception soft-ALLOW overrule(s)\n", len(g.Approvals.Overrules))
		for _, o := range g.Approvals.Overrules {
			fmt.Fprintf(b, "    – %s overruled %s (%s)\n", o.ResourceID, o.Kind, o.Binding)
		}
	case len(g.Approvals.Exceptions) > 0:
		fmt.Fprintf(b, "  • %d open exception(s) on local trail\n", len(g.Approvals.Exceptions))
	case g.Approvals.Count > 0:
		fmt.Fprintf(b, "  • %d approval record(s) on local trail\n", g.Approvals.Count)
	case g.Verdict == gate.Deny:
		b.WriteString("  • No local exception/approval trail for this DENY\n")
	}
}

func writeWhyRisk(b *strings.Builder, g *gate.Result) {
	if g.Risk.Enforced && len(g.Risk.Missing) > 0 && len(g.Approvals.Overrules) == 0 {
		fmt.Fprintf(b, "  • Risk %s missing approvals: %s\n",
			firstNonEmpty(g.Risk.Level, "UNKNOWN"), strings.Join(g.Risk.Missing, ", "))
		return
	}
	if g.Risk.Level == "" || g.Risk.Level == "NONE" {
		return
	}
	fmt.Fprintf(b, "  • Risk level %s", g.Risk.Level)
	if g.Risk.Enforced {
		b.WriteString(" (enforced)")
	} else {
		b.WriteString(" (advisory)")
	}
	b.WriteString("\n")
}

func writeWhyVerdict(b *strings.Builder, g *gate.Result) {
	switch {
	case g.Verdict == gate.Allow && len(g.Approvals.Overrules) > 0:
		b.WriteString("  → ALLOW via scoped exception soft-ALLOW (underlying FAIL sections preserved).\n")
	case g.Verdict == gate.Allow:
		b.WriteString("  → ALLOW because no blocking drift, policy, risk, or provenance requirement failed.\n")
	default:
		b.WriteString("  → DENY because a blocking section failed (see above).\n")
	}
}

func writeSectionWhy(b *strings.Builder, name string, status gate.SectionStatus, note string, labels map[gate.SectionStatus]string) {
	label := labels[status]
	if label == "" {
		switch status {
		case gate.StatusPass:
			label = name + " passed"
		case gate.StatusFail:
			label = name + " failed"
		case gate.StatusSkipped:
			label = name + " skipped"
		default:
			label = name + " " + string(status)
		}
	}
	b.WriteString("  • " + label)
	if note != "" {
		fmt.Fprintf(b, " — %s", note)
	}
	b.WriteString("\n")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
