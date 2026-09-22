package cmd

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/specular/internal/evidence"
	"github.com/felixgeelhaar/specular/internal/gate"
)

// sessionProductGateOptions configures the product change-control gate used by
// `session wait --gate` / `--bundle` (same Evaluate path as `specular gate`).
type sessionProductGateOptions struct {
	ProjectRoot     string
	PolicyPath      string
	ReportFile      string
	StrictSpec      bool
	RequireAttested bool
	RequireProtocol bool
	RequireGoverned bool
	NoEvidence      bool
	Quiet           bool
}

// runSessionProductGate runs specular's product gate (provenance → drift →
// policy → evidence). Brownfield soft-skips missing specs unless StrictSpec.
// DENY maps to the same exit-code strings as `specular gate` (drift → 4,
// policy/provenance/risk → 3).
func runSessionProductGate(opts sessionProductGateOptions) error {
	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return fmt.Errorf("session gate: cwd: %w", cwdErr)
		}
		projectRoot = cwd
	}
	reportFile := opts.ReportFile
	if reportFile == "" {
		reportFile = "drift.sarif"
	}

	res, err := gate.Evaluate(gate.Options{
		ProjectRoot:     projectRoot,
		PolicyPath:      opts.PolicyPath,
		ReportFile:      reportFile,
		StrictSpec:      opts.StrictSpec,
		RequireAttested: opts.RequireAttested,
		RequireProtocol: opts.RequireProtocol,
		RequireGoverned: opts.RequireGoverned,
	})
	if err != nil {
		return fmt.Errorf("session gate: %w", err)
	}

	if !opts.NoEvidence {
		if rec, recErr := evidence.NewFromGate(projectRoot, res); recErr == nil {
			if writeErr := evidence.Write(projectRoot, rec); writeErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not persist evidence: %v\n", writeErr)
			} else if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "Evidence: %s (.specular/evidence/)\n", rec.ID)
			}
		}
	}

	if !opts.Quiet {
		fmt.Print(gate.FormatText(res))
	}

	if res.Verdict == gate.Deny {
		return sessionGateDenyError(res)
	}
	return nil
}

func sessionGateDenyError(res *gate.Result) error {
	if res == nil {
		return fmt.Errorf("gate denied")
	}
	if res.Drift.Status == gate.StatusFail {
		return fmt.Errorf("drift detection failed: %s", res.Reason)
	}
	if res.Policy.Status == gate.StatusFail {
		return fmt.Errorf("policy violation: %s", res.Reason)
	}
	if res.Risk.Enforced && len(res.Risk.Missing) > 0 {
		return fmt.Errorf("policy violation: %s", res.Reason)
	}
	if res.Provenance.Enforced && res.Provenance.Status == gate.StatusFail {
		return fmt.Errorf("policy violation: %s", res.Reason)
	}
	return fmt.Errorf("gate denied: %s", res.Reason)
}
