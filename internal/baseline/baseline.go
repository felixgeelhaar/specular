// Package baseline implements PRODUCT_INTENT §25: explicit, versioned,
// reviewable acknowledgements of the current repository state.
//
// A baseline means "this is the acknowledged current state" — not that the
// state is good. New drift relative to the baseline can then be governed
// without requiring immediate remediation of all historical debt.
package baseline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/gate"
)

// Schema identifies the baseline document format.
const Schema = "specular.baseline/v1"

// FileName is the primary baseline path under .specular/.
const FileName = "baseline.yaml"

// DriftPointerName is a JSON pointer kept for approve pending-drift checks.
const DriftPointerName = "drift-baseline.json"

// Document is an explicit acknowledged repository state.
type Document struct {
	Schema    string    `yaml:"schema" json:"schema"`
	ID        string    `yaml:"id" json:"id"`
	CreatedAt time.Time `yaml:"createdAt" json:"createdAt"`
	Note      string    `yaml:"note,omitempty" json:"note,omitempty"`
	// Disclaimer is always set so readers cannot confuse baseline with approval.
	Disclaimer string   `yaml:"disclaimer" json:"disclaimer"`
	Posture    Posture  `yaml:"posture" json:"posture"`
	Snapshot   Snapshot `yaml:"snapshot" json:"snapshot"`
	EvidenceID string   `yaml:"evidenceId,omitempty" json:"evidenceId,omitempty"`
}

// Posture is the recommended Specular control mode at capture time.
type Posture struct {
	Provenance   string `yaml:"provenance" json:"provenance"`
	IntentDrift  string `yaml:"intentDrift" json:"intentDrift"`
	ScopeDrift   string `yaml:"scopeDrift" json:"scopeDrift"`
	Dependencies string `yaml:"dependencies" json:"dependencies"`
	Secrets      string `yaml:"secrets" json:"secrets"`
	Approvals    string `yaml:"approvals" json:"approvals"`
}

// Snapshot freezes gate section outcomes at capture time.
type Snapshot struct {
	Verdict            string   `yaml:"verdict" json:"verdict"`
	Reason             string   `yaml:"reason" json:"reason"`
	DriftStatus        string   `yaml:"driftStatus" json:"driftStatus"`
	PolicyStatus       string   `yaml:"policyStatus" json:"policyStatus"`
	ProvenanceAttested bool     `yaml:"provenanceAttested" json:"provenanceAttested"`
	DriftErrors        int      `yaml:"driftErrors,omitempty" json:"driftErrors,omitempty"`
	FindingCodes       []string `yaml:"findingCodes,omitempty" json:"findingCodes,omitempty"`
}

const defaultDisclaimer = "Acknowledged current state — not a claim that this state is good."

// Path returns .specular/baseline.yaml under root.
func Path(root string) string {
	return filepath.Join(root, ".specular", FileName)
}

// DefaultPosture returns the PRODUCT_INTENT §24 recommended baseline modes.
func DefaultPosture(approvals string) Posture {
	if approvals == "" {
		approvals = "advisory"
	}
	return Posture{
		Provenance:   "advisory",
		IntentDrift:  "advisory",
		ScopeDrift:   "advisory",
		Dependencies: "blocking",
		Secrets:      "blocking",
		Approvals:    approvals,
	}
}

// Capture builds a Document from a gate result.
func Capture(root string, res *gate.Result, posture Posture, note, evidenceID string) (*Document, error) {
	if res == nil {
		return nil, fmt.Errorf("baseline: nil gate result")
	}
	doc := &Document{
		Schema:     Schema,
		CreatedAt:  time.Now().UTC(),
		Note:       note,
		Disclaimer: defaultDisclaimer,
		Posture:    posture,
		EvidenceID: evidenceID,
		Snapshot: Snapshot{
			Verdict:            string(res.Verdict),
			Reason:             res.Reason,
			DriftStatus:        string(res.Drift.Status),
			PolicyStatus:       string(res.Policy.Status),
			ProvenanceAttested: res.Provenance.Attested,
			DriftErrors:        res.Drift.Errors,
			FindingCodes:       findingCodes(res),
		},
	}
	id, err := contentID(doc)
	if err != nil {
		return nil, err
	}
	doc.ID = id
	return doc, nil
}

func findingCodes(res *gate.Result) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range res.Drift.Findings {
		if f.Code == "" || seen[f.Code] {
			continue
		}
		seen[f.Code] = true
		out = append(out, f.Code)
	}
	return out
}

func contentID(doc *Document) (string, error) {
	payload := struct {
		Schema   string   `json:"schema"`
		Posture  Posture  `json:"posture"`
		Snapshot Snapshot `json:"snapshot"`
	}{Schema: doc.Schema, Posture: doc.Posture, Snapshot: doc.Snapshot}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "bl_" + hex.EncodeToString(sum[:12]), nil
}

// Write persists the YAML baseline and a JSON pointer for approve checks.
func Write(root string, doc *Document) error {
	if doc == nil || doc.ID == "" {
		return fmt.Errorf("baseline: missing document")
	}
	dir := filepath.Join(root, ".specular")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("baseline: mkdir: %w", err)
	}
	raw, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("baseline: marshal: %w", err)
	}
	if writeErr := os.WriteFile(Path(root), raw, 0o600); writeErr != nil {
		return fmt.Errorf("baseline: write: %w", writeErr)
	}
	pointer := map[string]string{
		"schema": Schema,
		"id":     doc.ID,
		"path":   FileName,
	}
	pj, _ := json.MarshalIndent(pointer, "", "  ")
	pj = append(pj, '\n')
	_ = os.WriteFile(filepath.Join(dir, DriftPointerName), pj, 0o600)
	return nil
}

