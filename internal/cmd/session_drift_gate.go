package cmd

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/specular/internal/drift"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/ux"
)

// sessionDriftGateOptions configures the thin outer-loop gate used by
// `session wait --gate` (no checkpoint/spinner UX from eval drift).
type sessionDriftGateOptions struct {
	ProjectRoot string
	ReportFile  string
	Quiet       bool
}

// runSessionDriftGate runs plan/code/(optional infra) drift with fail-on-drift
// semantics. Returns an error whose message maps to exitcode.DriftDetected (4).
func runSessionDriftGate(opts sessionDriftGateOptions) error {
	defaults := ux.NewPathDefaults()
	planFile := defaults.PlanFile()
	lockFile := defaults.SpecLockFile()
	specFile := defaults.SpecFile()
	policyFile := defaults.PolicyFile()
	reportFile := opts.ReportFile
	if reportFile == "" {
		reportFile = "drift.sarif"
	}
	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return fmt.Errorf("session gate: cwd: %w", cwdErr)
		}
		projectRoot = cwd
	}

	for _, pair := range []struct {
		path string
		name string
		hint string
	}{
		{planFile, "Plan file", "specular plan create"},
		{lockFile, "SpecLock file", "specular spec lock"},
		{specFile, "Spec file", "specular spec new"},
	} {
		if err := ux.ValidateRequiredFile(pair.path, pair.name, pair.hint); err != nil {
			return ux.EnhanceError(err)
		}
	}

	p, planErr := plan.LoadPlan(planFile)
	if planErr != nil {
		return fmt.Errorf("session gate: load plan: %w", planErr)
	}
	lock, lockErr := spec.LoadSpecLock(lockFile)
	if lockErr != nil {
		return fmt.Errorf("session gate: load lock: %w", lockErr)
	}
	s, specErr := spec.LoadSpec(specFile)
	if specErr != nil {
		return fmt.Errorf("session gate: load spec: %w", specErr)
	}

	planDrift := drift.DetectPlanDrift(lock, p)
	codeDrift := drift.DetectCodeDrift(s, lock, drift.CodeDriftOptions{
		ProjectRoot: projectRoot,
	})
	var infraDrift []drift.Finding
	if _, err := os.Stat(policyFile); err == nil {
		pol, polErr := policy.LoadPolicy(policyFile)
		if polErr != nil {
			return fmt.Errorf("session gate: load policy: %w", polErr)
		}
		infraDrift = drift.DetectInfraDrift(drift.InfraDriftOptions{
			Policy:     pol,
			TaskImages: map[string]string{},
		})
	}

	report := drift.GenerateReport(planDrift, codeDrift, infraDrift)
	if !opts.Quiet {
		fmt.Printf("\nSession gate (drift):\n")
		fmt.Printf("  Findings: %d  errors=%d  warnings=%d  info=%d\n",
			report.Summary.TotalFindings, report.Summary.Errors, report.Summary.Warnings, report.Summary.Info)
	}
	sarif := report.ToSARIF()
	if saveErr := drift.SaveSARIF(sarif, reportFile); saveErr != nil {
		return fmt.Errorf("session gate: save SARIF: %w", saveErr)
	}
	if !opts.Quiet {
		fmt.Printf("  SARIF: %s\n", reportFile)
	}
	if report.HasErrors() {
		return fmt.Errorf("drift detection failed with %d errors", report.Summary.Errors)
	}
	if !opts.Quiet && report.IsClean() {
		fmt.Println("  ✓ No drift detected")
	}
	return nil
}
