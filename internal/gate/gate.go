// Package gate implements Specular's primary change-control evaluation.
//
// The gate discovers a proposed change, summarizes provenance, evaluates
// drift (when Specular specs exist), and evaluates policy (when present),
// then returns an ALLOW / DENY verdict. See docs/PRODUCT_INTENT.md.
package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/specular/internal/drift"
	"github.com/felixgeelhaar/specular/internal/eval"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/safeutil"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/ux"
)

// Verdict is the gate decision.
type Verdict string

const (
	// Allow means the change may proceed under current evidence.
	Allow Verdict = "ALLOW"
	// Deny means drift or policy blocked the change.
	Deny Verdict = "DENY"
)

// SectionStatus is PASS, FAIL, or SKIPPED.
type SectionStatus string

const (
	// StatusPass means the section evaluated successfully.
	StatusPass SectionStatus = "PASS"
	// StatusFail means the section blocked the gate.
	StatusFail SectionStatus = "FAIL"
	// StatusSkipped means the section was not applicable (brownfield / missing inputs).
	StatusSkipped SectionStatus = "SKIPPED"
)

// Options configures Evaluate.
type Options struct {
	ProjectRoot string
	PolicyPath  string // empty = default .specular/policy.yaml if present
	ReportFile  string // drift SARIF path (default drift.sarif)
	StrictSpec  bool   // fail when plan/lock/spec missing (default: soft-skip drift)
}

// Result is the machine-readable gate outcome.
type Result struct {
	Verdict    Verdict           `json:"verdict"`
	Reason     string            `json:"reason"`
	Change     ChangeSection     `json:"change"`
	Provenance ProvenanceSection `json:"provenance"`
	Drift      DriftSection      `json:"drift"`
	Policy     PolicySection     `json:"policy"`
}

// ChangeSection summarizes the proposed git change.
type ChangeSection struct {
	Dirty  bool   `json:"dirty"`
	Files  int    `json:"files"`
	Branch string `json:"branch,omitempty"`
	Root   string `json:"root,omitempty"`
}

// ProvenanceSection summarizes known AI / git provenance.
type ProvenanceSection struct {
	Status    SectionStatus `json:"status"` // PASS = attested, SKIPPED = unattested
	Attested  bool          `json:"attested"`
	Sessions  []string      `json:"sessions,omitempty"`
	Harnesses []string      `json:"harnesses,omitempty"`
	GitBranch string        `json:"gitBranch,omitempty"`
	GitDirty  bool          `json:"gitDirty"`
	Note      string        `json:"note,omitempty"`
}

// DriftSection summarizes drift evaluation.
type DriftSection struct {
	Status   SectionStatus   `json:"status"`
	Errors   int             `json:"errors,omitempty"`
	Warnings int             `json:"warnings,omitempty"`
	Info     int             `json:"info,omitempty"`
	SARIF    string          `json:"sarif,omitempty"`
	Note     string          `json:"note,omitempty"`
	Findings []FindingDetail `json:"findings,omitempty"`
}

// FindingDetail is an explainable drift finding for the gate board and JSON.
// Every reported drift should answer: what changed, why it is drift, and where.
type FindingDetail struct {
	Category  string `json:"category"` // plan, code, infra
	Code      string `json:"code"`
	FeatureID string `json:"featureId,omitempty"`
	Message   string `json:"message"`
	Severity  string `json:"severity"` // error, warning, info
	Location  string `json:"location,omitempty"`
}

// PolicySection summarizes policy / verification evaluation.
type PolicySection struct {
	Status  SectionStatus `json:"status"`
	Passed  int           `json:"passed,omitempty"`
	Failed  int           `json:"failed,omitempty"`
	Skipped int           `json:"skipped,omitempty"`
	Note    string        `json:"note,omitempty"`
}

