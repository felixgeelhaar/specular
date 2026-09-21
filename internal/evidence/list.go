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
	Verdict      gate.Verdict // ALLOW or DENY; empty = any
	Since        time.Time    // inclusive lower bound on CreatedAt; zero = any
	PathContains string       // substring match on root / finding paths
	Limit        int          // max results; <=0 = unlimited
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
	if f.Verdict != "" {
		if rec.Gate == nil || rec.Gate.Verdict != f.Verdict {
			return false
		}
	}
	if !f.Since.IsZero() {
		if rec.CreatedAt.IsZero() || rec.CreatedAt.Before(f.Since) {
			return false
		}
	}
	if sub := strings.TrimSpace(f.PathContains); sub != "" {
		if !pathContains(rec, sub) {
			return false
		}
	}
	return true
}

func (f ListFilter) validate() error {
	if f.Verdict == "" {
		return nil
	}
	switch f.Verdict {
	case gate.Allow, gate.Deny:
		return nil
	default:
		return fmt.Errorf("evidence: invalid verdict %q (want ALLOW or DENY)", f.Verdict)
	}
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
