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
	"sort"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/gate"
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

// ListIDs returns evidence IDs newest-first by mtime.
func ListIDs(root string) ([]string, error) {
	dir := Dir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	type item struct {
		id  string
		mod time.Time
	}
	var items []item
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		items = append(items, item{
			id:  strings.TrimSuffix(name, ".json"),
			mod: info.ModTime(),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].mod.After(items[j].mod)
	})
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.id
	}
	return out, nil
}

// FormatExplain renders a human explanation of why the gate decided.
func FormatExplain(rec *Record) string {
	if rec == nil || rec.Gate == nil {
		return "No evidence record.\n"
	}
	g := rec.Gate
	var b strings.Builder
	b.WriteString("SPECULAR EXPLAIN\n")
	b.WriteString(strings.Repeat("─", 46) + "\n")
	fmt.Fprintf(&b, "Evidence       %s\n", rec.ID)
	fmt.Fprintf(&b, "Recorded       %s\n", rec.CreatedAt.Format(time.RFC3339))
	if rec.Branch != "" {
		fmt.Fprintf(&b, "Branch         %s\n", rec.Branch)
	}
	if rec.Commit != "" {
		fmt.Fprintf(&b, "Commit         %s\n", rec.Commit)
	}
	b.WriteString(strings.Repeat("─", 46) + "\n")
	fmt.Fprintf(&b, "VERDICT: %s\n", g.Verdict)
	fmt.Fprintf(&b, "REASON:  %s\n", g.Reason)
	b.WriteString("\nWhy this decision?\n")
	writeWhy(&b, g)
	if len(g.Drift.Findings) > 0 {
		b.WriteString("\nDrift findings\n")
		limit := len(g.Drift.Findings)
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			f := g.Drift.Findings[i]
			fmt.Fprintf(&b, "  [%s] %s", f.Severity, f.Code)
			if f.Category != "" {
				fmt.Fprintf(&b, " (%s)", f.Category)
			}
			b.WriteString("\n")
			if f.Message != "" {
				fmt.Fprintf(&b, "    %s\n", f.Message)
			}
			if f.Location != "" {
				fmt.Fprintf(&b, "    at %s\n", f.Location)
			}
		}
	}
	b.WriteString(strings.Repeat("─", 46) + "\n")
	b.WriteString("Re-run: specular gate\n")
	b.WriteString("Store:  .specular/evidence/\n")
	return b.String()
}

func writeWhy(b *strings.Builder, g *gate.Result) {
	writeSectionWhy(b, "Drift", g.Drift.Status, g.Drift.Note, map[gate.SectionStatus]string{
		gate.StatusSkipped: "Drift skipped (brownfield / missing spec)",
	})
	writeSectionWhy(b, "Policy", g.Policy.Status, g.Policy.Note, nil)
	if g.Provenance.Attested {
		b.WriteString("  • Provenance attested")
		if len(g.Provenance.Harnesses) > 0 {
			fmt.Fprintf(b, " (%s)", strings.Join(g.Provenance.Harnesses, ", "))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("  • Provenance unattested — not treated as verified\n")
	}
	if g.Verdict == gate.Allow {
		b.WriteString("  → ALLOW because no blocking drift or policy failure was present.\n")
	} else {
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