// Load reads the baseline document.
func Load(root string) (*Document, error) {
	raw, err := os.ReadFile(Path(root))
	if err != nil {
		return nil, fmt.Errorf("baseline: load: %w", err)
	}
	var doc Document
	if parseErr := yaml.Unmarshal(raw, &doc); parseErr != nil {
		return nil, fmt.Errorf("baseline: parse: %w", parseErr)
	}
	return &doc, nil
}

// Diff describes new issues relative to an acknowledged baseline.
type Diff struct {
	HasBaseline bool     `json:"hasBaseline"`
	Changed     bool     `json:"changed"`
	Summary     string   `json:"summary"`
	Details     []string `json:"details,omitempty"`
}

// Compare reports whether the current gate result introduces new issues
// beyond the acknowledged baseline snapshot.
func Compare(doc *Document, res *gate.Result) Diff {
	if doc == nil {
		return Diff{HasBaseline: false, Summary: "no baseline — run specular baseline capture"}
	}
	if res == nil {
		return Diff{HasBaseline: true, Summary: "no current gate result"}
	}
	d := Diff{HasBaseline: true}
	baseCodes := map[string]bool{}
	for _, c := range doc.Snapshot.FindingCodes {
		baseCodes[c] = true
	}
	if string(res.Drift.Status) == string(gate.StatusFail) && doc.Snapshot.DriftStatus != string(gate.StatusFail) {
		d.Changed = true
		d.Details = append(d.Details, "drift newly failing vs acknowledged baseline")
	}
	if string(res.Policy.Status) == string(gate.StatusFail) && doc.Snapshot.PolicyStatus != string(gate.StatusFail) {
		d.Changed = true
		d.Details = append(d.Details, "policy newly failing vs acknowledged baseline")
	}
	for _, f := range res.Drift.Findings {
		if f.Code == "" || baseCodes[f.Code] {
			continue
		}
		if !strings.EqualFold(f.Severity, "error") {
			continue
		}
		d.Changed = true
		d.Details = append(d.Details, "new error finding: "+f.Code)
	}
	if !d.Changed {
		d.Summary = "no new blocking issues beyond baseline"
		return d
	}
	d.Summary = fmt.Sprintf("%d change(s) beyond baseline", len(d.Details))
	return d
}

// FormatText renders a human baseline board.
func FormatText(doc *Document) string {
	if doc == nil {
		return "No baseline. Run: specular baseline capture\n"
	}
	var b strings.Builder
	b.WriteString("SPECULAR BASELINE\n")
	b.WriteString(strings.Repeat("─", 46) + "\n")
	fmt.Fprintf(&b, "ID             %s\n", doc.ID)
	fmt.Fprintf(&b, "Captured       %s\n", doc.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Disclaimer     %s\n", doc.Disclaimer)
	if doc.Note != "" {
		fmt.Fprintf(&b, "Note           %s\n", doc.Note)
	}
	if doc.EvidenceID != "" {
		fmt.Fprintf(&b, "Evidence       %s\n", doc.EvidenceID)
	}
	b.WriteString("Posture\n")
	fmt.Fprintf(&b, "  provenance      %s\n", doc.Posture.Provenance)
	fmt.Fprintf(&b, "  intent drift    %s\n", doc.Posture.IntentDrift)
	fmt.Fprintf(&b, "  scope drift     %s\n", doc.Posture.ScopeDrift)
	fmt.Fprintf(&b, "  dependencies    %s\n", doc.Posture.Dependencies)
	fmt.Fprintf(&b, "  secrets         %s\n", doc.Posture.Secrets)
	fmt.Fprintf(&b, "  approvals       %s\n", doc.Posture.Approvals)
	b.WriteString("Acknowledged snapshot\n")
	fmt.Fprintf(&b, "  Verdict         %s\n", doc.Snapshot.Verdict)
	fmt.Fprintf(&b, "  Drift           %s\n", doc.Snapshot.DriftStatus)
	fmt.Fprintf(&b, "  Policy          %s\n", doc.Snapshot.PolicyStatus)
	fmt.Fprintf(&b, "  Provenance      attested=%v\n", doc.Snapshot.ProvenanceAttested)
	if doc.Snapshot.Reason != "" {
		fmt.Fprintf(&b, "  Reason          %s\n", doc.Snapshot.Reason)
	}
	b.WriteString(strings.Repeat("─", 46) + "\n")
	b.WriteString("File: .specular/baseline.yaml\n")
	return b.String()
}

// FormatDiff renders Compare output.
func FormatDiff(d Diff) string {
	var b strings.Builder
	b.WriteString("SPECULAR BASELINE STATUS\n")
	b.WriteString(strings.Repeat("─", 46) + "\n")
	fmt.Fprintf(&b, "%s\n", d.Summary)
	for _, line := range d.Details {
		fmt.Fprintf(&b, "  • %s\n", line)
	}
	return b.String()
}

// DetectApprovalsLabel mirrors brownfield init CODEOWNERS detection.
func DetectApprovalsLabel(root string) string {
	for _, p := range []string{
		filepath.Join(root, "CODEOWNERS"),
		filepath.Join(root, ".github", "CODEOWNERS"),
		filepath.Join(root, "docs", "CODEOWNERS"),
	} {
		if _, err := os.Stat(p); err == nil {
			return "existing CODEOWNERS"
		}
	}
	return "advisory"
}
