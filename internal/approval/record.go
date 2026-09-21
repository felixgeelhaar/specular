// Package approval stores local governance approval and exception records
// under .specular/approvals/ (PRODUCT_INTENT §17–§18). This is a repo-local
// trail — not a control plane.
package approval

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DirName is the directory under .specular for approval records.
const DirName = "approvals"

// SchemaVersion is the on-disk record version.
const SchemaVersion = "1.0"

// Known record types.
const (
	TypeBundle    = "bundle"
	TypeDrift     = "drift"
	TypePolicy    = "policy"
	TypePlan      = "plan"
	TypeException = "exception"
)

// Record is one governance approval or controlled exception.
type Record struct {
	Version      string            `yaml:"version" json:"version"`
	Type         string            `yaml:"type" json:"type"`
	ResourceID   string            `yaml:"resource_id" json:"resource_id"`
	ResourceHash string            `yaml:"resource_hash,omitempty" json:"resource_hash,omitempty"`
	ApprovedBy   string            `yaml:"approved_by" json:"approved_by"`
	ApprovedAt   time.Time         `yaml:"approved_at" json:"approved_at"`
	Message      string            `yaml:"message,omitempty" json:"message,omitempty"`
	Reason       string            `yaml:"reason,omitempty" json:"reason,omitempty"`
	Scope        string            `yaml:"scope,omitempty" json:"scope,omitempty"`
	Policy       string            `yaml:"policy,omitempty" json:"policy,omitempty"`
	Requester    string            `yaml:"requester,omitempty" json:"requester,omitempty"`
	ExpiresAt    *time.Time        `yaml:"expires_at,omitempty" json:"expires_at,omitempty"`
	Artifact     string            `yaml:"artifact,omitempty" json:"artifact,omitempty"`
	EvidenceID   string            `yaml:"evidence_id,omitempty" json:"evidence_id,omitempty"`
	Metadata     map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`

	// Path is the relative file path when loaded (not written to disk).
	Path string `yaml:"-" json:"path,omitempty"`
}

// Dir returns .specular/approvals under root.
func Dir(root string) string {
	if root == "" {
		return filepath.Join(".specular", DirName)
	}
	return filepath.Join(root, ".specular", DirName)
}

// TypeFromResourceID maps a resource id prefix to a record type.
func TypeFromResourceID(resourceID string) (string, error) {
	id := strings.TrimSpace(resourceID)
	switch {
	case strings.HasPrefix(id, "bundle-"):
		return TypeBundle, nil
	case strings.HasPrefix(id, "drift-"):
		return TypeDrift, nil
	case strings.HasPrefix(id, "policy-"):
		return TypePolicy, nil
	case strings.HasPrefix(id, "plan-"):
		return TypePlan, nil
	case id == TypeException || strings.HasPrefix(id, "exception-"):
		return TypeException, nil
	default:
		return "", fmt.Errorf("unknown resource type: %s (expected bundle-*, drift-*, policy-*, plan-*, or exception-*)", resourceID)
	}
}

// IsExpired reports whether the exception/approval has passed ExpiresAt.
func (r *Record) IsExpired(now time.Time) bool {
	if r == nil || r.ExpiresAt == nil {
		return false
	}
	return !r.ExpiresAt.After(now)
}

// Write persists a record under .specular/approvals/ and returns the file path.
func Write(root string, rec *Record) (string, error) {
	if rec == nil {
		return "", fmt.Errorf("approval: nil record")
	}
	if rec.Version == "" {
		rec.Version = SchemaVersion
	}
	if rec.Type == "" {
		t, err := TypeFromResourceID(rec.ResourceID)
		if err != nil {
			return "", err
		}
		rec.Type = t
	}
	if rec.ResourceID == "" {
		return "", fmt.Errorf("approval: resource_id is required")
	}
	if rec.ApprovedBy == "" {
		return "", fmt.Errorf("approval: approved_by is required")
	}
	if rec.ApprovedAt.IsZero() {
		rec.ApprovedAt = time.Now().UTC()
	}
	if rec.Type == TypeException && strings.TrimSpace(rec.Reason) == "" {
		return "", fmt.Errorf("approval: exception reason is required")
	}

	dir := Dir(root)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("approval: mkdir: %w", err)
	}

	timestamp := rec.ApprovedAt.UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.yaml", rec.Type, timestamp)
	path := filepath.Join(dir, filename)

	data, err := yaml.Marshal(rec)
	if err != nil {
		return "", fmt.Errorf("approval: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("approval: write: %w", err)
	}
	rec.Path = filepath.ToSlash(filepath.Join(".specular", DirName, filename))
	return path, nil
}

// List reads approval YAML files newest-first (by ApprovedAt, then name).
func List(root string) ([]Record, error) {
	dir := Dir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("approval: list: %w", err)
	}

	var out []Record
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var rec Record
		if parseErr := yaml.Unmarshal(data, &rec); parseErr != nil {
			continue
		}
		if rec.ResourceID == "" {
			continue
		}
		rec.Path = filepath.ToSlash(filepath.Join(".specular", DirName, entry.Name()))
		out = append(out, rec)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].ApprovedAt.Equal(out[j].ApprovedAt) {
			return out[i].ApprovedAt.After(out[j].ApprovedAt)
		}
		return out[i].Path > out[j].Path
	})
	return out, nil
}

// FindByResourceID returns records matching resource_id (newest first).
func FindByResourceID(root, resourceID string) ([]Record, error) {
	all, err := List(root)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(resourceID)
	var out []Record
	for _, rec := range all {
		if rec.ResourceID == id {
			out = append(out, rec)
		}
	}
	return out, nil
}

// OpenExceptions returns non-expired exception records (newest first).
func OpenExceptions(root string, now time.Time) ([]Record, error) {
	all, err := List(root)
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var out []Record
	for _, rec := range all {
		if rec.Type != TypeException {
			continue
		}
		if rec.IsExpired(now) {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

// Recent returns up to limit newest records (limit <= 0 means all).
func Recent(root string, limit int) ([]Record, error) {
	all, err := List(root)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit >= len(all) {
		return all, nil
	}
	return all[:limit], nil
}

// NormalizeExceptionID ensures exception resources have an exception- prefix.
// Bare "exception" becomes exception-<timestamp>.
func NormalizeExceptionID(resourceID string, at time.Time) string {
	id := strings.TrimSpace(resourceID)
	if id == TypeException || id == "" {
		if at.IsZero() {
			at = time.Now().UTC()
		}
		return "exception-" + at.UTC().Format("20060102-150405")
	}
	if !strings.HasPrefix(id, "exception-") {
		return "exception-" + id
	}
	return id
}

// ParseExpires parses a duration (e.g. 7d, 24h) or RFC3339 timestamp.
func ParseExpires(raw string, now time.Time) (*time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if ts, err := time.Parse(time.RFC3339, s); err == nil {
		t := ts.UTC()
		return &t, nil
	}
	// Support Nd / Nh / Nm shorthand used in ops workflows.
	if len(s) >= 2 {
		unit := s[len(s)-1]
		num := s[:len(s)-1]
		var n int
		if _, err := fmt.Sscanf(num, "%d", &n); err == nil && n > 0 {
			var d time.Duration
			switch unit {
			case 'd', 'D':
				d = time.Duration(n) * 24 * time.Hour
			case 'h', 'H':
				d = time.Duration(n) * time.Hour
			case 'm', 'M':
				d = time.Duration(n) * time.Minute
			default:
				d = 0
			}
			if d > 0 {
				t := now.UTC().Add(d)
				return &t, nil
			}
		}
	}
	if d, err := time.ParseDuration(s); err == nil {
		t := now.UTC().Add(d)
		return &t, nil
	}
	return nil, fmt.Errorf("invalid --expires %q (use RFC3339 or duration like 7d, 24h)", raw)
}
