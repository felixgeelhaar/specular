package provenance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/specular/internal/attestation"
)

// VerifyResult is the machine-readable outcome of schema verification.
// This checks the open protocol envelope — not cryptographic signatures
// (those remain on specular auto verify / attestation.Verify).
type VerifyResult struct {
	OK      bool   `json:"ok"`
	Schema  string `json:"schema,omitempty"`
	Session string `json:"session,omitempty"`
	Source  string `json:"source,omitempty"`
	// Bound is "sibling" when harness/governed/source matched the sibling
	// attestation, or "projected" when only an attestation exists (no
	// .provenance.json on disk — sibling checks skipped). Empty on schema fail.
	Bound  string   `json:"bound,omitempty"`
	Errors []string `json:"errors,omitempty"`
}

// Validate checks required Agent Provenance Protocol v1 fields.
// Unknown extra JSON fields are allowed (agent-neutral extensibility).
func Validate(doc *Document) *VerifyResult {
	res := &VerifyResult{}
	if doc == nil {
		res.Errors = append(res.Errors, "document is nil")
		return res
	}
	res.Schema = doc.Schema
	res.Session = doc.Session
	res.Source = doc.Source
	if strings.TrimSpace(doc.Schema) == "" {
		res.Errors = append(res.Errors, "schema is required")
	} else if doc.Schema != Schema {
		res.Errors = append(res.Errors, fmt.Sprintf("schema %q != %q", doc.Schema, Schema))
	}
	if strings.TrimSpace(doc.Version) == "" {
		res.Errors = append(res.Errors, "version is required")
	} else if doc.Version != Version {
		res.Errors = append(res.Errors, fmt.Sprintf("version %q != %q", doc.Version, Version))
	}
	if strings.TrimSpace(doc.Session) == "" {
		res.Errors = append(res.Errors, "session is required")
	}
	res.OK = len(res.Errors) == 0
	return res
}

// ParseDocument unmarshals a provenance JSON document.
func ParseDocument(raw []byte) (*Document, error) {
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("provenance: parse document: %w", err)
	}
	return &doc, nil
}

// LoadDocumentFile reads and parses a .provenance.json (or any protocol JSON) path.
func LoadDocumentFile(path string) (*Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("provenance: read %s: %w", path, err)
	}
	doc, parseErr := ParseDocument(raw)
	if parseErr != nil {
		return nil, parseErr
	}
	if doc.Source == "" {
		base := filepath.Base(path)
		if strings.HasSuffix(base, ".provenance.json") {
			id := strings.TrimSuffix(base, ".provenance.json")
			if id != "" && id != base {
				doc.Source = AttestationFileRel(id)
			}
		}
	}
	return doc, nil
}

// ProvenanceFileRel returns the relative protocol doc path for a session id.
func ProvenanceFileRel(sessionID string) string {
	id := strings.TrimSpace(sessionID)
	return filepath.ToSlash(filepath.Join(".specular", SessionsDir, id+".provenance.json"))
}

// AttestationFileRel returns the relative attestation path for a session id.
func AttestationFileRel(sessionID string) string {
	id := strings.TrimSpace(sessionID)
	return filepath.ToSlash(filepath.Join(".specular", SessionsDir, id+".attestation.json"))
}

// ValidateBound runs schema Validate then binds the document to its sibling
// .attestation.json (session / harness / governed / source). No crypto.
// Projected-only docs (no .provenance.json on disk) skip sibling checks.
func ValidateBound(doc *Document, root string) *VerifyResult {
	res := Validate(doc)
	if !res.OK || doc == nil {
		return res
	}
	bindSiblingAttestation(res, doc, root)
	res.OK = len(res.Errors) == 0
	return res
}

func bindSiblingAttestation(res *VerifyResult, doc *Document, root string) {
	id := strings.TrimSpace(doc.Session)
	if id == "" {
		return
	}
	provPath := filepath.Join(SessionsPath(root), id+".provenance.json")
	attPath := filepath.Join(SessionsPath(root), id+".attestation.json")
	_, provErr := os.Stat(provPath)
	_, attErr := os.Stat(attPath)
	if os.IsNotExist(provErr) {
		// Projected from attestation only — nothing to bind.
		res.Bound = "projected"
		return
	}
	if attErr != nil {
		res.Errors = append(res.Errors, "sibling attestation missing")
		return
	}
	raw, readErr := os.ReadFile(attPath)
	if readErr != nil {
		res.Errors = append(res.Errors, fmt.Sprintf("sibling attestation unreadable: %v", readErr))
		return
	}
	attHarness, attGoverned, ok := siblingProvenanceFields(raw)
	if !ok {
		res.Errors = append(res.Errors, "sibling attestation parse failed")
		return
	}
	wantSource := AttestationFileRel(id)
	gotSource := filepath.ToSlash(strings.TrimSpace(doc.Source))
	if gotSource != "" {
		okSource := gotSource == wantSource || strings.HasSuffix(gotSource, id+".attestation.json")
		if !okSource {
			res.Errors = append(res.Errors,
				fmt.Sprintf("source %q != sibling attestation %q", doc.Source, wantSource))
		}
	}
	docHarness := strings.TrimSpace(doc.Harness)
	if attHarness != docHarness {
		res.Errors = append(res.Errors,
			fmt.Sprintf("harness %q != attestation harness %q", doc.Harness, attHarness))
	}
	if doc.Governed != attGoverned {
		res.Errors = append(res.Errors,
			fmt.Sprintf("governed %v != attestation governed %v", doc.Governed, attGoverned))
	}
	if len(res.Errors) == 0 {
		res.Bound = "sibling"
	}
}

