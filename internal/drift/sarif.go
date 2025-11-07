package drift

import (
	"encoding/json"
	"fmt"
	"os"
)

// SARIF represents a SARIF 2.1.0 report structure
type SARIF struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single run in a SARIF report
type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

// SARIFTool describes the tool that generated the report
type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

// SARIFDriver contains tool metadata
type SARIFDriver struct {
	Name            string `json:"name"`
	InformationURI  string `json:"informationUri,omitempty"`
	Version         string `json:"version,omitempty"`
	SemanticVersion string `json:"semanticVersion,omitempty"`
}

// SARIFResult represents a single finding
type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"` // "error", "warning", "note"
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations,omitempty"`
}

// SARIFMessage contains the finding message
type SARIFMessage struct {
	Text string `json:"text"`
}

// SARIFLocation describes where the finding occurred
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation,omitempty"`
}

// SARIFPhysicalLocation provides file-level location
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
}

// SARIFArtifactLocation identifies the artifact
type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// ToSARIF converts a drift report to SARIF format
func (r *Report) ToSARIF() *SARIF {
	sarif := &SARIF{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:            "ai-dev",
						InformationURI:  "https://github.com/felixgeelhaar/specular",
						SemanticVersion: "0.1.0",
					},
				},
				Results: convertFindingsToSARIF(r),
			},
		},
	}
	return sarif
}

// convertFindingsToSARIF converts drift findings to SARIF results
func convertFindingsToSARIF(r *Report) []SARIFResult {
	var results []SARIFResult

	allFindings := append(r.PlanDrift, r.CodeDrift...)
	allFindings = append(allFindings, r.InfraDrift...)

	for _, finding := range allFindings {
		level := "warning"
		switch finding.Severity {
		case "error":
			level = "error"
		case "info":
			level = "note"
		}

		result := SARIFResult{
			RuleID: finding.Code,
			Level:  level,
			Message: SARIFMessage{
				Text: finding.Message,
			},
		}

		// Add location if available
		if finding.Location != "" {
			result.Locations = []SARIFLocation{
				{
					PhysicalLocation: SARIFPhysicalLocation{
						ArtifactLocation: SARIFArtifactLocation{
							URI: finding.Location,
						},
					},
				},
			}
		}

		results = append(results, result)
	}

	return results
}

// SaveSARIF writes a SARIF report to disk
func SaveSARIF(sarif *SARIF, path string) error {
	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal SARIF: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write SARIF file: %w", err)
	}

	return nil
}
