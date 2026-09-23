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
	// ClosedAt / ClosedBy / CloseReason record an early revoke of an open
	// exception (PRODUCT_INTENT §18). Soft-ALLOW only applies while IsOpen.
	ClosedAt    *time.Time `yaml:"closed_at,omitempty" json:"closed_at,omitempty"`
	ClosedBy    string     `yaml:"closed_by,omitempty" json:"closed_by,omitempty"`
	CloseReason string     `yaml:"close_reason,omitempty" json:"close_reason,omitempty"`

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

// IsClosed reports whether the exception was explicitly revoked early.
func (r *Record) IsClosed() bool {
	return r != nil && r.ClosedAt != nil && !r.ClosedAt.IsZero()
}

// IsOpen reports whether an exception is eligible for soft-ALLOW
// (exception type, not closed, not expired).
func (r *Record) IsOpen(now time.Time) bool {
	if r == nil || r.Type != TypeException {
		return false
	}
	if r.IsClosed() {
		return false
	}
	return !r.IsExpired(now)
}

// LifecycleStatus is open | closed | expired for list filtering.
const (
	StatusOpen    = "open"
	StatusClosed  = "closed"
	StatusExpired = "expired"
)

// Lifecycle returns open, closed, or expired for auditor filters.
// Closed wins over expired when both apply.
func (r *Record) Lifecycle(now time.Time) string {
	if r == nil {
		return StatusOpen
	}
	if r.IsClosed() {
		return StatusClosed
	}
	if r.IsExpired(now) {
		return StatusExpired
	}
	return StatusOpen
}

// FilterByStatus keeps records whose Lifecycle matches status
// (open|closed|expired). Empty status returns all.
func FilterByStatus(recs []Record, status string, now time.Time) ([]Record, error) {
	want := strings.ToLower(strings.TrimSpace(status))
	if want == "" {
		return recs, nil
	}
	switch want {
	case StatusOpen, StatusClosed, StatusExpired:
	default:
		return nil, fmt.Errorf("approval: invalid status %q (want open|closed|expired)", status)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var out []Record
	for _, rec := range recs {
		if rec.Lifecycle(now) == want {
			out = append(out, rec)
		}
	}
	return out, nil
}

// FilterByType keeps records whose Type matches (case-insensitive).
// Empty type returns all. Unknown types error.
func FilterByType(recs []Record, typ string) ([]Record, error) {
	want := strings.ToLower(strings.TrimSpace(typ))
	if want == "" {
		return recs, nil
	}
	switch want {
	case TypeBundle, TypeDrift, TypePolicy, TypePlan, TypeException:
	default:
		return nil, fmt.Errorf("approval: invalid type %q (want bundle|drift|policy|plan|exception)", typ)
	}
	var out []Record
	for _, rec := range recs {
		if strings.ToLower(strings.TrimSpace(rec.Type)) == want {
			out = append(out, rec)
		}
	}
	return out, nil
}

// FilterByPolicy keeps records whose Policy contains substr (case-insensitive).
// Empty substr returns all.
func FilterByPolicy(recs []Record, substr string) []Record {
	needle := strings.ToLower(strings.TrimSpace(substr))
	if needle == "" {
		return recs
	}
	var out []Record
	for _, rec := range recs {
		if strings.Contains(strings.ToLower(rec.Policy), needle) {
			out = append(out, rec)
		}
	}
	return out
}

// FilterByScope keeps records whose Scope contains substr (case-insensitive).
// Empty substr returns all.
func FilterByScope(recs []Record, substr string) []Record {
	needle := strings.ToLower(strings.TrimSpace(substr))
	if needle == "" {
		return recs
	}
	var out []Record
	for _, rec := range recs {
		if strings.Contains(strings.ToLower(rec.Scope), needle) {
			out = append(out, rec)
		}
	}
	return out
}

// FilterByEvidence keeps records whose EvidenceID equals or has prefix needle
// (case-insensitive), or whose ResourceID is in softResourceIDs (SoftAllow
// overrule join for Soft List jumps before/without bind). Empty needle returns all.
func FilterByEvidence(recs []Record, substr string, softResourceIDs ...string) []Record {
	needle := strings.ToLower(strings.TrimSpace(substr))
	if needle == "" {
		return recs
	}
	soft := make(map[string]struct{}, len(softResourceIDs))
	for _, id := range softResourceIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			soft[id] = struct{}{}
		}
	}
	var out []Record
	seen := make(map[string]struct{})
	for _, rec := range recs {
		id := strings.ToLower(strings.TrimSpace(rec.EvidenceID))
		match := id != "" && (id == needle || strings.HasPrefix(id, needle))
		if !match {
			_, match = soft[strings.TrimSpace(rec.ResourceID)]
		}
		if !match {
			continue
		}
		key := rec.Path
		if key == "" {
			key = rec.ResourceID
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, rec)
	}
	return out
}

