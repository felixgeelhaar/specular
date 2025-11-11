package drift

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestToSARIF(t *testing.T) {
	tests := []struct {
		name            string
		report          *Report
		wantVersion     string
		wantResultCount int
	}{
		{
			name: "empty report",
			report: &Report{
				PlanDrift:  []Finding{},
				CodeDrift:  []Finding{},
				InfraDrift: []Finding{},
			},
			wantVersion:     "2.1.0",
			wantResultCount: 0,
		},
		{
			name: "report with plan drift",
			report: &Report{
				PlanDrift: []Finding{
					{
						Code:     "PLAN-001",
						Message:  "Hash mismatch",
						Severity: "error",
						Location: "spec.yaml",
					},
				},
				CodeDrift:  []Finding{},
				InfraDrift: []Finding{},
			},
			wantVersion:     "2.1.0",
			wantResultCount: 1,
		},
		{
			name: "report with code drift",
			report: &Report{
				PlanDrift: []Finding{},
				CodeDrift: []Finding{
					{
						Code:     "CODE-001",
						Message:  "Coverage below threshold",
						Severity: "warning",
						Location: "src/main.go",
					},
				},
				InfraDrift: []Finding{},
			},
			wantVersion:     "2.1.0",
			wantResultCount: 1,
		},
		{
			name: "report with infra drift",
			report: &Report{
				PlanDrift: []Finding{},
				CodeDrift: []Finding{},
				InfraDrift: []Finding{
					{
						Code:     "INFRA-001",
						Message:  "Image not in allowlist",
						Severity: "error",
						Location: "policy.yaml",
					},
				},
			},
			wantVersion:     "2.1.0",
			wantResultCount: 1,
		},
		{
			name: "report with multiple findings",
			report: &Report{
				PlanDrift: []Finding{
					{
						Code:     "PLAN-001",
						Message:  "Hash mismatch",
						Severity: "error",
						Location: "spec.yaml",
					},
				},
				CodeDrift: []Finding{
					{
						Code:     "CODE-001",
						Message:  "Coverage below threshold",
						Severity: "warning",
						Location: "src/main.go",
					},
					{
						Code:     "CODE-002",
						Message:  "Missing API endpoint",
						Severity: "info",
						Location: "src/api.go",
					},
				},
				InfraDrift: []Finding{
					{
						Code:     "INFRA-001",
						Message:  "Image not in allowlist",
						Severity: "error",
						Location: "policy.yaml",
					},
				},
			},
			wantVersion:     "2.1.0",
			wantResultCount: 4,
		},
		{
			name: "report with finding without location",
			report: &Report{
				PlanDrift: []Finding{
					{
						Code:     "PLAN-002",
						Message:  "General drift detected",
						Severity: "warning",
						Location: "", // No location
					},
				},
				CodeDrift:  []Finding{},
				InfraDrift: []Finding{},
			},
			wantVersion:     "2.1.0",
			wantResultCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sarif := tt.report.ToSARIF()

			if sarif == nil {
				t.Fatal("ToSARIF() returned nil")
			}

			if sarif.Version != tt.wantVersion {
				t.Errorf("ToSARIF() Version = %v, want %v", sarif.Version, tt.wantVersion)
			}

			if sarif.Schema == "" {
				t.Error("ToSARIF() Schema is empty")
			}

			if len(sarif.Runs) != 1 {
				t.Errorf("ToSARIF() Runs count = %d, want 1", len(sarif.Runs))
			}

			if len(sarif.Runs) > 0 {
				run := sarif.Runs[0]

				if run.Tool.Driver.Name != "specular" {
					t.Errorf("ToSARIF() Tool Name = %v, want specular", run.Tool.Driver.Name)
				}

				if len(run.Results) != tt.wantResultCount {
					t.Errorf("ToSARIF() Results count = %d, want %d", len(run.Results), tt.wantResultCount)
				}

				// Verify severity mapping
				for i, finding := range append(append(tt.report.PlanDrift, tt.report.CodeDrift...), tt.report.InfraDrift...) {
					if i >= len(run.Results) {
						break
					}
					result := run.Results[i]

					expectedLevel := "warning"
					switch finding.Severity {
					case "error":
						expectedLevel = "error"
					case "info":
						expectedLevel = "note"
					}

					if result.Level != expectedLevel {
						t.Errorf("ToSARIF() Result[%d] Level = %v, want %v (from severity %v)",
							i, result.Level, expectedLevel, finding.Severity)
					}

					if result.RuleID != finding.Code {
						t.Errorf("ToSARIF() Result[%d] RuleID = %v, want %v", i, result.RuleID, finding.Code)
					}

					if result.Message.Text != finding.Message {
						t.Errorf("ToSARIF() Result[%d] Message = %v, want %v", i, result.Message.Text, finding.Message)
					}

					// Check location if present
					if finding.Location != "" {
						if len(result.Locations) == 0 {
							t.Errorf("ToSARIF() Result[%d] has no locations, expected location for %s", i, finding.Location)
						} else if result.Locations[0].PhysicalLocation.ArtifactLocation.URI != finding.Location {
							t.Errorf("ToSARIF() Result[%d] Location = %v, want %v",
								i, result.Locations[0].PhysicalLocation.ArtifactLocation.URI, finding.Location)
						}
					} else {
						if len(result.Locations) > 0 {
							t.Errorf("ToSARIF() Result[%d] has locations, expected none", i)
						}
					}
				}
			}
		})
	}
}

