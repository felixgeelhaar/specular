package evidence

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/gate"
)

// ListFilter selects local evidence records. Zero-valued fields mean no constraint.
// Sorting is always newest CreatedAt first, then ID ascending for stability.
type ListFilter struct {
	Verdict         gate.Verdict // ALLOW or DENY; empty = any
	Since           time.Time    // inclusive lower bound on CreatedAt; zero = any
	PathContains    string       // substring match on root / finding paths
	ControlContains string       // substring match on failed checks / exception policy / overrule bind
	CommitPrefix    string       // case-insensitive prefix match on record Commit / gate.change.commit
	RiskLevel       string       // NONE|LOW|MEDIUM|HIGH|CRITICAL; empty = any
	Session         string       // exact match against gate.provenance.sessions[]
	Harness         string       // case-insensitive substring against harnesses[]
	SoftAllow       *bool        // nil = any; true = has exception overrules; false = none
	Attested        *bool        // nil = any; true/false = gate.provenance.attested
	Governed        *bool        // nil = any; true/false = gate.provenance.governed
	Protocol        *bool        // nil = any; true = APP docs present+schema+bound; false = missing/invalid/unbound
	Limit           int          // max results; <=0 = unlimited
}

// List returns evidence records matching filter, newest-first.
func List(root string, filter ListFilter) ([]*Record, error) {
	if err := filter.validate(); err != nil {
		return nil, err
	}
	dir := Dir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []*Record
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		rec, loadErr := Load(root, id)
		if loadErr != nil {
			continue
		}
		if !filter.Match(rec) {
			continue
		}
		out = append(out, rec)
	}

	sort.SliceStable(out, func(i, j int) bool {
		ti, tj := out[i].CreatedAt, out[j].CreatedAt
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return out[i].ID < out[j].ID
	})

	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// Match reports whether rec satisfies the filter.
func (f ListFilter) Match(rec *Record) bool {
	if rec == nil {
		return false
	}
	if !f.matchVerdict(rec) || !f.matchSince(rec) || !f.matchPath(rec) || !f.matchControl(rec) || !f.matchCommit(rec) {
		return false
	}
	if !f.matchRisk(rec) || !f.matchSession(rec) || !f.matchHarness(rec) {
		return false
	}
	if !f.matchSoftAllow(rec) || !f.matchAttested(rec) || !f.matchGoverned(rec) || !f.matchProtocol(rec) {
		return false
	}
	return true
}

func (f ListFilter) matchVerdict(rec *Record) bool {
	if f.Verdict == "" {
		return true
	}
	return rec.Gate != nil && rec.Gate.Verdict == f.Verdict
}

func (f ListFilter) matchSince(rec *Record) bool {
	if f.Since.IsZero() {
		return true
	}
	return !rec.CreatedAt.IsZero() && !rec.CreatedAt.Before(f.Since)
}

func (f ListFilter) matchPath(rec *Record) bool {
	sub := strings.TrimSpace(f.PathContains)
	if sub == "" {
		return true
	}
	return pathContains(rec, sub)
}

func (f ListFilter) matchControl(rec *Record) bool {
	sub := strings.TrimSpace(f.ControlContains)
	if sub == "" {
		return true
	}
	return controlContains(rec, sub)
}

func (f ListFilter) matchCommit(rec *Record) bool {
	prefix := strings.TrimSpace(f.CommitPrefix)
	if prefix == "" {
		return true
	}
	return commitPrefixMatch(rec, prefix)
}

func (f ListFilter) matchRisk(rec *Record) bool {
	level := strings.ToUpper(strings.TrimSpace(f.RiskLevel))
	if level == "" {
		return true
	}
	got := ""
	if rec.Gate != nil {
		got = strings.ToUpper(strings.TrimSpace(rec.Gate.Risk.Level))
	}
	if got == "" {
		got = "NONE"
	}
	return got == level
}

func (f ListFilter) matchSession(rec *Record) bool {
	session := strings.TrimSpace(f.Session)
	if session == "" {
		return true
	}
	return provenanceSessionMatch(rec, session)
}

func (f ListFilter) matchHarness(rec *Record) bool {
	harness := strings.TrimSpace(f.Harness)
	if harness == "" {
		return true
	}
	return provenanceHarnessContains(rec, harness)
}

func (f ListFilter) matchSoftAllow(rec *Record) bool {
	if f.SoftAllow == nil {
		return true
	}
	has := rec != nil && rec.Gate != nil && len(rec.Gate.Approvals.Overrules) > 0
	return has == *f.SoftAllow
}

func (f ListFilter) matchAttested(rec *Record) bool {
	if f.Attested == nil {
		return true
	}
	attested := rec != nil && rec.Gate != nil && rec.Gate.Provenance.Attested
	return attested == *f.Attested
}

func (f ListFilter) matchGoverned(rec *Record) bool {
	if f.Governed == nil {
		return true
	}
	governed := rec != nil && rec.Gate != nil && rec.Gate.Provenance.Governed
	return governed == *f.Governed
}

func (f ListFilter) matchProtocol(rec *Record) bool {
	if f.Protocol == nil {
		return true
	}
	ok := protocolDocsOK(rec)
	return ok == *f.Protocol
}