// Evaluate runs the thin gate pipeline for a project root.
func Evaluate(opts Options) (*Result, error) {
	root := opts.ProjectRoot
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("gate: cwd: %w", err)
		}
		root = cwd
	}
	abs, absErr := filepath.Abs(root)
	if absErr != nil {
		return nil, fmt.Errorf("gate: resolve root: %w", absErr)
	}
	root = abs

	reportFile := opts.ReportFile
	if reportFile == "" {
		reportFile = "drift.sarif"
	}

	res := &Result{}
	res.Change = discoverChange(root)
	res.Provenance = discoverProvenance(root)
	res.Drift = evaluateDrift(root, reportFile, opts.StrictSpec)
	res.Policy = evaluatePolicy(root, opts.PolicyPath)
	res.Verdict, res.Reason = decide(res)
	return res, nil
}

func discoverChange(root string) ChangeSection {
	sec := ChangeSection{Root: root}
	out, err := runGit(root, "status", "--porcelain")
	if err == nil {
		lines := nonEmptyLines(out)
		sec.Files = len(lines)
		sec.Dirty = len(lines) > 0
	}
	if branch, bErr := runGit(root, "rev-parse", "--abbrev-ref", "HEAD"); bErr == nil {
		sec.Branch = strings.TrimSpace(branch)
	}
	return sec
}

func discoverProvenance(root string) ProvenanceSection {
	sec := ProvenanceSection{
		Status: StatusSkipped,
		Note:   "unattested — no session attestations found (Level 0–1 provenance)",
	}
	if branch, err := runGit(root, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		sec.GitBranch = strings.TrimSpace(branch)
	}
	if out, err := runGit(root, "status", "--porcelain"); err == nil {
		sec.GitDirty = len(nonEmptyLines(out)) > 0
	}

	dir := filepath.Join(root, ".specular", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return sec
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".attestation.json") {
			continue
		}
		id := strings.TrimSuffix(name, ".attestation.json")
		sec.Sessions = append(sec.Sessions, id)
		data, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			continue
		}
		var payload struct {
			Provenance struct {
				Harness string `json:"harness"`
			} `json:"provenance"`
		}
		if json.Unmarshal(data, &payload) == nil && payload.Provenance.Harness != "" {
			sec.Harnesses = append(sec.Harnesses, payload.Provenance.Harness)
		}
	}
	if len(sec.Sessions) > 0 {
		sec.Attested = true
		sec.Status = StatusPass
		sec.Note = "session attestation(s) present"
	}
	return sec
}