// siblingProvenanceFields extracts harness/governed without requiring a fully
// signed attestation envelope (binding is not cryptographic).
func siblingProvenanceFields(raw []byte) (harness string, governed bool, ok bool) {
	if att, err := attestation.FromJSON(raw); err == nil && att != nil {
		return strings.TrimSpace(att.Provenance.Harness), att.Provenance.Governed, true
	}
	var payload struct {
		Provenance struct {
			Harness  string `json:"harness"`
			Governed bool   `json:"governed"`
		} `json:"provenance"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", false, false
	}
	return strings.TrimSpace(payload.Provenance.Harness), payload.Provenance.Governed, true
}

// WriteBesideAttestation writes <id>.provenance.json next to an attestation.
// path is the attestation absolute path (…/<id>.attestation.json).
func WriteBesideAttestation(attestationPath string, doc *Document) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("provenance: nil document")
	}
	dir := filepath.Dir(attestationPath)
	base := filepath.Base(attestationPath)
	id := strings.TrimSuffix(base, ".attestation.json")
	if id == base || id == "" {
		id = strings.TrimSpace(doc.Session)
	}
	if id == "" {
		return "", fmt.Errorf("provenance: cannot derive session id for emit")
	}
	if strings.TrimSpace(doc.Session) == "" {
		doc.Session = id
	}
	if doc.Schema == "" {
		doc.Schema = Schema
	}
	if doc.Version == "" {
		doc.Version = Version
	}
	doc.Source = AttestationFileRel(id)
	outPath := filepath.Join(dir, id+".provenance.json")
	data, err := doc.ToJSON()
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if writeErr := os.WriteFile(outPath, data, 0o600); writeErr != nil {
		return "", fmt.Errorf("provenance: write %s: %w", outPath, writeErr)
	}
	return outPath, nil
}

// Resolve loads a protocol document for show/verify.
// Prefer sibling .provenance.json when present; otherwise project from attestation.
// target may be empty (latest), a session id, or a filesystem path to a JSON file.
func Resolve(root, target string) (*Document, error) {
	target = strings.TrimSpace(target)
	if target != "" && (strings.Contains(target, string(os.PathSeparator)) || strings.HasSuffix(target, ".json")) {
		path := target
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if strings.HasSuffix(path, ".provenance.json") {
			return LoadDocumentFile(path)
		}
		if strings.HasSuffix(path, ".attestation.json") {
			id := strings.TrimSuffix(filepath.Base(path), ".attestation.json")
			return loadPreferProvenance(root, id)
		}
		// Generic JSON — try as protocol doc first.
		doc, err := LoadDocumentFile(path)
		if err == nil {
			return doc, nil
		}
		return nil, err
	}
	if target != "" {
		return loadPreferProvenance(root, target)
	}
	return LoadLatestPreferProvenance(root)
}

func loadPreferProvenance(root, sessionID string) (*Document, error) {
	id := strings.TrimSpace(sessionID)
	provPath := filepath.Join(SessionsPath(root), id+".provenance.json")
	if _, err := os.Stat(provPath); err == nil {
		doc, loadErr := LoadDocumentFile(provPath)
		if loadErr != nil {
			return nil, loadErr
		}
		if strings.TrimSpace(doc.Source) == "" {
			doc.Source = AttestationFileRel(id)
		}
		return doc, nil
	}
	return LoadSession(root, id)
}

// LoadLatestPreferProvenance finds the newest attestation and prefers its
// sibling .provenance.json when present.
func LoadLatestPreferProvenance(root string) (*Document, error) {
	doc, err := LoadLatest(root)
	if err != nil {
		return nil, err
	}
	if doc.Session == "" {
		return doc, nil
	}
	return loadPreferProvenance(root, doc.Session)
}

// FormatVerifyHuman renders a verify result for the CLI.
func FormatVerifyHuman(res *VerifyResult) string {
	if res == nil {
		return "VERIFY: FAIL (no result)\n"
	}
	var b strings.Builder
	if res.OK {
		b.WriteString("VERIFY: OK\n")
	} else {
		b.WriteString("VERIFY: FAIL\n")
	}
	if res.Schema != "" {
		fmt.Fprintf(&b, "Schema       %s\n", res.Schema)
	}
	if res.Session != "" {
		fmt.Fprintf(&b, "Session      %s\n", res.Session)
	}
	if res.Source != "" {
		fmt.Fprintf(&b, "Source       %s\n", res.Source)
	}
	switch res.Bound {
	case "sibling":
		b.WriteString("Bound        sibling attestation (harness/governed/source)\n")
	case "projected":
		b.WriteString("Bound        projected (no .provenance.json on disk)\n")
	}
	for _, e := range res.Errors {
		fmt.Fprintf(&b, "Error        %s\n", e)
	}
	writeProvenanceRefs(&b, res.Session, "show")
	return b.String()
}

// FormatProtocolDocsOK annotates gate/evidence protocol counts: ok means
// schema validation plus sibling attestation binding (not crypto).
func FormatProtocolDocsOK(ok, docs int) string {
	return fmt.Sprintf("docs=%d ok=%d schema+bound", docs, ok)
}