func protocolDocsOK(rec *Record) bool {
	// ProtocolOK already means schema + sibling attestation binding for
	// records written after ValidateBound (#117); filter compares stored counts.
	if rec == nil || rec.Gate == nil {
		return false
	}
	docs := rec.Gate.Provenance.ProtocolDocs
	ok := rec.Gate.Provenance.ProtocolOK
	return docs > 0 && ok >= docs
}

// Active reports whether any selection constraint is set (ignores Limit).
func (f ListFilter) Active() bool {
	return f.Verdict != "" || !f.Since.IsZero() || strings.TrimSpace(f.PathContains) != "" ||
		strings.TrimSpace(f.ControlContains) != "" || strings.TrimSpace(f.CommitPrefix) != "" ||
		strings.TrimSpace(f.RiskLevel) != "" || strings.TrimSpace(f.Session) != "" ||
		strings.TrimSpace(f.Harness) != "" || f.SoftAllow != nil || f.Attested != nil ||
		f.Governed != nil || f.Protocol != nil
}

func (f ListFilter) validate() error {
	if f.Verdict != "" {
		switch f.Verdict {
		case gate.Allow, gate.Deny:
		default:
			return fmt.Errorf("evidence: invalid verdict %q (want ALLOW or DENY)", f.Verdict)
		}
	}
	if level := strings.TrimSpace(f.RiskLevel); level != "" {
		switch strings.ToUpper(level) {
		case "NONE", "LOW", "MEDIUM", "HIGH", "CRITICAL":
		default:
			return fmt.Errorf("evidence: invalid risk %q (want NONE|LOW|MEDIUM|HIGH|CRITICAL)", level)
		}
	}
	return nil
}

// ParseSince parses a duration (e.g. "24h", "30m") or RFC3339 timestamp.
// Durations are subtracted from now (typically time.Now().UTC()).
func ParseSince(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		if d < 0 {
			return time.Time{}, fmt.Errorf("evidence: since duration must be non-negative")
		}
		return now.Add(-d), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("evidence: since %q: want duration (24h) or RFC3339", s)
}

func pathContains(rec *Record, sub string) bool {
	subLower := strings.ToLower(sub)
	for _, p := range recordPathStrings(rec) {
		if strings.Contains(strings.ToLower(p), subLower) {
			return true
		}
	}
	return false
}

func controlContains(rec *Record, sub string) bool {
	subLower := strings.ToLower(sub)
	for _, s := range recordControlStrings(rec) {
		if strings.Contains(strings.ToLower(s), subLower) {
			return true
		}
	}
	return false
}

func commitPrefixMatch(rec *Record, prefix string) bool {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return false
	}
	for _, c := range recordCommitStrings(rec) {
		c = strings.ToLower(strings.TrimSpace(c))
		if c != "" && strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func recordCommitStrings(rec *Record) []string {
	if rec == nil {
		return nil
	}
	var out []string
	if rec.Commit != "" {
		out = append(out, rec.Commit)
	}
	if rec.Gate != nil && rec.Gate.Change.Commit != "" {
		out = append(out, rec.Gate.Change.Commit)
	}
	return out
}

// recordControlStrings collects policy/control tokens from failed checks,
// exception --policy fields, and soft-ALLOW overrule kind/binding.
func recordControlStrings(rec *Record) []string {
	if rec == nil || rec.Gate == nil {
		return nil
	}
	g := rec.Gate
	var out []string
	out = append(out, g.Policy.FailedChecks...)
	for _, ex := range g.Approvals.Exceptions {
		if ex.Policy != "" {
			out = append(out, ex.Policy)
		}
		if ex.Scope != "" {
			out = append(out, ex.Scope)
		}
	}
	for _, o := range g.Approvals.Overrules {
		if o.Kind != "" {
			out = append(out, o.Kind)
		}
		if o.Binding != "" {
			out = append(out, o.Binding)
		}
		if o.Policy != "" {
			out = append(out, o.Policy)
		}
		if o.ResourceID != "" {
			out = append(out, o.ResourceID)
		}
	}
	return out
}

func provenanceSessionMatch(rec *Record, id string) bool {
	if rec == nil || rec.Gate == nil {
		return false
	}
	for _, s := range rec.Gate.Provenance.Sessions {
		if strings.TrimSpace(s) == id {
			return true
		}
	}
	return false
}

func provenanceHarnessContains(rec *Record, sub string) bool {
	if rec == nil || rec.Gate == nil {
		return false
	}
	subLower := strings.ToLower(sub)
	for _, h := range rec.Gate.Provenance.Harnesses {
		if strings.Contains(strings.ToLower(h), subLower) {
			return true
		}
	}
	return false
}

func recordPathStrings(rec *Record) []string {
	var paths []string
	if rec.Root != "" {
		paths = append(paths, rec.Root)
	}
	if rec.Gate == nil {
		return paths
	}
	if rec.Gate.Change.Root != "" {
		paths = append(paths, rec.Gate.Change.Root)
	}
	for _, f := range rec.Gate.Drift.Findings {
		if f.Path != "" {
			paths = append(paths, f.Path)
		}
		if f.Location != "" {
			paths = append(paths, f.Location)
		}
	}
	return paths
}