func runGit(dir string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", dir}, args...)
	cmd, err := safeutil.SafeCommand(context.Background(), "git", cmdArgs...)
	if err != nil {
		return "", err
	}
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func evaluateDrift(root, reportFile string, strict bool) DriftSection {
	defaults := ux.NewPathDefaults()
	planFile := filepath.Join(root, defaults.PlanFile())
	lockFile := filepath.Join(root, defaults.SpecLockFile())
	specFile := filepath.Join(root, defaults.SpecFile())
	policyFile := filepath.Join(root, defaults.PolicyFile())
	sarifPath := reportFile
	if !filepath.IsAbs(sarifPath) {
		sarifPath = filepath.Join(root, reportFile)
	}

	missing := missingRequired(planFile, lockFile, specFile)
	if missing != "" {
		if strict {
			return DriftSection{Status: StatusFail, Note: missing}
		}
		return DriftSection{
			Status: StatusSkipped,
			Note:   "no .specular/spec+plan+lock — drift skipped (brownfield); pass --strict-spec to require",
		}
	}

	p, planErr := plan.LoadPlan(planFile)
	if planErr != nil {
		return DriftSection{Status: StatusFail, Note: planErr.Error()}
	}
	lock, lockErr := spec.LoadSpecLock(lockFile)
	if lockErr != nil {
		return DriftSection{Status: StatusFail, Note: lockErr.Error()}
	}
	s, specErr := spec.LoadSpec(specFile)
	if specErr != nil {
		return DriftSection{Status: StatusFail, Note: specErr.Error()}
	}

	planDrift := drift.DetectPlanDrift(lock, p)
	codeDrift := drift.DetectCodeDrift(s, lock, drift.CodeDriftOptions{ProjectRoot: root})
	var infraDrift []drift.Finding
	if _, stErr := os.Stat(policyFile); stErr == nil {
		pol, polErr := policy.LoadPolicy(policyFile)
		if polErr != nil {
			return DriftSection{Status: StatusFail, Note: polErr.Error()}
		}
		infraDrift = drift.DetectInfraDrift(drift.InfraDriftOptions{
			Policy:     pol,
			TaskImages: map[string]string{},
		})
	}
	report := drift.GenerateReport(planDrift, codeDrift, infraDrift)
	_ = drift.SaveSARIF(report.ToSARIF(), sarifPath)

	findings := collectFindings(report)
	sec := DriftSection{
		Errors:   report.Summary.Errors,
		Warnings: report.Summary.Warnings,
		Info:     report.Summary.Info,
		SARIF:    reportFile,
		Findings: findings,
	}
	if report.HasErrors() {
		sec.Status = StatusFail
		sec.Note = explainDriftFailure(findings, report.Summary.Errors)
		return sec
	}
	sec.Status = StatusPass
	if report.IsClean() {
		sec.Note = "no drift detected"
	} else {
		sec.Note = fmt.Sprintf("drift warnings/info present (%d findings)", len(findings))
	}
	return sec
}

func collectFindings(report *drift.Report) []FindingDetail {
	var out []FindingDetail
	out = append(out, mapFindings("plan", report.PlanDrift)...)
	out = append(out, mapFindings("code", report.CodeDrift)...)
	out = append(out, mapFindings("infra", report.InfraDrift)...)
	return out
}

func mapFindings(category string, in []drift.Finding) []FindingDetail {
	if len(in) == 0 {
		return nil
	}
	out := make([]FindingDetail, 0, len(in))
	for _, f := range in {
		out = append(out, FindingDetail{
			Category:  category,
			Code:      f.Code,
			FeatureID: string(f.FeatureID),
			Message:   f.Message,
			Severity:  f.Severity,
			Location:  f.Location,
		})
	}
	return out
}

func explainDriftFailure(findings []FindingDetail, errorCount int) string {
	for _, f := range findings {
		if strings.EqualFold(f.Severity, "error") {
			msg := f.Message
			if msg == "" {
				msg = f.Code
			}
			if f.Location != "" {
				return fmt.Sprintf("%s (%s)", msg, f.Location)
			}
			return msg
		}
	}
	return fmt.Sprintf("drift detection failed with %d errors", errorCount)
}

func missingRequired(paths ...string) string {
	var missing []string
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			missing = append(missing, p)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "missing required files: " + strings.Join(missing, ", ")
}

func evaluatePolicy(root, policyPath string) PolicySection {
	path := strings.TrimSpace(policyPath)
	if path == "" {
		path = filepath.Join(root, ".specular", "policy.yaml")
		if _, err := os.Stat(path); err != nil {
			alt := filepath.Join(root, ".specular", "policies.yaml")
			if _, altErr := os.Stat(alt); altErr == nil {
				path = alt
			} else {
				return PolicySection{
					Status: StatusSkipped,
					Note:   "no .specular/policy.yaml — policy verification skipped",
				}
			}
		}
	}
	pol, err := policy.LoadPolicy(path)
	if err != nil {
		return PolicySection{Status: StatusFail, Note: err.Error()}
	}
	report, gateErr := eval.RunEvalGate(eval.GateOptions{
		Policy:      pol,
		ProjectRoot: root,
	})
	if gateErr != nil {
		return PolicySection{Status: StatusFail, Note: gateErr.Error()}
	}
	sec := PolicySection{
		Passed:  report.TotalPassed,
		Failed:  report.TotalFailed,
		Skipped: report.TotalSkipped,
	}
	if !report.AllPassed {
		sec.Status = StatusFail
		sec.Note = fmt.Sprintf("policy verification failed (%d checks)", report.TotalFailed)
		return sec
	}
	sec.Status = StatusPass
	sec.Note = "policy verification passed"
	return sec
}

func decide(res *Result) (Verdict, string) {
	if res.Drift.Status == StatusFail {
		return Deny, firstNonEmpty(res.Drift.Note, "drift evaluation failed")
	}
	if res.Policy.Status == StatusFail {
		return Deny, firstNonEmpty(res.Policy.Note, "policy evaluation failed")
	}
	parts := []string{}
	if res.Drift.Status == StatusPass {
		parts = append(parts, "drift pass")
	} else {
		parts = append(parts, "drift skipped")
	}
	if res.Policy.Status == StatusPass {
		parts = append(parts, "policy pass")
	} else {
		parts = append(parts, "policy skipped")
	}
	if res.Provenance.Attested {
		parts = append(parts, "provenance attested")
	} else {
		parts = append(parts, "provenance unattested")
	}
	return Allow, strings.Join(parts, "; ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// FormatText renders the human gate board.
func FormatText(res *Result) string {
	var b strings.Builder
	b.WriteString("SPECULAR CHANGE CONTROL\n")
	b.WriteString(strings.Repeat("─", 46) + "\n")
	b.WriteString("Change\n")
	fmt.Fprintf(&b, "  Dirty          %v\n", res.Change.Dirty)
	fmt.Fprintf(&b, "  Uncommitted    %d\n", res.Change.Files)
	if res.Change.Branch != "" {
		fmt.Fprintf(&b, "  Branch         %s\n", res.Change.Branch)
	}
	b.WriteString("Provenance\n")
	fmt.Fprintf(&b, "  Status         %s\n", res.Provenance.Status)
	if len(res.Provenance.Harnesses) > 0 {
		fmt.Fprintf(&b, "  Harnesses      %s\n", strings.Join(unique(res.Provenance.Harnesses), ", "))
	}
	if len(res.Provenance.Sessions) > 0 {
		fmt.Fprintf(&b, "  Sessions       %s\n", strings.Join(res.Provenance.Sessions, ", "))
	}
	if res.Provenance.Note != "" {
		fmt.Fprintf(&b, "  Note           %s\n", res.Provenance.Note)
	}
	b.WriteString("Drift\n")
	fmt.Fprintf(&b, "  Status         %s\n", res.Drift.Status)
	if res.Drift.Status != StatusSkipped {
		fmt.Fprintf(&b, "  Findings       errors=%d warnings=%d info=%d\n",
			res.Drift.Errors, res.Drift.Warnings, res.Drift.Info)
	}
	writeFindingDetails(&b, res.Drift.Findings)
	if res.Drift.SARIF != "" {
		fmt.Fprintf(&b, "  SARIF          %s\n", res.Drift.SARIF)
	}
	if res.Drift.Note != "" {
		fmt.Fprintf(&b, "  Note           %s\n", res.Drift.Note)
	}
	b.WriteString("Policy\n")
	fmt.Fprintf(&b, "  Status         %s\n", res.Policy.Status)
	if res.Policy.Status != StatusSkipped {
		fmt.Fprintf(&b, "  Checks         passed=%d failed=%d skipped=%d\n",
			res.Policy.Passed, res.Policy.Failed, res.Policy.Skipped)
	}
	if res.Policy.Note != "" {
		fmt.Fprintf(&b, "  Note           %s\n", res.Policy.Note)
	}
	b.WriteString(strings.Repeat("─", 46) + "\n")
	fmt.Fprintf(&b, "VERDICT: %s\n", res.Verdict)
	fmt.Fprintf(&b, "REASON:  %s\n", res.Reason)
	return b.String()
}

const maxPrintedFindings = 8

func writeFindingDetails(b *strings.Builder, findings []FindingDetail) {
	if len(findings) == 0 {
		return
	}
	b.WriteString("  Detail\n")
	limit := len(findings)
	if limit > maxPrintedFindings {
		limit = maxPrintedFindings
	}
	for i := 0; i < limit; i++ {
		f := findings[i]
		sev := f.Severity
		if sev == "" {
			sev = "info"
		}
		line := fmt.Sprintf("    [%s] %s", sev, f.Code)
		if f.Category != "" {
			line += " (" + f.Category + ")"
		}
		if f.FeatureID != "" {
			line += " feature=" + f.FeatureID
		}
		b.WriteString(line + "\n")
		if f.Message != "" {
			fmt.Fprintf(b, "      %s\n", f.Message)
		}
		if f.Location != "" {
			fmt.Fprintf(b, "      at %s\n", f.Location)
		}
	}
	if len(findings) > maxPrintedFindings {
		fmt.Fprintf(b, "    … and %d more (see SARIF / --json)\n", len(findings)-maxPrintedFindings)
	}
}

func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
