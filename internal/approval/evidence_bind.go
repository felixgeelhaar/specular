package approval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BindEvidence stamps evidence_id onto open exception records for the given
// resource ids (soft-ALLOW overrules). Rewrites YAML in place. Idempotent when
// evidence_id already matches. Returns the number of records updated.
func BindEvidence(root string, resourceIDs []string, evidenceID string) (int, error) {
	evidenceID = strings.TrimSpace(evidenceID)
	if evidenceID == "" || len(resourceIDs) == 0 {
		return 0, nil
	}
	now := time.Now().UTC()
	updated := 0
	seen := make(map[string]struct{}, len(resourceIDs))
	for _, raw := range resourceIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		n, err := bindOneEvidence(root, id, evidenceID, now)
		if err != nil {
			return updated, err
		}
		updated += n
	}
	return updated, nil
}

func bindOneEvidence(root, resourceID, evidenceID string, now time.Time) (int, error) {
	matches, err := FindByResourceID(root, resourceID)
	if err != nil {
		return 0, err
	}
	var target *Record
	for i := range matches {
		if matches[i].Type != TypeException {
			continue
		}
		if matches[i].IsOpen(now) {
			target = &matches[i]
			break
		}
	}
	if target == nil {
		return 0, nil
	}
	if strings.TrimSpace(target.EvidenceID) == evidenceID {
		return 0, nil
	}
	if strings.TrimSpace(target.Path) == "" {
		return 0, fmt.Errorf("approval bind: %q has empty path", resourceID)
	}
	target.EvidenceID = evidenceID
	absPath := target.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, filepath.FromSlash(target.Path))
	}
	if _, err := os.Stat(absPath); err != nil {
		return 0, fmt.Errorf("approval bind: %w", err)
	}
	if err := rewrite(absPath, target); err != nil {
		return 0, err
	}
	return 1, nil
}