func TestSaveSARIF(t *testing.T) {
	tests := []struct {
		name    string
		sarif   *SARIF
		wantErr bool
	}{
		{
			name: "save empty SARIF",
			sarif: &SARIF{
				Version: "2.1.0",
				Schema:  "https://example.com/schema",
				Runs:    []SARIFRun{},
			},
			wantErr: false,
		},
		{
			name: "save SARIF with results",
			sarif: &SARIF{
				Version: "2.1.0",
				Schema:  "https://example.com/schema",
				Runs: []SARIFRun{
					{
						Tool: SARIFTool{
							Driver: SARIFDriver{
								Name:    "test-tool",
								Version: "1.0.0",
							},
						},
						Results: []SARIFResult{
							{
								RuleID: "TEST-001",
								Level:  "error",
								Message: SARIFMessage{
									Text: "Test error",
								},
								Locations: []SARIFLocation{
									{
										PhysicalLocation: SARIFPhysicalLocation{
											ArtifactLocation: SARIFArtifactLocation{
												URI: "test.go",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sarifPath := filepath.Join(tmpDir, "report.sarif")

			err := SaveSARIF(tt.sarif, sarifPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("SaveSARIF() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify file was created
			if _, err := os.Stat(sarifPath); os.IsNotExist(err) {
				t.Error("SaveSARIF() did not create file")
				return
			}

			// Verify file can be loaded back
			data, err := os.ReadFile(sarifPath)
			if err != nil {
				t.Fatalf("Failed to read saved SARIF file: %v", err)
			}

			var loaded SARIF
			if err := json.Unmarshal(data, &loaded); err != nil {
				t.Fatalf("Failed to unmarshal saved SARIF: %v", err)
			}

			// Verify content
			if loaded.Version != tt.sarif.Version {
				t.Errorf("Loaded SARIF Version = %v, want %v", loaded.Version, tt.sarif.Version)
			}

			if len(loaded.Runs) != len(tt.sarif.Runs) {
				t.Errorf("Loaded SARIF Runs count = %d, want %d", len(loaded.Runs), len(tt.sarif.Runs))
			}

			if len(loaded.Runs) > 0 && len(tt.sarif.Runs) > 0 {
				if len(loaded.Runs[0].Results) != len(tt.sarif.Runs[0].Results) {
					t.Errorf("Loaded SARIF Results count = %d, want %d",
						len(loaded.Runs[0].Results), len(tt.sarif.Runs[0].Results))
				}
			}
		})
	}
}

func TestSARIFRoundTrip(t *testing.T) {
	// Create a report with various findings
	report := &Report{
		PlanDrift: []Finding{
			{
				Code:     "PLAN-001",
				Message:  "Hash mismatch detected",
				Severity: "error",
				Location: "spec.yaml",
			},
		},
		CodeDrift: []Finding{
			{
				Code:     "CODE-001",
				Message:  "Coverage below 80%",
				Severity: "warning",
				Location: "src/main.go",
			},
			{
				Code:     "CODE-002",
				Message:  "Missing endpoint implementation",
				Severity: "info",
				Location: "src/api.go",
			},
		},
		InfraDrift: []Finding{
			{
				Code:     "INFRA-001",
				Message:  "Image not allowed",
				Severity: "error",
				Location: "docker-compose.yml",
			},
		},
	}

	// Convert to SARIF
	sarif := report.ToSARIF()

	// Save to file
	tmpDir := t.TempDir()
	sarifPath := filepath.Join(tmpDir, "drift-report.sarif")

	if err := SaveSARIF(sarif, sarifPath); err != nil {
		t.Fatalf("SaveSARIF() failed: %v", err)
	}

	// Load back
	data, err := os.ReadFile(sarifPath)
	if err != nil {
		t.Fatalf("Failed to read SARIF file: %v", err)
	}

	var loaded SARIF
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal SARIF: %v", err)
	}

	// Verify round-trip integrity
	if loaded.Version != "2.1.0" {
		t.Errorf("Round-trip Version = %v, want 2.1.0", loaded.Version)
	}

	if len(loaded.Runs) != 1 {
		t.Fatalf("Round-trip Runs count = %d, want 1", len(loaded.Runs))
	}

	run := loaded.Runs[0]
	if run.Tool.Driver.Name != "specular" {
		t.Errorf("Round-trip Tool Name = %v, want specular", run.Tool.Driver.Name)
	}

	expectedResultCount := len(report.PlanDrift) + len(report.CodeDrift) + len(report.InfraDrift)
	if len(run.Results) != expectedResultCount {
		t.Errorf("Round-trip Results count = %d, want %d", len(run.Results), expectedResultCount)
	}

	// Verify all findings are present
	resultCodes := make(map[string]bool)
	for _, result := range run.Results {
		resultCodes[result.RuleID] = true
	}

	allFindings := append(append(report.PlanDrift, report.CodeDrift...), report.InfraDrift...)
	for _, finding := range allFindings {
		if !resultCodes[finding.Code] {
			t.Errorf("Round-trip missing finding with code: %s", finding.Code)
		}
	}
}