// CloseOptions configures early revoke of an open exception.
type CloseOptions struct {
	Now    time.Time
	By     string
	Reason string
}

// Close early-ends an open exception by rewriting its YAML in place:
// stamps closed_* and clamps expires_at to now so soft-ALLOW stops.
// Idempotent when already closed or already expired.
func Close(root, resourceID string, opts CloseOptions) (*Record, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	id, by, resolveErr := resolveCloseIdentity(resourceID, opts.By, now)
	if resolveErr != nil {
		return nil, resolveErr
	}
	matches, err := FindByResourceID(root, id)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no approval record for %q", id)
	}
	target, pickErr := pickExceptionForClose(matches, now, id)
	if pickErr != nil {
		return nil, pickErr
	}
	if target.IsClosed() {
		return target, nil
	}
	applyCloseStamp(target, now, by, opts.Reason)
	absPath := target.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, filepath.FromSlash(target.Path))
	}
	if rewriteErr := rewrite(absPath, target); rewriteErr != nil {
		return nil, rewriteErr
	}
	return target, nil
}

func resolveCloseIdentity(resourceID, by string, now time.Time) (id, closer string, err error) {
	rawID := strings.TrimSpace(resourceID)
	if rawID == "" {
		return "", "", fmt.Errorf("approval close: resource_id is required")
	}
	if typ, typErr := TypeFromResourceID(rawID); typErr == nil && typ != TypeException {
		return "", "", fmt.Errorf("approval close: %q is not an exception record", rawID)
	}
	closer = strings.TrimSpace(by)
	if closer == "" {
		closer = "unknown"
	}
	return NormalizeExceptionID(rawID, now), closer, nil
}

func pickExceptionForClose(matches []Record, now time.Time, id string) (*Record, error) {
	var target *Record
	for i := range matches {
		if matches[i].Type != TypeException {
			continue
		}
		if matches[i].IsOpen(now) {
			target = &matches[i]
			break
		}
		if target == nil {
			target = &matches[i]
		}
	}
	if target == nil {
		return nil, fmt.Errorf("approval close: %q is not an exception record", id)
	}
	if strings.TrimSpace(target.Path) == "" {
		return nil, fmt.Errorf("approval close: record path missing for %q", id)
	}
	return target, nil
}

func applyCloseStamp(rec *Record, now time.Time, by, reason string) {
	closedAt := now
	rec.ClosedAt = &closedAt
	rec.ClosedBy = by
	if note := strings.TrimSpace(reason); note != "" {
		rec.CloseReason = note
	}
	if rec.ExpiresAt == nil || rec.ExpiresAt.After(now) {
		exp := now
		rec.ExpiresAt = &exp
	}
}

func rewrite(absPath string, rec *Record) error {
	if rec == nil {
		return fmt.Errorf("approval: nil record")
	}
	data, err := yaml.Marshal(rec)
	if err != nil {
		return fmt.Errorf("approval: marshal: %w", err)
	}
	if writeErr := os.WriteFile(absPath, data, 0o600); writeErr != nil {
		return fmt.Errorf("approval: rewrite: %w", writeErr)
	}
	return nil
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
	if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
		return "", fmt.Errorf("approval: write: %w", writeErr)
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

// OpenExceptions returns open (non-closed, non-expired) exception records
// newest first.
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
		if !rec.IsOpen(now) {
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
