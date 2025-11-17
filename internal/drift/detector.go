package drift

import (
	"fmt"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// DetectPlanDrift compares a plan against the SpecLock to find mismatches
func DetectPlanDrift(lock *spec.SpecLock, p *plan.Plan) []Finding {
	var findings []Finding

	// Check each task in the plan
	for _, task := range p.Tasks {
		// Verify feature exists in SpecLock
		lockedFeature, exists := lock.Features[types.FeatureID(task.FeatureID)]
		if !exists {
			findings = append(findings, Finding{
				Code:      "UNKNOWN_FEATURE",
				FeatureID: task.FeatureID,
				Message:   fmt.Sprintf("Task %s references unknown feature %s", task.ID, task.FeatureID),
				Severity:  "error",
				Location:  fmt.Sprintf("task:%s", task.ID),
			})
			continue
		}

		// Compare hashes
		if task.ExpectedHash != lockedFeature.Hash {
			findings = append(findings, Finding{
				Code:      "HASH_MISMATCH",
				FeatureID: task.FeatureID,
				Message: fmt.Sprintf("Task %s has mismatched hash (expected: %s, got: %s)",
					task.ID, lockedFeature.Hash, task.ExpectedHash),
				Severity: "error",
				Location: fmt.Sprintf("task:%s", task.ID),
			})
		}
	}

	// Check for features in SpecLock not covered by plan
	taskFeatures := make(map[string]bool)
	for _, task := range p.Tasks {
		taskFeatures[task.FeatureID.String()] = true
	}

	for featureID := range lock.Features {
		if !taskFeatures[featureID.String()] {
			findings = append(findings, Finding{
				Code:      "MISSING_TASK",
				FeatureID: featureID,
				Message:   fmt.Sprintf("Feature %s in SpecLock has no corresponding task in plan", featureID),
				Severity:  "warning",
				Location:  fmt.Sprintf("feature:%s", featureID),
			})
		}
	}

	return findings
}

// GenerateReport creates a comprehensive drift report
func GenerateReport(planDrift, codeDrift, infraDrift []Finding) *Report {
	allFindings := append(planDrift, codeDrift...)
	allFindings = append(allFindings, infraDrift...)

	summary := Summary{
		TotalFindings: len(allFindings),
	}

	for _, f := range allFindings {
		switch f.Severity {
		case "error":
			summary.Errors++
		case "warning":
			summary.Warnings++
		case "info":
			summary.Info++
		}
	}

	return &Report{
		PlanDrift:  planDrift,
		CodeDrift:  codeDrift,
		InfraDrift: infraDrift,
		Summary:    summary,
	}
}

// HasErrors returns true if the report contains any error-level findings
func (r *Report) HasErrors() bool {
	return r.Summary.Errors > 0
}

// HasWarnings returns true if the report contains any warning-level findings
func (r *Report) HasWarnings() bool {
	return r.Summary.Warnings > 0
}

// IsClean returns true if the report has no findings
func (r *Report) IsClean() bool {
	return r.Summary.TotalFindings == 0
}
